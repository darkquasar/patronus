package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/remove"
	"github.com/darkquasar/patronus/internal/state"
)

func TestCodexInstructionRemovalFailureRetainsOwnership(t *testing.T) {
	for _, mode := range []string{"missing", "failed-first", "failed-after-sidecar"} {
		t.Run(mode, func(t *testing.T) {
			home, project := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
			path := filepath.Join(home, ".codex/AGENTS.md")
			sp := filepath.Join(home, ".patronus/state.json")
			prose := []byte("Invented user prose\n")
			a := diff.FileDiff{Artifact: "first-cx", Tool: "codex", Scope: "global", Path: path, Action: diff.Append, Before: prose, Section: &diff.SectionEdit{Name: "first-cx", Body: []byte("First body")}}
			a.After = adapter.AppendSection(prose, "first-cx", a.Section.Body)
			b := a
			b.Artifact, b.Section = "second-cx", &diff.SectionEdit{Name: "second-cx", Body: []byte("Second body")}
			b.Before, b.After = a.After, adapter.AppendSection(a.After, "second-cx", b.Section.Body)
			codexWrite(t, path, b.After)
			s := &state.State{Version: state.Version, Items: state.FromChangeSet([]diff.FileDiff{a}, "original-time")}
			state.Merge(s, state.FromChangeSet([]diff.FileDiff{b}, "second-time"))
			state.RefreshSectionChecksums(s.Items, []diff.FileDiff{b})
			side := diff.FileDiff{Artifact: a.Artifact, Tool: "codex", Scope: "global", Path: filepath.Join(home, "owned-sidecar.md"), Action: diff.Create, After: []byte("Invented owned sidecar")}
			tail := side
			tail.Path = filepath.Join(home, "unattempted-sidecar.md")
			codexWrite(t, side.Path, side.After)
			codexWrite(t, tail.Path, tail.After)
			state.Merge(s, state.FromChangeSet([]diff.FileDiff{side, tail}, "original-time"))
			if err := state.Save(sp, s); err != nil {
				t.Fatal(err)
			}
			beforeState := mustRead(t, sp)
			if mode == "missing" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if _, _, err := execRemove(t, a.Artifact, "--target", "codex", "--global", "--deploy"); err == nil {
					t.Fatal("missing context admitted")
				}
				if !bytes.Equal(mustRead(t, sp), beforeState) || !bytes.Equal(mustRead(t, side.Path), side.After) {
					t.Fatal("missing context changed ownership or sidecar")
				}
				return
			}
			selected := s.Find(a.Artifact, "codex", "global")
			inverse, err := remove.Compute(selected, func(path string) ([]byte, bool, error) { raw, err := os.ReadFile(path); return raw, err == nil, err }, occupancyOf([]*state.State{s}))
			if err != nil {
				t.Fatal(err)
			}
			var section, first, last diff.FileDiff
			for _, d := range inverse.ChangeSet.Diffs {
				switch d.Path {
				case path:
					section = d
				case side.Path:
					first = d
				case tail.Path:
					last = d
				}
			}
			inverse.ChangeSet.Diffs = []diff.FileDiff{section, first, last}
			if mode == "failed-after-sidecar" {
				inverse.ChangeSet.Diffs = []diff.FileDiff{first, section, last}
			}
			if err := os.Chmod(filepath.Dir(path), 0500); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(filepath.Dir(path), 0700) })
			cmd := newRemoveCmd(nil)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			if err := runRemove(cmd, inverse.ChangeSet, inverse.Ledger, selected, map[string]*state.State{"global": s}, removeStateOpts{home: home, projectDir: project}); err == nil {
				t.Fatal("failed physical section write reported success")
			}
			got, err := state.Load(sp)
			if err != nil {
				t.Fatal(err)
			}
			want := &state.State{}
			if err := os.WriteFile(filepath.Join(home, "original.json"), beforeState, 0600); err != nil {
				t.Fatal(err)
			}
			want, err = state.Load(filepath.Join(home, "original.json"))
			if err != nil {
				t.Fatal(err)
			}
			if mode == "failed-after-sidecar" {
				state.ForgetEffects(want, selected[0], []state.FileState{state.FromChangeSet([]diff.FileDiff{side}, "")[0].Files[0]})
				if _, err := os.Stat(side.Path); !os.IsNotExist(err) {
					t.Fatal("sidecar deletion did not land")
				}
			}
			if !reflect.DeepEqual(got.Items, want.Items) {
				t.Fatalf("failed/unattempted ownership credited or settled sidecar retained: got=%+v want=%+v", got.Items, want.Items)
			}
			if !bytes.Equal(mustRead(t, path), b.After) || !bytes.Equal(mustRead(t, tail.Path), tail.After) {
				t.Fatal("failed/unattempted physical effect changed")
			}
		})
	}
}

