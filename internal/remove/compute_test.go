package remove

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
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
