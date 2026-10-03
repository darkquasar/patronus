// Package plan computes an install change set without touching disk. It resolves
// which tools and scope an artifact targets, drives the adapter transform engine
// per (artifact × tool × scope), composes changes that land on the same path,
// and classifies each against the real filesystem into CREATE/APPEND/MERGE/
// CONFLICT/SKIP. The result is a diff.ChangeSet the renderer (dry-run) and the
// Phase-3 applier both consume.
package plan

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// Request is the input to Compute.
type Request struct {
	Catalog   *registry.Catalog
	Inventory *scan.Inventory
	Adapters  map[string]*manifest.Adapter // keyed by tool
	Resolver  toolpath.Resolver
	Names     []string // positional artifact names to install
	Tool      string   // "claude"|"codex"|"opencode"|"pi"|"all"|"" (=> detected/all targeted)
	Scope     string   // "global"|"local"|"" (=> artifact default)
}

// toolRank orders tools by the DESIGN build order (claude → opencode → codex → pi)
// for stable, intuitive output.
var toolRank = map[string]int{"claude": 0, "opencode": 1, "codex": 2, "pi": 3}

// Compute resolves req into a classified change set. It performs read-only
// filesystem access (to read existing targets and stat for classification).
func Compute(req Request) (*diff.ChangeSet, error) {
	eng := adapter.New(req.Resolver)
	readExisting := func(path string) ([]byte, bool, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		return b, true, nil
	}

	var raw []diff.FileDiff
	for _, name := range req.Names {
		entry, err := findArtifact(req.Catalog, name)
		if err != nil {
			return nil, err
		}
		art := entry.Manifest

		scope := resolveScope(req.Scope, art)
		tools, err := resolveTools(req.Tool, art, req.Inventory, scope)
		if err != nil {
			return nil, err
		}

		for _, tool := range tools {
			ad, ok := req.Adapters[tool]
			if !ok {
				return nil, fmt.Errorf("no adapter for tool %q", tool)
			}
			read := adapter.ReadExisting(readExisting)
			if tool == "pi" {
				read = adapter.ReadExisting(scan.ReadPiFile)
			}
			diffs, err := eng.Transform(art, ad, scope, entry.Source.LocalDir, read)
			if err != nil {
				return nil, fmt.Errorf("%s -> %s: %w", name, tool, err)
			}
			raw = append(raw, diffs...)
		}
	}

	// Native named skills must be in this selected catalog closure, not merely
	// happen to exist in a user's runtime environment.
	for _, d := range raw {
		if d.Tool != "pi" || d.Type != string(manifest.TypeAgent) {
			continue
		}
		skills, err := adapter.ValidatePiAgent(d.Artifact, d.After)
		if err != nil {
			return nil, err
		}
		for _, skill := range skills {
			found := false
			for _, name := range req.Names {
				entry, err := findArtifact(req.Catalog, name)
				if err == nil && entry.Manifest.Name == skill && entry.Manifest.Type == manifest.TypeSkill {
					found = true
				}
			}
			if !found {
				return nil, fmt.Errorf("pi agent %q field skills: %q is absent from selected skill closure", d.Artifact, skill)
			}
		}
	}
	return Finalize(raw, readExisting)
}

// Finalize is the shared tail of the change-set spine: it composes diffs that
// land on the same path, classifies each against the real filesystem, and sorts
// the result deterministically. Both producers feed it — plan.Compute (artifacts)
// and recipe.Compute (recipes) — so artifact and recipe installs converge on one
// classified diff.ChangeSet rather than two parallel paths (the brief's
// "one spine" requirement). read supplies current target bytes for classification.
func Finalize(raw []diff.FileDiff, read adapter.ReadExisting) (*diff.ChangeSet, error) {
	for i, d := range raw {
		if d.Tool != "pi" {
			continue
		}
		if err := scan.PiSafePath(d.Path); err != nil {
			return nil, err
		}
		for _, other := range raw[:i] {
			if other.Path == d.Path && (other.Tool != "pi" || (d.Action == diff.Create && other.Artifact != d.Artifact)) {
				return nil, fmt.Errorf("pi destination conflict %s: %s (%s) and %s (%s)", d.Path, other.Artifact, other.Tool, d.Artifact, d.Tool)
			}
		}
	}
	composed, err := composeByPath(raw)
	if err != nil {
		return nil, err
	}
	classified, err := classify(composed, read)
	if err != nil {
		return nil, err
	}
	sortDiffs(classified)
	return &diff.ChangeSet{Diffs: classified, DryRun: true}, nil
}

