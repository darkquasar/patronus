package remove

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/state"
)

// readerFrom builds a ReadExisting over an in-memory file map.
func readerFrom(files map[string][]byte) ReadExisting {
	return func(path string) ([]byte, bool, error) {
		b, ok := files[path]
		return b, ok, nil
	}
}

// computeForTest keeps the pre-composition (changeSet, warnings, error) shape the
// older cases were written against, so they keep asserting the same behavior
// without restating the ledger they do not exercise. Occupancy is nil: these
// cases each remove a single artifact, which is the sole-contributor case.
func computeForTest(items []state.Item, read ReadExisting) (*diff.ChangeSet, []Warning, error) {
	r, err := Compute(items, read, nil)
	if err != nil {
		return nil, nil, err
	}
	return r.ChangeSet, r.Warnings, nil
}

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}

func TestCreateBecomesDelete(t *testing.T) {
	body := []byte("SKILL body")
	items := []state.Item{{
		Artifact: "s", Tool: "claude", Scope: "global",
		Files: []state.FileState{{Path: "/c/SKILL.md", Action: "CREATE", Checksum: sum(body)}},
	}}
	cs, warns, err := computeForTest(items, readerFrom(map[string][]byte{"/c/SKILL.md": body}))
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %+v", warns)
	}
	if len(cs.Diffs) != 1 || cs.Diffs[0].Action != diff.Delete {
		t.Fatalf("want one DELETE, got %+v", cs.Diffs)
	}
	if !bytes.Equal(cs.Diffs[0].Before, body) {
		t.Error("DELETE should carry current bytes as Before for the diff view")
	}
}

func TestCreateAlreadyAbsentSkips(t *testing.T) {
	items := []state.Item{{
		Artifact: "s", Tool: "claude", Scope: "global",
		Files: []state.FileState{{Path: "/c/SKILL.md", Action: "CREATE", Checksum: sum([]byte("x"))}},
	}}
	cs, _, err := computeForTest(items, readerFrom(map[string][]byte{}))
	if err != nil {
		t.Fatal(err)
	}
	if cs.Diffs[0].Action != diff.Skip {
		t.Errorf("absent file should SKIP, got %s", cs.Diffs[0].Action)
	}
}

func TestCreateDriftSkipsWithIntent(t *testing.T) {
	items := []state.Item{{
		Artifact: "s", Tool: "claude", Scope: "global",
		Files: []state.FileState{{Path: "/c/SKILL.md", Action: "CREATE", Checksum: sum([]byte("original"))}},
	}}
	cs, warns, err := computeForTest(items, readerFrom(map[string][]byte{"/c/SKILL.md": []byte("USER EDITED")}))
	if err != nil {
		t.Fatal(err)
	}
	if cs.Diffs[0].Action != diff.Skip || cs.Diffs[0].Intended != diff.Delete {
		t.Fatalf("drift should be SKIP intending DELETE, got action=%s intended=%s", cs.Diffs[0].Action, cs.Diffs[0].Intended)
	}
	if len(warns) != 1 {
		t.Errorf("want one drift warning, got %d", len(warns))
	}
	// --force promotes it.
	Promote(Result{ChangeSet: cs})
	if cs.Diffs[0].Action != diff.Delete {
		t.Errorf("Promote should turn drift SKIP into DELETE, got %s", cs.Diffs[0].Action)
	}
}

func TestAppendBecomesUnappend(t *testing.T) {
	prior := []byte("# Notes\n\nuser prose\n")
	installed := adapter.AppendSection(prior, "ap", []byte("injected"))
	items := []state.Item{{
		Artifact: "ap", Tool: "claude", Scope: "local",
		Files: []state.FileState{{Path: "/p/CLAUDE.md", Action: "APPEND", Section: "ap", Prior: prior, Checksum: sum(installed)}},
	}}
	cs, warns, err := computeForTest(items, readerFrom(map[string][]byte{"/p/CLAUDE.md": installed}))
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %+v", warns)
	}
	d := cs.Diffs[0]
	if d.Action != diff.Unappend {
		t.Fatalf("want UNAPPEND, got %s", d.Action)
	}
	if !bytes.Equal(d.After, prior) {
		t.Errorf("un-append should restore the prior prose:\n got %q\nwant %q", d.After, prior)
	}
}