func TestCodexInstructionRemovalPreservesSiblingSections(t *testing.T) {
	for _, selection := range [][]string{
		{"fix-instruction-global", "fix-instruction-2"},
		{"fix-instruction-2", "fix-instruction-global"},
		{"fix-instruction-global", "fix-instruction-2", "together"},
		{"fix-instruction-2", "fix-instruction-global", "together"},
	} {
		t.Run(strings.Join(selection, "/"), func(t *testing.T) {
			root := fixtureCatalog(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
			t.Chdir(root)
			path := filepath.Join(home, ".codex/AGENTS.md")
			sp := filepath.Join(home, ".patronus/state.json")
			prose := []byte("Invented verified user prose\n")
			// Seed existing positively owned context, not implicit adoption of prose.
			d := diff.FileDiff{Artifact: "fix-instruction-global", Tool: "codex", Scope: "global", Path: path, Action: diff.Append, Before: prose, Section: &diff.SectionEdit{Name: "fix-instruction-global", Body: []byte("Invented initial body")}}
			d.After = adapter.AppendSection(prose, d.Section.Name, d.Section.Body)
			codexWrite(t, path, d.After)
			if err := state.Save(sp, &state.State{Version: state.Version, Items: state.FromChangeSet([]diff.FileDiff{d}, "initial")}); err != nil {
				t.Fatal(err)
			}
			if _, _, err := runInstall(t, "fix-instruction-global", "fix-instruction-2", "--target", "codex", "--global", "--deploy"); err != nil {
				t.Fatal(err)
			}
			assertCodexInstructionChecksums(t, home, 2)
			if len(selection) == 3 {
				if _, _, err := execRemove(t, selection[0], selection[1], "--target", "codex", "--deploy"); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, _, err := execRemove(t, selection[0], "--target", "codex", "--deploy"); err != nil {
					t.Fatal(err)
				}
				assertCodexInstructionChecksums(t, home, 1)
				if _, present := adapter.SectionBody(mustRead(t, path), selection[0]); present {
					t.Fatal("selected section survived")
				}
				if _, present := adapter.SectionBody(mustRead(t, path), selection[1]); !present {
					t.Fatal("sibling section lost")
				}
				if _, _, err := runInstall(t, selection[1], "--target", "codex", "--global", "--deploy"); err != nil {
					t.Fatalf("survivor reinstall: %v", err)
				}
				mp := filepath.Join(root, "artifacts/instructions", selection[1], "patronus.yaml")
				codexWrite(t, mp, []byte(strings.ReplaceAll(string(mustRead(t, mp)), "version: 1.0.0", "version: 2.0.0")))
				body := filepath.Join(root, "artifacts/instructions", selection[1], "INSTRUCTIONS.md")
				codexWrite(t, body, append(mustRead(t, body), []byte("\nInvented updated rule.\n")...))
				if _, _, err := runUpdate(t, selection[1], "--target", "codex", "--deploy"); err != nil {
					t.Fatalf("survivor update: %v", err)
				}
				if _, _, err := execRemove(t, selection[1], "--target", "codex", "--deploy"); err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(mustRead(t, path), prose) {
				t.Fatalf("user prose changed: %q", mustRead(t, path))
			}
			s, err := state.Load(sp)
			if err != nil {
				t.Fatal(err)
			}
			if len(s.Items) != 0 {
				t.Fatalf("completed owners retained: %+v", s.Items)
			}
		})
	}
}