// findArtifact looks up an artifact by name in the catalog.
func findArtifact(cat *registry.Catalog, name string) (*registry.ArtifactEntry, error) {
	for i := range cat.Artifacts {
		if cat.Artifacts[i].Manifest.Name == name {
			return &cat.Artifacts[i], nil
		}
	}
	return nil, fmt.Errorf("unknown artifact %q", name)
}

// resolveScope picks the install scope: explicit flag wins, else the artifact
// default ("project" is normalized to "local").
func resolveScope(flag string, art *manifest.Artifact) string {
	s := flag
	if s == "" {
		s = art.Defaults.Scope
	}
	if s == "project" {
		s = "local"
	}
	if s == "" {
		s = "local"
	}
	return s
}

// resolveTools picks which tools to plan for. A specific --tool must be one the
// artifact targets. "all"/empty expands to the artifact's targets that are
// detected at the chosen scope; if none are detected, it falls back to all
// targeted tools (so a dry-run is still useful on a clean machine).
func resolveTools(flag string, art *manifest.Artifact, inv *scan.Inventory, scope string) ([]string, error) {
	targeted := map[string]bool{}
	for _, t := range art.Targets {
		targeted[t] = true
	}

	if flag != "" && flag != "all" {
		if !targeted[flag] {
			return nil, fmt.Errorf("artifact %q does not target tool %q (targets: %v)", art.Name, flag, art.Targets)
		}
		return []string{flag}, nil
	}

	var detected []string
	for _, t := range art.Targets {
		if isDetected(inv, t, scope) {
			detected = append(detected, t)
		}
	}
	if len(detected) > 0 {
		return sortByRank(detected), nil
	}
	return sortByRank(art.Targets), nil
}

// isDetected reports whether a tool was detected at the given scope.
func isDetected(inv *scan.Inventory, tool, scope string) bool {
	if inv == nil {
		return false
	}
	for _, ts := range inv.Tools {
		if ts.Tool != tool {
			continue
		}
		d := ts.Global
		if scope == "local" {
			d = ts.Local
		}
		return d != nil && d.Detected
	}
	return false
}

func sortByRank(tools []string) []string {
	out := append([]string(nil), tools...)
	sort.SliceStable(out, func(i, j int) bool { return toolRank[out[i]] < toolRank[out[j]] })
	return out
}