func TestAppendSectionAlreadyGoneRetiresRow(t *testing.T) {
	// The recorded APPEND section is ALREADY ABSENT from the file — e.g. a later
	// rebuild dropped it (the `beads` case: beads→ticket migration removed the
	// section, but its state row survived). The file work is already done, so remove
	// must UNAPPEND-as-no-op (After == current) — landing in Applied, which is what
	// retires the orphaned state row — NOT SKIP, which would strand the row forever
	// and keep `scan` reporting MISSING with no way to clean it up.
	other := adapter.AppendSection([]byte("# Notes\n"), "keep", []byte("a different section"))
	items := []state.Item{{
		Artifact: "gone", Tool: "claude", Scope: "local",
		Files: []state.FileState{{Path: "/p/CLAUDE.md", Action: "APPEND", Section: "gone", Prior: nil, Checksum: sum([]byte("whatever was recorded"))}},
	}}
	cs, warns, err := computeForTest(items, readerFrom(map[string][]byte{"/p/CLAUDE.md": other}))
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Errorf("an already-absent section is not drift; want no warnings, got %+v", warns)
	}
	d := cs.Diffs[0]
	if d.Action != diff.Unappend {
		t.Fatalf("already-absent section must UNAPPEND (no-op) so the row is retired, got %s", d.Action)
	}
	if !bytes.Equal(d.After, other) {
		t.Errorf("the no-op un-append must leave the file byte-identical (the OTHER section survives):\n got %q\nwant %q", d.After, other)
	}
}

func TestAppendDriftSkipsWithIntent(t *testing.T) {
	prior := []byte("# Notes\n\nuser prose\n")
	installed := adapter.AppendSection(prior, "ap", []byte("injected"))
	// The user appended their OWN text after install → un-appending our section
	// no longer yields exactly prior.
	edited := append(append([]byte{}, installed...), []byte("\nuser added a line\n")...)
	items := []state.Item{{
		Artifact: "ap", Tool: "claude", Scope: "local",
		Files: []state.FileState{{Path: "/p/CLAUDE.md", Action: "APPEND", Section: "ap", Prior: prior, Checksum: sum(installed)}},
	}}
	cs, warns, err := computeForTest(items, readerFrom(map[string][]byte{"/p/CLAUDE.md": edited}))
	if err != nil {
		t.Fatal(err)
	}
	if cs.Diffs[0].Action != diff.Skip || cs.Diffs[0].Intended != diff.Unappend {
		t.Fatalf("drift should be SKIP intending UNAPPEND, got %s / %s", cs.Diffs[0].Action, cs.Diffs[0].Intended)
	}
	if len(warns) != 1 {
		t.Errorf("want one drift warning, got %d", len(warns))
	}
}

func TestAppendIntoFreshFileDriftSkips(t *testing.T) {
	// Installed into a file that did NOT exist before → Prior is nil. The user then
	// added their own text outside our fenced section. Un-appending would silently
	// drop that text, so it must be detected as drift and skipped.
	installed := adapter.AppendSection(nil, "ap", []byte("patronus body"))
	edited := append(append([]byte{}, installed...), []byte("\n## the user's own heading\n")...)
	items := []state.Item{{
		Artifact: "ap", Tool: "claude", Scope: "local",
		Files: []state.FileState{{Path: "/p/CLAUDE.md", Action: "APPEND", Section: "ap", Prior: nil, Checksum: sum(installed)}},
	}}
	cs, warns, err := computeForTest(items, readerFrom(map[string][]byte{"/p/CLAUDE.md": edited}))
	if err != nil {
		t.Fatal(err)
	}
	if cs.Diffs[0].Action != diff.Skip || cs.Diffs[0].Intended != diff.Unappend {
		t.Fatalf("fresh-file drift should be SKIP intending UNAPPEND, got %s / %s", cs.Diffs[0].Action, cs.Diffs[0].Intended)
	}
	if len(warns) != 1 {
		t.Errorf("want one drift warning, got %d", len(warns))
	}
}

func TestAppendIntoFreshFileCleanUnappend(t *testing.T) {
	// Installed into a fresh file, NOT edited since → clean UNAPPEND (no drift).
	installed := adapter.AppendSection(nil, "ap", []byte("patronus body"))
	items := []state.Item{{
		Artifact: "ap", Tool: "claude", Scope: "local",
		Files: []state.FileState{{Path: "/p/CLAUDE.md", Action: "APPEND", Section: "ap", Prior: nil, Checksum: sum(installed)}},
	}}
	cs, warns, err := computeForTest(items, readerFrom(map[string][]byte{"/p/CLAUDE.md": installed}))
	if err != nil {
		t.Fatal(err)
	}
	if cs.Diffs[0].Action != diff.Unappend {
		t.Fatalf("clean fresh-file remove should UNAPPEND, got %s", cs.Diffs[0].Action)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %+v", warns)
	}
}

