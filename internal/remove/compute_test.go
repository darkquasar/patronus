package remove

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	cs, warns, err := Compute(items, readerFrom(map[string][]byte{"/c/SKILL.md": body}))
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
	cs, _, err := Compute(items, readerFrom(map[string][]byte{}))
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
	cs, warns, err := Compute(items, readerFrom(map[string][]byte{"/c/SKILL.md": []byte("USER EDITED")}))
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
	Promote(cs)
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
	cs, warns, err := Compute(items, readerFrom(map[string][]byte{"/p/CLAUDE.md": installed}))
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
	cs, warns, err := Compute(items, readerFrom(map[string][]byte{"/p/CLAUDE.md": other}))
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
	cs, warns, err := Compute(items, readerFrom(map[string][]byte{"/p/CLAUDE.md": edited}))
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
	cs, warns, err := Compute(items, readerFrom(map[string][]byte{"/p/CLAUDE.md": edited}))
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
	cs, warns, err := Compute(items, readerFrom(map[string][]byte{"/p/CLAUDE.md": installed}))
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
	cs, _, err := Compute(items, readerFrom(map[string][]byte{"/p/.mcp.json": installed}))
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
	cs, warns, err := Compute(items, readerFrom(map[string][]byte{"/p/settings.json": installed}))
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

	cs, warns, err := Compute(
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

	cs, _, err := Compute(
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

	cs, _, err := Compute(
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
	_, warns, err := Compute(items, readerFrom(map[string][]byte{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 1 {
		t.Fatalf("want one self-wired warning, got %d: %+v", len(warns), warns)
	}
}