// composeByPath folds multiple diffs that target the same absolute path into a
// single diff. CREATE collisions keep the first (identical content is the common
// case across SKILL.md-native tools); APPEND/MERGE accumulate so e.g. codex and
// opencode both appending to a shared project AGENTS.md produce one combined
// result rather than two competing writes.
func composeByPath(diffs []diff.FileDiff) ([]diff.FileDiff, error) {
	order := []string{}
	byPath := map[string]*diff.FileDiff{}
	settings := map[string][]diff.FileDiff{}

	for _, d := range diffs {
		// FETCH/EXEC are per-recipe intents (a download, a command) that must
		// not be folded together even if their Path collides (EXEC rows carry no
		// real path). Pass each through under a unique key so all are preserved.
		if d.Action == diff.Fetch || d.Action == diff.Exec || d.Native != nil {
			key := string(d.Action) + "\x00" + d.Path + "\x00" + d.Note
			cp := d
			byPath[key] = &cp
			order = append(order, key)
			continue
		}
		if d.Action == diff.Merge && d.Setting == nil {
			return nil, fmt.Errorf("compose %s: merge missing structural evidence", d.Path)
		}
		// Check every leaf, including contributors from an earlier Finalize.
		rows := settingRows(d)
		duplicates := 0
		for _, row := range rows {
			// SettingContrib inherits its enclosing diff's tool/scope. Until it
			// carries independent identities, refuse heterogeneous composition
			// before a fold can erase the original owner (even for disjoint keys).
			if strings.Contains(row.Tool, "+") {
				return nil, fmt.Errorf("compose %s: ambiguous structural tool identity %q", d.Path, row.Tool)
			}
			duplicate := false
			if err := adapter.ValidateSettingEdit(row.Setting); err != nil {
				return nil, fmt.Errorf("compose %s: %w", d.Path, err)
			}
			if _, _, err := adapter.SettingStatus(row.Before, row.Setting); err != nil {
				return nil, fmt.Errorf("compose %s prior: %w", d.Path, err)
			}
			for _, old := range settings[d.Path] {
				if old.Tool != row.Tool || old.Scope != row.Scope {
					return nil, fmt.Errorf("compose %s: structural identity conflict between %s/%s and %s/%s", d.Path, old.Tool, old.Scope, row.Tool, row.Scope)
				}
				if err := adapter.CheckSettingPair(old.Setting, settingOwner(old), row.Setting, settingOwner(row)); err != nil {
					return nil, fmt.Errorf("compose %s: %w", d.Path, err)
				}
				if adapter.SettingEditsOverlap(old.Setting, row.Setting) {
					duplicate = true
				}
			}
			if duplicate {
				duplicates++
			}
			settings[d.Path] = append(settings[d.Path], row)
		}
		if duplicates > 0 {
			if duplicates == len(rows) {
				continue
			}
			return nil, fmt.Errorf("compose %s: partially overlapping composite ownership", d.Path)
		}
		prev, ok := byPath[d.Path]
		if !ok {
			cp := d
			cp.SettingContrib = append([]diff.SettingContrib(nil), d.SettingContrib...)
			cp.Contrib = append([]diff.SectionContrib(nil), d.Contrib...)
			byPath[d.Path] = &cp
			order = append(order, d.Path)
			continue
		}
		if (prev.Setting == nil) != (d.Setting == nil) {
			return nil, fmt.Errorf("compose %s: structural and nonstructural writes overlap", d.Path)
		}
		switch {
		case d.Action == diff.Append && d.Section != nil:
			// Re-fold this section onto the already-composed After so multiple
			// appends to the same file accumulate. When the contribution comes from
			// a DIFFERENT artifact (not just the same artifact re-appending for
			// another tool), record it so state can track + remove each section
			// under its own artifact. Prior is the composed bytes before this fold,
			// so remove reverses contributions in order.
			if d.Artifact != "" && d.Artifact != prev.Artifact {
				prev.Contrib = append(prev.Contrib, diff.SectionContrib{
					Artifact: d.Artifact,
					Version:  d.Version,
					Section:  d.Section.Name,
					Prior:    prev.After,
				})
			}
			prev.After = adapter.AppendSection(prev.After, d.Section.Name, d.Section.Body)
			prev.Tool = mergeTool(prev.Tool, d.Tool)
		case d.Setting != nil:
			// Every settings MERGE was computed against the ORIGINAL file, so it must
			// be re-applied onto the already-composed After or a second edit into one
			// file silently drops the first. This covers hook registrations, scalar
			// toggles, permission gates, and MCP server blocks alike: all of them are
			// SettingEdits, and all of them compose. This is the MERGE-side twin of
			// the composed-APPEND fold: record a per-artifact contributor so remove
			// can strip exactly this element later.
			if prev.Setting == nil {
				return nil, fmt.Errorf("compose %s: structural and nonstructural writes overlap", d.Path)
			}
			folded := prev.After
			var err error
			for _, row := range settingRows(d) {
				folded, err = adapter.ApplySettingEdit(folded, row.Setting)
				if err != nil {
					break
				}
			}
			if err != nil {
				return nil, fmt.Errorf("compose %s (%s): %w", d.Path, d.Artifact, err)
			}
			prev.After = folded
			// Record a per-edit contributor for every folded setting so state/remove
			// can strip each element. This covers two shapes: distinct artifacts
			// merging into one settings.json (the composed-hook case), AND a single
			// artifact emitting several edits to one file — an opencode gate whose
			// matcher maps to more than one permission key (Edit|Bash → permission.edit
			// + permission.bash). Only the FIRST diff for the path is the owning
			// prev (its edit rides fileState); every subsequent fold needs a contrib
			// or its deny would leak on remove.
			if d.Artifact != "" {
				prev.SettingContrib = append(prev.SettingContrib, diff.SettingContrib{
					Artifact: d.Artifact,
					Version:  d.Version,
					Edit:     d.Setting,
					// Carry the contributor's OWN type/role so the plan can render an
					// honest row for it instead of inheriting the owning diff's.
					Type: d.Type,
					Role: d.Role,
				})
			}
			prev.SettingContrib = append(prev.SettingContrib, d.SettingContrib...)
			prev.Tool = mergeTool(prev.Tool, d.Tool)
		default:
			if prev.Setting != nil {
				return nil, fmt.Errorf("compose %s: structural and nonstructural writes overlap", d.Path)
			}
			// Nonstructural CREATE or append-without-section sharing retains the
			// existing first-result semantics. MERGE evidence was checked above.
			prev.Tool = mergeTool(prev.Tool, d.Tool)
		}
	}

	out := make([]diff.FileDiff, 0, len(order))
	for _, p := range order {
		d := *byPath[p]
		for _, row := range settingRows(d) {
			present, equal, err := adapter.SettingStatus(d.After, row.Setting)
			if err != nil {
				return nil, fmt.Errorf("compose %s result: %w", d.Path, err)
			}
			if !present || !equal {
				return nil, fmt.Errorf("compose %s: result lost setting %s", d.Path, row.Setting.Dotted)
			}
		}
		out = append(out, d)
	}
	return out, nil
}