func TestMergeBecomesRestore(t *testing.T) {
	prior := []byte("{}")
	installed := []byte(`{"mcpServers":{"x":{}}}`)
	items := []state.Item{{
		Artifact: "mem", Tool: "claude", Scope: "local",
		Files: []state.FileState{{Path: "/p/.mcp.json", Action: "MERGE", Prior: prior, Checksum: sum(installed)}},
	}}
	cs, _, err := computeForTest(items, readerFrom(map[string][]byte{"/p/.mcp.json": installed}))
	if err != nil {
		t.Fatal(err)
	}
	d := cs.Diffs[0]
	if d.Action != diff.Restore || !bytes.Equal(d.After, prior) {
		t.Fatalf("want RESTORE to prior bytes, got action=%s after=%q", d.Action, d.After)
	}
}

// A hook MERGE reverts by stripping EXACTLY its array element, leaving sibling
// hooks (another artifact's and the user's) intact — the targeted-remove twin of
// APPEND's surgical un-section.
func TestHookMergeStripsOneElement(t *testing.T) {
	ft := manifest.FileTarget{File: "settings.json", Format: "json"}
	dotted := "hooks.PreToolUse"

	// A user hook the install must never touch (no patronusId, seeded via the same
	// list-append so it occupies the array honestly).
	userHook := map[string]any{"matcher": "Bash", "hooks": []any{map[string]any{"type": "command", "command": "user"}}}
	base, err := adapter.AppendSettingsList(nil, ft, dotted, "patronusId", userHook)
	if err != nil {
		t.Fatal(err)
	}
	// Two patronus hooks fold in on top of the user's.
	elemA := map[string]any{"patronusId": "A", "matcher": "Edit", "hooks": []any{map[string]any{"type": "command", "command": "tdd"}}}
	elemB := map[string]any{"patronusId": "B", "matcher": "Write", "hooks": []any{map[string]any{"type": "command", "command": "leaks"}}}
	withA, err := adapter.AppendSettingsList(base, ft, dotted, "patronusId", elemA)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := adapter.AppendSettingsList(withA, ft, dotted, "patronusId", elemB)
	if err != nil {
		t.Fatal(err)
	}

	editA := &diff.SettingEdit{
		Target: diff.FileTargetRef{File: ft.File, Format: ft.Format}, Dotted: dotted,
		IdentityKey: "patronusId", Identity: "A", Elem: elemA,
	}
	items := []state.Item{{
		Artifact: "tdd-guard", Tool: "claude", Scope: "global",
		Files: []state.FileState{{Path: "/p/settings.json", Action: "MERGE", Setting: editA, Checksum: sum(installed)}},
	}}
	cs, warns, err := computeForTest(items, readerFrom(map[string][]byte{"/p/settings.json": installed}))
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %+v", warns)
	}
	d := cs.Diffs[0]
	if d.Action != diff.Restore {
		t.Fatalf("want RESTORE (element-stripped bytes), got %s", d.Action)
	}
	// A is gone; B and the user hook remain.
	if bytes.Contains(d.After, []byte(`"A"`)) {
		t.Errorf("removed element A still present:\n%s", d.After)
	}
	for _, want := range []string{`"B"`, `"user"`} {
		if !bytes.Contains(d.After, []byte(want)) {
			t.Errorf("sibling %s was clobbered:\n%s", want, d.After)
		}
	}
}

// mcpItem builds the state one MCP recipe records for a shared config: a MERGE
// row whose SettingEdit names exactly its own server key.
func mcpItem(artifact, path, name string, installed []byte, prior any, priorPresent bool) state.Item {
	return state.Item{
		Artifact: artifact, Tool: "claude", Scope: "global",
		Files: []state.FileState{{
			Path: path, Action: string(diff.Merge), Checksum: sum(installed),
			Setting: &diff.SettingEdit{
				Target:       diff.FileTargetRef{File: ".claude.json", Format: "json"},
				Dotted:       "mcpServers." + name,
				PriorValue:   prior,
				PriorPresent: priorPresent,
			},
		}},
	}
}

func assertServers(t *testing.T, b []byte, present, absent []string) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse result: %v (%s)", err, b)
	}
	got, _ := m["mcpServers"].(map[string]any)
	for _, name := range present {
		if _, ok := got[name]; !ok {
			t.Errorf("sibling %q was clobbered:\n%s", name, b)
		}
	}
	for _, name := range absent {
		if _, ok := got[name]; ok {
			t.Errorf("%q should have been removed:\n%s", name, b)
		}
	}
}

