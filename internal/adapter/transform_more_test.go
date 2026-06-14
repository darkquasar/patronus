package adapter

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func TestTransformInstructionAppend(t *testing.T) {
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "INSTRUCTIONS.md"), "house rules")
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "agent-principles", Role: manifest.RoleInstruction}, Type: manifest.TypeInstruction, Entry: "INSTRUCTIONS.md"}

	diffs, err := eng.Transform(art, claudeAdapter(t), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].Action != diff.Append {
		t.Fatalf("want 1 APPEND diff, got %+v", diffs)
	}
	wantPath := filepath.Join(home, ".claude", "CLAUDE.md")
	if diffs[0].Path != wantPath {
		t.Errorf("path = %q, want %q", diffs[0].Path, wantPath)
	}
	if !strings.Contains(string(diffs[0].After), "house rules") {
		t.Errorf("body not present: %q", diffs[0].After)
	}
}

func TestTransformInstructionFoldsExisting(t *testing.T) {
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "INSTRUCTIONS.md"), "new body")
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "ap"}, Type: manifest.TypeInstruction, Entry: "INSTRUCTIONS.md"}

	existing := []byte("user prose\n")
	read := func(string) ([]byte, bool, error) { return existing, true, nil }

	diffs, err := eng.Transform(art, claudeAdapter(t), "global", src, read)
	if err != nil {
		t.Fatal(err)
	}
	d := diffs[0]
	if string(d.Before) != "user prose\n" {
		t.Errorf("Before not captured: %q", d.Before)
	}
	if !strings.Contains(string(d.After), "user prose") {
		t.Errorf("existing prose dropped: %q", d.After)
	}
}

func TestTransformCommand(t *testing.T) {
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "do-thing.md"), "command body")
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "do-thing"}, Type: manifest.TypeCommand, Entry: "do-thing.md"}

	diffs, err := eng.Transform(art, claudeAdapter(t), "global", src, noExisting)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".claude", "commands", "do-thing.md")
	if diffs[0].Path != want || string(diffs[0].After) != "command body" {
		t.Errorf("command diff wrong: %+v", diffs[0])
	}
}

func TestTransformCommandDefaultEntry(t *testing.T) {
	src := t.TempDir()
	// No explicit Entry; defaults to <name>.md.
	mustWrite(t, filepath.Join(src, "foo.md"), "x")
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	art := &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "foo"}, Type: manifest.TypeCommand}
	if _, err := eng.Transform(art, claudeAdapter(t), "global", src, noExisting); err != nil {
		t.Fatalf("default entry resolution failed: %v", err)
	}
}