// mergeTool combines tool labels for a shared path. Identical tools collapse;
// distinct tools are joined so the renderer can show "claude+opencode".
func mergeTool(a, b string) string {
	if a == "" || a == b {
		return b
	}
	if b == "" {
		return a
	}
	for _, t := range splitTools(a) {
		if t == b {
			return a
		}
	}
	return a + "+" + b
}

func splitTools(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '+' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// classify stats each composed diff's path and assigns the terminal action.
func classify(diffs []diff.FileDiff, read adapter.ReadExisting) ([]diff.FileDiff, error) {
	for i := range diffs {
		d := &diffs[i]
		// FETCH and EXEC are not file-content edits: FETCH is pre-classified by
		// the recipe engine (sha-vs-disk), and EXEC is a display-only command
		// row. Neither compares Before/After bytes, so skip the fs read here.
		if d.Action == diff.Fetch || d.Action == diff.Exec || d.Native != nil {
			continue
		}
		before, exists, err := read(d.Path)
		if err != nil {
			return nil, err
		}
		// For CREATE the engine didn't set Before; populate it for accurate
		// classification and for the applier/renderer.
		if d.Before == nil {
			d.Before = before
		}
		d.Action = diff.Classify(d.Action, d.Before, d.After, exists)
	}
	return diffs, nil
}

// sortDiffs orders the change set deterministically: tool rank, then scope, then
// path.
func sortDiffs(diffs []diff.FileDiff) {
	sort.SliceStable(diffs, func(i, j int) bool {
		a, b := diffs[i], diffs[j]
		if ra, rb := toolRank[a.Tool], toolRank[b.Tool]; ra != rb {
			return ra < rb
		}
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		return a.Path < b.Path
	})
}

// settingRows retains the owner of every leaf in an already-composed diff.
func settingRows(d diff.FileDiff) []diff.FileDiff {
	if d.Setting == nil {
		return nil
	}
	first := d
	first.SettingContrib = nil
	rows := []diff.FileDiff{first}
	for _, c := range d.SettingContrib {
		row := first
		row.Artifact, row.Version, row.Type, row.Role, row.Setting = c.Artifact, c.Version, c.Type, c.Role, c.Edit
		rows = append(rows, row)
	}
	return rows
}

func settingOwner(d diff.FileDiff) adapter.SettingOwner {
	return adapter.SettingOwner{Artifact: d.Artifact, Tool: d.Tool, Scope: d.Scope}
}

// AdmitSettings checks installed evidence supplied by the command boundary. It
// does no I/O and never turns byte equality into ownership. Both scopes must be
// supplied by the caller because recorded absolute paths can cross scopes.
func AdmitSettings(cs *diff.ChangeSet, owners []state.Item) (*diff.ChangeSet, error) {
	var raw, noops []diff.FileDiff
	for _, d := range cs.Diffs {
		rows := settingRows(d)
		if len(rows) == 0 {
			raw = append(raw, d)
			continue
		}
		for _, row := range rows {
			owned := false
			for _, it := range owners {
				if it.Tool == "pi" && it.Root != "" && row.Root != "" && it.Root != row.Root {
					continue
				}
				owner := adapter.SettingOwner{Artifact: it.Artifact, Tool: it.Tool, Scope: it.Scope}
				for _, file := range it.Files {
					if file.Action != string(diff.Merge) {
						continue
					}
					if owner == settingOwner(row) && (file.Path != row.Path || !adapter.SameSettingTarget(file.Setting, row.Setting)) {
						// Multiple disjoint leaves of one item are legitimate; a dropped or
						// relocated recorded leaf is not evidence for a new inverse baseline.
						retained := false
						for _, candidate := range cs.Diffs {
							for _, leaf := range settingRows(candidate) {
								if settingOwner(leaf) == owner && leaf.Path == file.Path && adapter.SameSettingTarget(leaf.Setting, file.Setting) {
									retained = true
								}
							}
						}
						if !retained {
							return nil, fmt.Errorf("setting %s: recorded path changed for %s; explicit migration required", row.Path, it.Artifact)
						}
					}
					if file.Path != row.Path {
						continue
					}
					if err := adapter.ValidateSettingEdit(file.Setting); err != nil {
						return nil, fmt.Errorf("setting %s owner %s: %w", row.Path, it.Artifact, err)
					}
					if !adapter.SettingEditsOverlap(file.Setting, row.Setting) {
						if err := adapter.CheckSettingPair(file.Setting, owner, row.Setting, settingOwner(row)); err != nil {
							return nil, fmt.Errorf("setting %s: %w", row.Path, err)
						}
						continue
					}
					if owner != settingOwner(row) || !adapter.SameSettingTarget(file.Setting, row.Setting) || owned {
						return nil, fmt.Errorf("setting ownership conflict at %s: %s (%s/%s) overlaps %s (%s/%s); explicit migration required", row.Path, it.Artifact, it.Tool, it.Scope, row.Artifact, row.Tool, row.Scope)
					}
					present, equal, err := adapter.SettingStatus(row.Before, file.Setting)
					if err != nil {
						return nil, fmt.Errorf("setting %s: %w", row.Path, err)
					}
					if !present || !equal {
						return nil, fmt.Errorf("setting %s: recorded leaf %s changed or missing", row.Path, file.Setting.Dotted)
					}
					owned = true
				}
			}
			present, equal, err := adapter.SettingStatus(row.Before, row.Setting)
			if err != nil {
				return nil, fmt.Errorf("setting %s: %w", row.Path, err)
			}
			if !owned && present {
				if !equal {
					return nil, fmt.Errorf("setting %s at %s differs from unmanaged contribution; explicit migration required", row.Setting.Dotted, row.Path)
				}
				// Preserve the intent for repeat admission, but keep it detached
				// from writable contributors. SKIP grants no ownership; erasing
				// Setting would misclassify this row as a nonstructural write.
				row.Action, row.After = diff.Skip, row.Before
				row.Note = "equal external setting — unmanaged; no removal authority"
				noops = append(noops, row)
				continue
			}
			row.Action = diff.Merge
			row.After, err = adapter.ApplySettingEdit(row.Before, row.Setting)
			if err != nil {
				return nil, fmt.Errorf("setting %s: %w", row.Path, err)
			}
			raw = append(raw, row)
		}
	}
	composed, err := composeByPath(raw)
	if err != nil {
		return nil, err
	}
	for i := range composed {
		d := &composed[i]
		if d.Setting != nil {
			d.Action = diff.Classify(diff.Merge, d.Before, d.After, len(d.Before) > 0)
		}
	}
	composed = append(composed, noops...)
	sortDiffs(composed)
	return &diff.ChangeSet{Diffs: composed, DryRun: cs.DryRun}, nil
}