func TestRemoveMcpContributorLeavesSiblings(t *testing.T) {
	// graphify installed first (the owner), serena second (a contributor).
	// Removing serena must leave graphify and the user's context7 alone.
	const path = "/p/.claude.json"
	installed := []byte(`{"mcpServers":{"context7":{"command":"c7"},"graphify":{"command":"gq"},"serena":{"command":"uvx"}}}`)

	cs, warns, err := computeForTest(
		[]state.Item{mcpItem("serena", path, "serena", installed, nil, false)},
		readerFrom(map[string][]byte{path: installed}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %+v", warns)
	}
	d := cs.Diffs[0]
	if d.Action != diff.Restore {
		t.Fatalf("want RESTORE (surgically edited bytes), got %s", d.Action)
	}
	assertServers(t, d.After, []string{"context7", "graphify"}, []string{"serena"})
}

func TestRemoveMcpOwnerLeavesSiblings(t *testing.T) {
	// The FIRST-installed recipe is the owning `prev`, so its edit rides FileState
	// rather than SettingContrib. It must still remove surgically.
	const path = "/p/.claude.json"
	installed := []byte(`{"mcpServers":{"context7":{"command":"c7"},"graphify":{"command":"gq"},"serena":{"command":"uvx"}}}`)

	cs, _, err := computeForTest(
		[]state.Item{mcpItem("graphify", path, "graphify", installed, nil, false)},
		readerFrom(map[string][]byte{path: installed}),
	)
	if err != nil {
		t.Fatal(err)
	}
	assertServers(t, cs.Diffs[0].After, []string{"context7", "serena"}, []string{"graphify"})
}

func TestRemoveMcpRestoresUserPrior(t *testing.T) {
	// The user had their own serena block before install; remove puts it back
	// rather than deleting the key.
	const path = "/p/.claude.json"
	installed := []byte(`{"mcpServers":{"serena":{"command":"uvx"}}}`)
	prior := map[string]any{"command": "my-own"}

	cs, _, err := computeForTest(
		[]state.Item{mcpItem("serena", path, "serena", installed, prior, true)},
		readerFrom(map[string][]byte{path: installed}),
	)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal(cs.Diffs[0].After, &m); err != nil {
		t.Fatal(err)
	}
	blk, ok := m["mcpServers"].(map[string]any)["serena"].(map[string]any)
	if !ok {
		t.Fatalf("serena was deleted, not restored:\n%s", cs.Diffs[0].After)
	}
	if blk["command"] != "my-own" {
		t.Fatalf("serena.command = %v, want my-own", blk["command"])
	}
}

func TestSelfWiredWarnsAndSkips(t *testing.T) {
	items := []state.Item{{
		Artifact: "ai-memory", Tool: "claude", Scope: "global",
		SelfWired: true, PostInstall: []string{"docker run ..."},
	}}
	_, warns, err := computeForTest(items, readerFrom(map[string][]byte{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 1 {
		t.Fatalf("want one self-wired warning, got %d: %+v", len(warns), warns)
	}
}

// --- composition (pat-1cnz) --------------------------------------------------

// The reproduction the ticket was filed on: two MCP recipes wired into ONE
// config, removed in ONE command. Computed independently, each undo was correct
// in isolation and wrong in sequence — the second write, built from the original
// bytes, put back the server the first had just removed. The artifact the user
// asked to remove SURVIVED the removal.
func TestTwoContributorsOnOnePathCompose(t *testing.T) {
	const path = "/p/.claude.json"
	installed := []byte(`{"mcpServers":{"context7":{"command":"c7"},"graphify":{"command":"gq"},"serena":{"command":"uvx"}}}`)

	r, err := Compute(
		[]state.Item{
			mcpItem("graphify", path, "graphify", installed, nil, false),
			mcpItem("serena", path, "serena", installed, nil, false),
		},
		readerFrom(map[string][]byte{path: installed}),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	writes := 0
	for _, d := range r.ChangeSet.Diffs {
		if d.Action == diff.Restore {
			writes++
			// Both are gone from the ONE composed buffer; the user's server stays.
			assertServers(t, d.After, []string{"context7"}, []string{"graphify", "serena"})
		}
	}
	if writes != 1 {
		t.Fatalf("want ONE composed write, got %d: %+v", writes, r.ChangeSet.Diffs)
	}
	// Both contributors are credited, and the display carries the second one so
	// the plan and the footer still report two logical removals.
	assertLedger(t, r.Ledger, map[string]Outcome{"graphify": Applied, "serena": Applied})
	if got := len(r.ChangeSet.Diffs[0].RestoreContrib); got != 1 {
		t.Errorf("want one RestoreContrib for the second artifact, got %d", got)
	}
}

// Removing one of N leaves the others alone — the single-contributor promise the
// composition must not break.
func TestOneOfSeveralContributorsLeavesSiblings(t *testing.T) {
	const path = "/p/.claude.json"
	installed := []byte(`{"mcpServers":{"context7":{"command":"c7"},"graphify":{"command":"gq"},"serena":{"command":"uvx"}}}`)

	r, err := Compute(
		[]state.Item{mcpItem("serena", path, "serena", installed, nil, false)},
		readerFrom(map[string][]byte{path: installed}),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertServers(t, r.ChangeSet.Diffs[0].After, []string{"context7", "graphify"}, []string{"serena"})
}

// Two hooks in one array are distinct list identities, so they commute and fold.
func TestTwoHookContributorsCompose(t *testing.T) {
	ft := manifest.FileTarget{File: "settings.json", Format: "json"}
	const dotted = "hooks.PreToolUse"
	const path = "/p/settings.json"

	elemA := map[string]any{"patronusId": "A", "matcher": "Edit"}
	elemB := map[string]any{"patronusId": "B", "matcher": "Write"}
	withA, err := adapter.AppendSettingsList(nil, ft, dotted, "patronusId", elemA)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := adapter.AppendSettingsList(withA, ft, dotted, "patronusId", elemB)
	if err != nil {
		t.Fatal(err)
	}

	hookItem := func(artifact, id string, elem map[string]any) state.Item {
		return state.Item{
			Artifact: artifact, Tool: "claude", Scope: "global",
			Files: []state.FileState{{
				Path: path, Action: string(diff.Merge), Checksum: sum(installed),
				Setting: &diff.SettingEdit{
					Target:      diff.FileTargetRef{File: ft.File, Format: ft.Format},
					Dotted:      dotted,
					IdentityKey: "patronusId", Identity: id, Elem: elem,
				},
			}},
		}
	}

	r, err := Compute(
		[]state.Item{hookItem("tdd-guard", "A", elemA), hookItem("gitleaks", "B", elemB)},
		readerFrom(map[string][]byte{path: installed}),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	restores := restoreDiffs(r.ChangeSet)
	if len(restores) != 1 {
		t.Fatalf("want one composed write, got %d", len(restores))
	}
	for _, gone := range []string{`"A"`, `"B"`} {
		if bytes.Contains(restores[0].After, []byte(gone)) {
			t.Errorf("element %s survived the composed removal:\n%s", gone, restores[0].After)
		}
	}
}

// Two edits at the SAME dotted key do not commute: no order-independent result
// exists, and state carries no trustworthy chronology to break the tie. Both are
// refused, and neither refusal is promotable — --force is consent to lose your
// own edit, not another artifact's wiring.
func TestOverlappingScalarEditsAreRefused(t *testing.T) {
	const path = "/p/.claude.json"
	installed := []byte(`{"mcpServers":{"serena":{"command":"uvx"}}}`)

	r, err := Compute(
		[]state.Item{
			mcpItem("recipe-a", path, "serena", installed, nil, false),
			mcpItem("recipe-b", path, "serena", installed, nil, false),
		},
		readerFrom(map[string][]byte{path: installed}),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(restoreDiffs(r.ChangeSet)); got != 0 {
		t.Fatalf("an ambiguous overlap must write nothing, got %d writes", got)
	}
	for _, d := range r.ChangeSet.Diffs {
		if d.Action != diff.Skip {
			t.Errorf("want SKIP, got %s", d.Action)
		}
		if d.Intended != "" {
			t.Errorf("an ambiguous SKIP must not be promotable, got Intended=%s", d.Intended)
		}
	}
	assertLedger(t, r.Ledger, map[string]Outcome{"recipe-a": AmbiguousSkipped, "recipe-b": AmbiguousSkipped})
	if len(r.Warnings) != 2 {
		t.Errorf("want a warning per refused contributor, got %d", len(r.Warnings))
	}
}

// An ancestor/descendant pair overlaps too: removing the parent key would take
// the child with it.
func TestAncestorDescendantEditsAreRefused(t *testing.T) {
	const path = "/p/settings.json"
	installed := []byte(`{"a":{"b":1}}`)
	item := func(artifact, dotted string) state.Item {
		return state.Item{
			Artifact: artifact, Tool: "claude", Scope: "global",
			Files: []state.FileState{{
				Path: path, Action: string(diff.Merge), Checksum: sum(installed),
				Setting: &diff.SettingEdit{
					Target: diff.FileTargetRef{File: "settings.json", Format: "json"},
					Dotted: dotted,
				},
			}},
		}
	}
	r, err := Compute(
		[]state.Item{item("parent", "a"), item("child", "a.b")},
		readerFrom(map[string][]byte{path: installed}),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLedger(t, r.Ledger, map[string]Outcome{"parent": AmbiguousSkipped, "child": AmbiguousSkipped})
}

// A sibling key under the same parent is NOT an overlap — "hooks.Pre" and
// "hooks.PreToolUse" share a text prefix but not a path segment.
func TestSiblingDottedKeysStillCompose(t *testing.T) {
	const path = "/p/settings.json"
	installed := []byte(`{"hooks":{"Pre":1,"PreToolUse":2}}`)
	item := func(artifact, dotted string) state.Item {
		return state.Item{
			Artifact: artifact, Tool: "claude", Scope: "global",
			Files: []state.FileState{{
				Path: path, Action: string(diff.Merge), Checksum: sum(installed),
				Setting: &diff.SettingEdit{
					Target: diff.FileTargetRef{File: "settings.json", Format: "json"},
					Dotted: dotted,
				},
			}},
		}
	}
	r, err := Compute(
		[]state.Item{item("one", "hooks.Pre"), item("two", "hooks.PreToolUse")},
		readerFrom(map[string][]byte{path: installed}),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertLedger(t, r.Ledger, map[string]Outcome{"one": Applied, "two": Applied})
}

// An unparseable file refuses the WHOLE same-path group. A partial buffer written
// over a file we could not read is worse than doing nothing.
func TestUnparseableFileRefusesWholeGroup(t *testing.T) {
	const path = "/p/.claude.json"
	broken := []byte(`{"mcpServers": {`)

	r, err := Compute(
		[]state.Item{
			mcpItem("graphify", path, "graphify", broken, nil, false),
			mcpItem("serena", path, "serena", broken, nil, false),
		},
		readerFrom(map[string][]byte{path: broken}),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(restoreDiffs(r.ChangeSet)); got != 0 {
		t.Fatalf("an unparseable file must produce no write, got %d", got)
	}
	assertLedger(t, r.Ledger, map[string]Outcome{"graphify": UnreadableSkipped, "serena": UnreadableSkipped})
}

// --- the legacy (pre-compose) arm --------------------------------------------

// legacyItem records a MERGE with NO SettingEdit: the pre-compose shape whose
// only undo is a whole-file snapshot restore.
func legacyItem(artifact, path string, prior, installed []byte) state.Item {
	return state.Item{
		Artifact: artifact, Tool: "claude", Scope: "global",
		Files: []state.FileState{{
			Path: path, Action: string(diff.Merge), Prior: prior, Checksum: sum(installed),
		}},
	}
}

func TestLegacySoleContributorStillRestores(t *testing.T) {
	const path = "/p/.claude.json"
	prior := []byte(`{}`)
	installed := []byte(`{"mcpServers":{"x":{}}}`)

	r, err := Compute(
		[]state.Item{legacyItem("old", path, prior, installed)},
		readerFrom(map[string][]byte{path: installed}),
		Occupancy{path: {contributor("old")}},
	)
	if err != nil {
		t.Fatal(err)
	}
	d := r.ChangeSet.Diffs[0]
	if d.Action != diff.Restore || !bytes.Equal(d.After, prior) {
		t.Fatalf("sole contributor should still restore its snapshot, got %s / %q", d.Action, d.After)
	}
	assertLedger(t, r.Ledger, map[string]Outcome{"old": Applied})
}

// The sole-contributor check must see UNSELECTED artifacts. Within the selection
// "old" looks alone; the loaded state knows better, and restoring its snapshot
// would silently un-wire the other artifact.
func TestLegacyRefusedWhenAnUnselectedArtifactSharesThePath(t *testing.T) {
	const path = "/p/.claude.json"
	installed := []byte(`{"mcpServers":{"x":{}}}`)

	r, err := Compute(
		[]state.Item{legacyItem("old", path, []byte(`{}`), installed)},
		readerFrom(map[string][]byte{path: installed}),
		Occupancy{path: {contributor("old"), contributor("still-installed")}},
	)
	if err != nil {
		t.Fatal(err)
	}
	d := r.ChangeSet.Diffs[0]
	if d.Action != diff.Skip {
		t.Fatalf("a shared legacy path must be refused, got %s", d.Action)
	}
	if d.Intended != "" {
		t.Error("the legacy refusal must not be promotable under --force")
	}
	assertLedger(t, r.Ledger, map[string]Outcome{"old": UnsafeLegacySkipped})
	if len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0].Message, "still-installed") {
		t.Errorf("want a warning naming the other contributor, got %+v", r.Warnings)
	}
}

// A path carrying BOTH shapes: the modern composite is still written and the
// legacy row is still refused. Neither blocks the other.
func TestMixedModernAndLegacyOnOnePath(t *testing.T) {
	const path = "/p/.claude.json"
	installed := []byte(`{"mcpServers":{"graphify":{"command":"gq"},"serena":{"command":"uvx"}}}`)

	r, err := Compute(
		[]state.Item{
			legacyItem("old", path, []byte(`{}`), installed),
			mcpItem("serena", path, "serena", installed, nil, false),
		},
		readerFrom(map[string][]byte{path: installed}),
		Occupancy{path: {contributor("old"), contributor("serena")}},
	)
	if err != nil {
		t.Fatal(err)
	}
	restores := restoreDiffs(r.ChangeSet)
	if len(restores) != 1 {
		t.Fatalf("the modern removal must still be written, got %d writes", len(restores))
	}
	assertServers(t, restores[0].After, []string{"graphify"}, []string{"serena"})
	assertLedger(t, r.Ledger, map[string]Outcome{"old": UnsafeLegacySkipped, "serena": Applied})
}

// --- helpers -----------------------------------------------------------------

// contributor builds an occupancy entry at the (claude, global) identity every
// case in this file uses.
func contributor(artifact string) Contributor {
	return Contributor{Artifact: artifact, Tool: "claude", Scope: "global"}
}

func restoreDiffs(cs *diff.ChangeSet) []diff.FileDiff {
	var out []diff.FileDiff
	for _, d := range cs.Diffs {
		if d.Action == diff.Restore {
			out = append(out, d)
		}
	}
	return out
}

// assertLedger checks the outcome recorded for each named artifact. It asserts on
// STRUCTURED codes, never on Note prose, which is user-facing copy and free to
// change.
func assertLedger(t *testing.T, l Ledger, want map[string]Outcome) {
	t.Helper()
	got := map[string]Outcome{}
	for _, e := range l {
		got[e.Artifact] = e.Outcome
	}
	for artifact, w := range want {
		if got[artifact] != w {
			t.Errorf("ledger[%s] = %q, want %q", artifact, got[artifact], w)
		}
	}
}

// --- peer-review regressions -------------------------------------------------

// The same artifact installed for two TOOLS is two independent state records. If
// occupancy identified contributors by name alone, the second record would look
// like "self" and the legacy snapshot would be restored straight over its wiring.
func TestLegacyRefusedWhenSameArtifactIsInstalledForAnotherTool(t *testing.T) {
	const path = "/p/shared.json"
	installed := []byte(`{"a":1}`)

	item := legacyItem("foo", path, []byte(`{}`), installed)
	other := Contributor{Artifact: "foo", Tool: "opencode", Scope: "global"}

	r, err := Compute(
		[]state.Item{item},
		readerFrom(map[string][]byte{path: installed}),
		Occupancy{path: {contributor("foo"), other}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if r.ChangeSet.Diffs[0].Action != diff.Skip {
		t.Fatalf("a second record of the same artifact is still another contributor; want SKIP, got %s",
			r.ChangeSet.Diffs[0].Action)
	}
	assertLedger(t, r.Ledger, map[string]Outcome{"foo": UnsafeLegacySkipped})
}

// A row whose edit cannot parse the file must refuse the WHOLE group, even when
// an earlier row in the group parsed it happily.
func TestUnparseableLaterRowStillRefusesWholeGroup(t *testing.T) {
	const path = "/p/config"
	// Valid JSON, so a json-targeted edit parses it; a toml-targeted edit cannot.
	content := []byte(`{"mcpServers":{"serena":{"command":"uvx"}}}`)

	tomlRow := state.Item{
		Artifact: "mismatched", Tool: "claude", Scope: "global",
		Files: []state.FileState{{
			Path: path, Action: string(diff.Merge), Checksum: sum(content),
			Setting: &diff.SettingEdit{
				Target: diff.FileTargetRef{File: "config", Format: "toml"},
				Dotted: "whatever",
			},
		}},
	}

	r, err := Compute(
		[]state.Item{mcpItem("serena", path, "serena", content, nil, false), tomlRow},
		readerFrom(map[string][]byte{path: content}),
		nil,
	)
	if err != nil {
		t.Fatalf("an unreadable config is a recoverable warn-and-skip, never a fatal: %v", err)
	}
	if got := len(restoreDiffs(r.ChangeSet)); got != 0 {
		t.Fatalf("the whole group must be refused, got %d writes", got)
	}
	assertLedger(t, r.Ledger, map[string]Outcome{"serena": UnreadableSkipped, "mismatched": UnreadableSkipped})
}

// --- completion classification (pat-mc0r) ------------------------------------

// An already-gone file means the deletion is DONE, not pending. Classifying it as
// merely "skipped" left the state row open forever: re-running remove after a
// partial cleanup never converged, and scan kept reporting the item installed
// with no way to ever retire it.
func TestAbsentTargetIsLogicallyComplete(t *testing.T) {
	cases := []struct {
		name  string
		item  state.Item
		files map[string][]byte
		want  Outcome
	}{{
		name: "CREATE target already deleted",
		item: state.Item{
			Artifact: "s", Tool: "claude", Scope: "global",
			Files: []state.FileState{{Path: "/c/SKILL.md", Action: "CREATE", Checksum: sum([]byte("x"))}},
		},
		files: map[string][]byte{},
		want:  AlreadyAbsent,
	}, {
		name: "FETCH target already deleted",
		item: state.Item{
			Artifact: "bin", Tool: "claude", Scope: "global",
			Files: []state.FileState{{Path: "/c/tool", Action: "FETCH", Checksum: sum([]byte("x"))}},
		},
		files: map[string][]byte{},
		want:  AlreadyAbsent,
	}, {
		name: "settings file itself is gone",
		item: state.Item{
			Artifact: "serena", Tool: "claude", Scope: "global",
			Files: []state.FileState{{
				Path: "/p/.claude.json", Action: string(diff.Merge), Checksum: sum([]byte("x")),
				Setting: &diff.SettingEdit{
					Target: diff.FileTargetRef{File: ".claude.json", Format: "json"},
					Dotted: "mcpServers.serena",
				},
			}},
		},
		files: map[string][]byte{},
		want:  AlreadyAbsent,
	}, {
		name: "settings file present, our key already gone",
		item: state.Item{
			Artifact: "serena", Tool: "claude", Scope: "global",
			Files: []state.FileState{{
				Path: "/p/.claude.json", Action: string(diff.Merge), Checksum: sum([]byte("x")),
				Setting: &diff.SettingEdit{
					Target: diff.FileTargetRef{File: ".claude.json", Format: "json"},
					Dotted: "mcpServers.serena",
				},
			}},
		},
		files: map[string][]byte{"/p/.claude.json": []byte(`{"mcpServers":{"other":{}}}`)},
		want:  SettingAbsent,
	}}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Compute([]state.Item{tt.item}, readerFrom(tt.files), nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := r.Ledger[0].Outcome; got != tt.want {
				t.Fatalf("outcome = %q, want %q", got, tt.want)
			}
			if !r.Ledger[0].Outcome.Complete() {
				t.Error("an already-satisfied removal must be COMPLETE, or the state row can never retire")
			}
		})
	}
}

// The incomplete codes are the feature that lets a later --force (or a repair and
// re-run) finish the job. Retiring the row on these would strand a file that is
// still on disk.
func TestUnfinishedRemovalsAreNotComplete(t *testing.T) {
	for _, o := range []Outcome{DriftSkipped, UnreadableSkipped, UnsafeLegacySkipped, AmbiguousSkipped, UnknownAction} {
		if o.Complete() {
			t.Errorf("%q leaves work undone; it must NOT count as complete", o)
		}
	}
}

// A drift SKIP holds the row open even when the item's OTHER files are settled.
// Partial completion is not completion.
func TestMixedCompleteAndIncompleteFilesHoldTheRowOpen(t *testing.T) {
	original := []byte("original")
	item := state.Item{
		Artifact: "s", Tool: "claude", Scope: "global",
		Files: []state.FileState{
			{Path: "/c/gone.md", Action: "CREATE", Checksum: sum(original)},   // already absent
			{Path: "/c/edited.md", Action: "CREATE", Checksum: sum(original)}, // drifted
		},
	}
	r, err := Compute(
		[]state.Item{item},
		readerFrom(map[string][]byte{"/c/edited.md": []byte("USER EDITED")}),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	complete := 0
	for _, e := range r.Ledger {
		if e.Outcome.Complete() {
			complete++
		}
	}
	if complete != 1 {
		t.Fatalf("want exactly one settled file of two, got %d: %+v", complete, r.Ledger)
	}
}
