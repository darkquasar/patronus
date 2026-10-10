package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/state"
)

// sharedCreateFixture is an invented catalog whose `toy` adapter places every
// skill at ONE fixed project path, so two invented skills with identical bytes
// are two artifacts placing one shared file (the shape a generic hook runner
// has). It returns the project root, the shared path and a registry with a
// strict, admitted `toy` descriptor.
func sharedCreateFixture(t *testing.T) (project, shared string, reg *harness.Registry) {
	t.Helper()
	project = fixtureCatalog(t)
	write := func(rel, body string) {
		p := filepath.Join(project, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("adapters/toy.yaml", `apiVersion: patronus/v2
family: adapter
tool: toy
detect:
  global: ["~/.toy/"]
  project: [".toy/"]
layout:
  skill:
    global: "~/.toy/plugins/shared.md"
    project: ".toy/plugins/shared.md"
    nameSource: dir
    frontmatter: passthrough
`)
	for _, name := range []string{"alpha", "beta"} {
		write("artifacts/skills/"+name+"/patronus.yaml", `apiVersion: patronus/v2
family: artifact
type: skill
role: capability
name: `+name+`
description: "Invented contributor of one shared placed file."
version: 1.0.0
entry: SKILL.md
targets: [toy]
defaults:
  scope: project
`)
		write("artifacts/skills/"+name+"/SKILL.md", "shared runner bytes\n")
	}
	reg, err := harness.NewRegistry(harness.Descriptor{
		Name: "toy", ContextFile: "TOY.md", GlobalMarker: "~/.toy", StrictUnowned: true, SafePathCheck: true,
		AdmissionPrefix: "toy admission", ErrorPrefix: "toy", AdmitPlan: true, AdmitRemoval: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return project, filepath.Join(project, ".toy/plugins/shared.md"), reg
}

func runWith(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func createRows(t *testing.T, project string) map[string]string {
	t.Helper()
	s, err := state.Load(filepath.Join(project, ".patronus/state.json"))
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]string{}
	for _, it := range s.Items {
		for _, f := range it.Files {
			rows[it.Artifact] = f.Action + " " + f.Checksum
		}
	}
	return rows
}

func mustExist(t *testing.T, path string, want bool) {
	t.Helper()
	_, err := os.Stat(path)
	if got := err == nil; got != want {
		t.Fatalf("%s exists=%v, want %v (%v)", path, got, want, err)
	}
}

// TestHarnessSharedCreateLifecycle drives install and remove through the real
// commands for both acquisition orders: both contributors in one install, and
// one after the other (the second is adopted from an identical SKIP). Removing
// one contributor releases its row and keeps the file; removing the last one
// deletes the file and leaves no row.
func TestHarnessSharedCreateLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name     string
		installs [][]string
	}{
		{"simultaneous", [][]string{{"alpha", "beta"}}},
		{"sequential", [][]string{{"alpha"}, {"beta"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project, shared, reg := sharedCreateFixture(t)
			t.Setenv("HOME", t.TempDir())
			t.Chdir(project)
			for _, names := range tc.installs {
				args := append(append([]string{}, names...), "--target", "toy", "--local", "--deploy")
				if out, err := runWith(t, newInstallCmd(reg), args...); err != nil {
					t.Fatalf("install %v: %v\n%s", names, err, out)
				}
			}
			want := "CREATE " + state.Checksum([]byte("shared runner bytes\n"))
			if rows := createRows(t, project); rows["alpha"] != want || rows["beta"] != want {
				t.Fatalf("rows after install: %v", rows)
			}
			mustExist(t, shared, true)

			out, err := runWith(t, newRemoveCmd([]string{"revert"}, reg), "alpha", "--local", "--deploy")
			if err != nil {
				t.Fatalf("remove alpha: %v\n%s", err, out)
			}
			if !strings.Contains(out, "shared placed file still owned by beta") {
				t.Fatalf("release not reported:\n%s", out)
			}
			mustExist(t, shared, true)
			if rows := createRows(t, project); len(rows) != 1 || rows["beta"] != want {
				t.Fatalf("rows after releasing alpha: %v", rows)
			}

			if out, err := runWith(t, newRemoveCmd([]string{"revert"}, reg), "beta", "--local", "--deploy"); err != nil {
				t.Fatalf("remove beta: %v\n%s", err, out)
			}
			mustExist(t, shared, false)
			if rows := createRows(t, project); len(rows) != 0 {
				t.Fatalf("rows after removing the last contributor: %v", rows)
			}
		})
	}
}

// TestHarnessSharedCreateRemoveBoth: removing every contributor in one command
// deletes the file once and retires both rows.
func TestHarnessSharedCreateRemoveBoth(t *testing.T) {
	project, shared, reg := sharedCreateFixture(t)
	t.Setenv("HOME", t.TempDir())
	t.Chdir(project)
	if out, err := runWith(t, newInstallCmd(reg), "alpha", "beta", "--target", "toy", "--local", "--deploy"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	out, err := runWith(t, newRemoveCmd([]string{"revert"}, reg), "alpha", "beta", "--local", "--deploy")
	if err != nil {
		t.Fatalf("remove both: %v\n%s", err, out)
	}
	mustExist(t, shared, false)
	if rows := createRows(t, project); len(rows) != 0 {
		t.Fatalf("rows after removing both: %v", rows)
	}
}

// TestHarnessSharedCreateFailedDeleteKeepsRows: when the physical delete fails,
// no contributor row is retired and the command reports the failure.
func TestHarnessSharedCreateFailedDeleteKeepsRows(t *testing.T) {
	project, shared, reg := sharedCreateFixture(t)
	t.Setenv("HOME", t.TempDir())
	t.Chdir(project)
	if out, err := runWith(t, newInstallCmd(reg), "alpha", "beta", "--target", "toy", "--local", "--deploy"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	dir := filepath.Dir(shared)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	if _, err := runWith(t, newRemoveCmd([]string{"revert"}, reg), "alpha", "beta", "--local", "--deploy"); err == nil {
		t.Fatal("remove succeeded although the delete was refused")
	}
	mustExist(t, shared, true)
	if rows := createRows(t, project); len(rows) != 2 {
		t.Fatalf("rows after failed delete: %v", rows)
	}
}
