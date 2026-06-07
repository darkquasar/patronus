package state

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
)

func TestLoadMissingIsEmpty(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if s.Version != Version || len(s.Items) != 0 {
		t.Errorf("want empty state at current version, got %+v", s)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	s := &State{Version: Version, Items: []Item{{
		Artifact: "team-research", Tool: "claude", Scope: "global",
		Files: []FileState{{Path: "/x/SKILL.md", Action: "CREATE", Checksum: "sha256:abc"}},
	}}}
	if err := Save(p, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].Artifact != "team-research" {
		t.Errorf("round trip lost data: %+v", got)
	}
}

func TestSaveDeterministic(t *testing.T) {
	dir := t.TempDir()
	s := &State{Version: Version, Items: []Item{{Artifact: "a", Tool: "claude", Scope: "global"}}}
	p1, p2 := filepath.Join(dir, "1.json"), filepath.Join(dir, "2.json")
	if err := Save(p1, s); err != nil {
		t.Fatal(err)
	}
	if err := Save(p2, s); err != nil {
		t.Fatal(err)
	}
	b1, _ := os.ReadFile(p1)
	b2, _ := os.ReadFile(p2)
	if !bytes.Equal(b1, b2) {
		t.Error("identical state should serialize to identical bytes")
	}
}

func TestMergeUpserts(t *testing.T) {
	s := &State{Version: Version}
	Merge(s, []Item{{Artifact: "a", Tool: "claude", Scope: "global", ItemVersion: "1.0"}})
	Merge(s, []Item{{Artifact: "a", Tool: "claude", Scope: "global", ItemVersion: "2.0"}}) // same key
	Merge(s, []Item{{Artifact: "a", Tool: "claude", Scope: "local"}})                      // different scope

	if len(s.Items) != 2 {
		t.Fatalf("want 2 items (upsert + distinct scope), got %d", len(s.Items))
	}
	// The global one was replaced, not duplicated.
	for _, it := range s.Items {
		if it.Scope == "global" && it.ItemVersion != "2.0" {
			t.Errorf("upsert failed: global version = %q, want 2.0", it.ItemVersion)
		}
	}
}

func TestFromChangeSetGroupsAndCaptures(t *testing.T) {
	applied := []diff.FileDiff{
		{Path: "/c/SKILL.md", Action: diff.Create, After: []byte("body"), Artifact: "s", Tool: "claude", Scope: "global"},
		{Path: "/c/p/a.md", Action: diff.Create, After: []byte("a"), Artifact: "s", Tool: "claude", Scope: "global"},
		{Path: "/proj/CLAUDE.md", Action: diff.Append, Before: []byte("user prose\n"), After: []byte("user prose\n\nblock"),
			Artifact: "ap", Tool: "claude", Scope: "local", Section: &diff.SectionEdit{Name: "ap"}},
		{Path: "/proj/.mcp.json", Action: diff.Merge, Before: []byte("{}"), After: []byte(`{"mcpServers":{}}`),
			Artifact: "mem", Tool: "claude", Scope: "local"},
	}
	items := FromChangeSet(applied, "2026-06-07T00:00:00Z")

	// 3 groups: (s,claude,global), (ap,claude,local), (mem,claude,local).
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %d: %+v", len(items), items)
	}

	byArtifact := map[string]Item{}
	for _, it := range items {
		byArtifact[it.Artifact] = it
		if it.InstalledAt != "2026-06-07T00:00:00Z" {
			t.Errorf("timestamp not propagated: %q", it.InstalledAt)
		}
	}

	// Skill: 2 files, CREATE, no Prior (revert = delete), checksum set.
	s := byArtifact["s"]
	if len(s.Files) != 2 {
		t.Errorf("skill files = %d, want 2", len(s.Files))
	}
	for _, f := range s.Files {
		if f.Prior != nil {
			t.Error("CREATE must not capture Prior")
		}
		if f.Checksum == "" {
			t.Error("checksum missing")
		}
	}

	// Append: Section name + Prior captured.
	ap := byArtifact["ap"].Files[0]
	if ap.Section != "ap" {
		t.Errorf("append section = %q, want ap", ap.Section)
	}
	if string(ap.Prior) != "user prose\n" {
		t.Errorf("append prior = %q, want the pre-install prose", ap.Prior)
	}

	// Merge: Prior captured (pre-install config).
	mg := byArtifact["mem"].Files[0]
	if string(mg.Prior) != "{}" {
		t.Errorf("merge prior = %q, want {}", mg.Prior)
	}
}

func TestChecksumStable(t *testing.T) {
	a := checksum([]byte("x"))
	b := checksum([]byte("x"))
	if a != b {
		t.Error("checksum not stable")
	}
	if checksum([]byte("x")) == checksum([]byte("y")) {
		t.Error("checksum collision for different content")
	}
}
