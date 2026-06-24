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

// TestFromChangeSetExpandsContrib covers a composed APPEND file carrying sections
// from two artifacts: state must record each under its own artifact (with its
// section + prior) so remove can strip exactly one.
func TestFromChangeSetExpandsContrib(t *testing.T) {
	applied := []diff.FileDiff{{
		Path: "/proj/CLAUDE.md", Action: diff.Append,
		Before:   []byte(""),
		After:    []byte("spine\n\nrules"),
		Artifact: "spine", Version: "1.0.0", Tool: "claude", Scope: "local",
		Section: &diff.SectionEdit{Name: "spine"},
		Contrib: []diff.SectionContrib{{
			Artifact: "rules", Version: "2.0.0", Section: "rules", Prior: []byte("spine"),
		}},
	}}
	items := FromChangeSet(applied, "now")

	if len(items) != 2 {
		t.Fatalf("want 2 items (spine + rules), got %d: %+v", len(items), items)
	}
	by := map[string]Item{}
	for _, it := range items {
		by[it.Artifact] = it
	}
	r, ok := by["rules"]
	if !ok {
		t.Fatal("contrib artifact 'rules' not recorded")
	}
	if r.ItemVersion != "2.0.0" {
		t.Errorf("rules version = %q, want 2.0.0", r.ItemVersion)
	}
	if len(r.Files) != 1 || r.Files[0].Section != "rules" || r.Files[0].Path != "/proj/CLAUDE.md" {
		t.Fatalf("rules file record wrong: %+v", r.Files)
	}
	if string(r.Files[0].Prior) != "spine" {
		t.Errorf("rules prior = %q, want the file before rules folded in (spine)", r.Files[0].Prior)
	}
}

func TestFromChangeSetCapturesItemVersion(t *testing.T) {
	applied := []diff.FileDiff{
		{Path: "/c/SKILL.md", Action: diff.Create, After: []byte("b"), Artifact: "s", Version: "1.2.0", Tool: "claude", Scope: "global"},
	}
	items := FromChangeSet(applied, "now")
	if len(items) != 1 || items[0].ItemVersion != "1.2.0" {
		t.Fatalf("itemVersion not captured: %+v", items)
	}
}

func TestFind(t *testing.T) {
	s := &State{Version: Version, Items: []Item{
		{Artifact: "a", Tool: "claude", Scope: "global"},
		{Artifact: "a", Tool: "codex", Scope: "global"},
		{Artifact: "a", Tool: "claude", Scope: "local"},
		{Artifact: "b", Tool: "claude", Scope: "global"},
	}}
	if got := s.Find("a", "", ""); len(got) != 3 {
		t.Errorf("Find(a) = %d items, want 3", len(got))
	}
	if got := s.Find("a", "claude", ""); len(got) != 2 {
		t.Errorf("Find(a,claude) = %d items, want 2", len(got))
	}
	if got := s.Find("a", "claude", "local"); len(got) != 1 {
		t.Errorf("Find(a,claude,local) = %d items, want 1", len(got))
	}
	if got := s.Find("missing", "", ""); len(got) != 0 {
		t.Errorf("Find(missing) = %d items, want 0", len(got))
	}
}

func TestRemove(t *testing.T) {
	s := &State{Version: Version, Items: []Item{
		{Artifact: "a", Tool: "claude", Scope: "global"},
		{Artifact: "a", Tool: "codex", Scope: "global"},
		{Artifact: "b", Tool: "claude", Scope: "global"},
	}}
	if n := s.Remove("a", "", ""); n != 2 {
		t.Errorf("Remove(a) removed %d, want 2", n)
	}
	if len(s.Items) != 1 || s.Items[0].Artifact != "b" {
		t.Errorf("after Remove(a), items = %+v, want only b", s.Items)
	}
	if n := s.Remove("nope", "", ""); n != 0 {
		t.Errorf("Remove(nope) removed %d, want 0", n)
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

func TestStateMergeRemovePlugin(t *testing.T) {
	s := &State{Version: 1}
	Merge(s, []Item{{Artifact: "superpowers", Tool: "claude", Scope: "global", ItemVersion: "2.1.0"}})
	if len(s.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(s.Items))
	}
	// Re-merge same identity replaces, not duplicates.
	Merge(s, []Item{{Artifact: "superpowers", Tool: "claude", Scope: "global", ItemVersion: "2.2.0"}})
	if len(s.Items) != 1 || s.Items[0].ItemVersion != "2.2.0" {
		t.Fatalf("re-merge wrong: %+v", s.Items)
	}
	if removed := s.Remove("superpowers", "claude", "global"); removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if len(s.Items) != 0 {
		t.Errorf("items after remove = %d, want 0", len(s.Items))
	}
}
