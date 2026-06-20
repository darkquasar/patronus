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

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// Request is the input to Compute.
type Request struct {
	Catalog   *registry.Catalog
	Inventory *scan.Inventory
	Adapters  map[string]*manifest.Adapter // keyed by tool
	Resolver  toolpath.Resolver
	Names     []string // positional artifact names to install
	Tool      string   // "claude"|"codex"|"opencode"|"all"|"" (=> detected/all targeted)
	Scope     string   // "global"|"local"|"" (=> artifact default)
}

// toolRank orders tools by the DESIGN build order (claude → opencode → codex)
// for stable, intuitive output.
var toolRank = map[string]int{"claude": 0, "opencode": 1, "codex": 2}

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
			diffs, err := eng.Transform(art, ad, scope, entry.Source.LocalDir, readExisting)
			if err != nil {
				return nil, fmt.Errorf("%s -> %s: %w", name, tool, err)
			}
			raw = append(raw, diffs...)
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
	composed := composeByPath(raw)
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
func composeByPath(diffs []diff.FileDiff) []diff.FileDiff {
	order := []string{}
	byPath := map[string]*diff.FileDiff{}

	for _, d := range diffs {
		// FETCH/EXEC are per-recipe intents (a download, a command) that must
		// not be folded together even if their Path collides (EXEC rows carry no
		// real path). Pass each through under a unique key so all are preserved.
		if d.Action == diff.Fetch || d.Action == diff.Exec {
			key := string(d.Action) + "\x00" + d.Path + "\x00" + d.Note
			cp := d
			byPath[key] = &cp
			order = append(order, key)
			continue
		}
		prev, ok := byPath[d.Path]
		if !ok {
			cp := d
			byPath[d.Path] = &cp
			order = append(order, d.Path)
			continue
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
		case d.Action == diff.Merge && d.Setting != nil:
			// A settings list-append (hook registration) was computed against the
			// ORIGINAL file, so — unlike a scalar merge — it must be re-applied onto
			// the already-composed After, or a second hook into one settings.json
			// would silently drop the first. This is the MERGE-side twin of the
			// composed-APPEND fold: record a per-artifact contributor so remove can
			// strip exactly this element later.
			folded, err := adapter.ApplySettingEdit(prev.After, d.Setting)
			if err != nil {
				// A malformed re-fold is a planner bug, not user input; surface it by
				// keeping the standalone result rather than masking a half-merge.
				prev.After = d.After
			} else {
				prev.After = folded
			}
			if d.Artifact != "" && d.Artifact != prev.Artifact {
				prev.SettingContrib = append(prev.SettingContrib, diff.SettingContrib{
					Artifact: d.Artifact,
					Version:  d.Version,
					Edit:     d.Setting,
				})
			}
			prev.Tool = mergeTool(prev.Tool, d.Tool)
		case d.Action == diff.Merge:
			// Scalar merge (MCP, native-switch): already operates on accumulated
			// config; keep latest result.
			prev.After = d.After
			prev.Tool = mergeTool(prev.Tool, d.Tool)
		default:
			// CREATE (or append without section): keep the first, note sharing.
			prev.Tool = mergeTool(prev.Tool, d.Tool)
		}
	}

	out := make([]diff.FileDiff, 0, len(order))
	for _, p := range order {
		out = append(out, *byPath[p])
	}
	return out
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
		if d.Action == diff.Fetch || d.Action == diff.Exec {
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
