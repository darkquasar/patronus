package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// scriptPath resolves a plan-execute helper by absolute path, because the
// helpers resolve their siblings from their own directory.
func scriptPath(t *testing.T, name string) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("../../artifacts/skills/plan-execute/scripts", name))
	if err != nil {
		t.Fatalf("resolve %s: %v", name, err)
	}
	return p
}

// planRepo builds a throwaway git repo holding one spec folder with two streams
// and one plan outside docs/specs, which is the shape the workspace rules are
// written against.
func planRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	// macOS temp dirs are symlinked through /private; resolve so the paths the
	// scripts print (which canonicalize) compare equal to the ones we build.
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	for _, dir := range []string{"docs/specs/01-thing", "elsewhere"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	plan := "# Plan\n\n## Task 1: alpha\n\nbody alpha\n\n## Task 2: beta\n\nbody beta\n"
	for _, rel := range []string{
		"docs/specs/01-thing/stream-a-plan.md",
		"docs/specs/01-thing/stream-b-plan.md",
		"elsewhere/adhoc.md",
	} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(plan), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@example.test"},
		{"config", "user.name", "t"},
		{"add", "-A"},
		{"commit", "-qm", "seed"},
	} {
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func runScript(t *testing.T, dir, script string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), script, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// TestWorkspaceLivesBesideItsPlan is the whole point of the per-plan workspace:
// run state colocates with the plan that produced it, and never creates a
// directory at the repository root. A root-level workspace is what a user sees
// as junk appearing in their project after a fresh install.
func TestWorkspaceLivesBesideItsPlan(t *testing.T) {
	root := planRepo(t)
	ws := scriptPath(t, "sdd-workspace")

	got, err := runScript(t, root, ws, "docs/specs/01-thing/stream-a-plan.md")
	if err != nil {
		t.Fatalf("sdd-workspace: %v\n%s", err, got)
	}
	want := filepath.Join(root, "docs/specs/01-thing/.sdd/stream-a-plan")
	if got != want {
		t.Errorf("workspace = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, ".superpowers")); !os.IsNotExist(err) {
		t.Errorf("a root-level .superpowers/ was created; the workspace must live beside the plan")
	}

	// The self-ignoring file belongs INSIDE the workspace, never in the spec
	// folder, whose other contents are the user's to track.
	if b, err := os.ReadFile(filepath.Join(want, ".gitignore")); err != nil || strings.TrimSpace(string(b)) != "*" {
		t.Errorf(".sdd/<stem>/.gitignore = %q, %v; want a self-ignoring \"*\"", b, err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/specs/01-thing/.gitignore")); !os.IsNotExist(err) {
		t.Errorf("the spec folder itself must not be given a .gitignore")
	}
}

// TestWorkspaceSeparatesStreamsAndExternalPlans covers the two cases that drove
// the per-plan shape: one spec folder holds several streams whose ledgers must
// not mix, and a plan kept outside docs/specs colocates the same way rather
// than falling back to the repository root.
func TestWorkspaceSeparatesStreamsAndExternalPlans(t *testing.T) {
	root := planRepo(t)
	ws := scriptPath(t, "sdd-workspace")

	a, err := runScript(t, root, ws, "docs/specs/01-thing/stream-a-plan.md")
	if err != nil {
		t.Fatalf("stream-a: %v\n%s", err, a)
	}
	b, err := runScript(t, root, ws, "docs/specs/01-thing/stream-b-plan.md")
	if err != nil {
		t.Fatalf("stream-b: %v\n%s", err, b)
	}
	if a == b {
		t.Errorf("two streams in one spec folder share the workspace %q; their ledgers would mix", a)
	}

	ext, err := runScript(t, root, ws, "elsewhere/adhoc.md")
	if err != nil {
		t.Fatalf("external plan: %v\n%s", err, ext)
	}
	if want := filepath.Join(root, "elsewhere/.sdd/adhoc"); ext != want {
		t.Errorf("external plan workspace = %q, want %q", ext, want)
	}
}

// TestWorkspaceStemEdgeCases pins the two ways a filename can defeat the
// per-plan isolation: a dotfile whose extension strip leaves nothing, and two
// spellings of one plan through a symlink. Both would silently hand separate
// runs the same ledger, or hide an existing ledger from a resuming controller.
func TestWorkspaceStemEdgeCases(t *testing.T) {
	root := planRepo(t)
	ws := scriptPath(t, "sdd-workspace")
	dir := filepath.Join(root, "docs/specs/01-thing")

	if err := os.WriteFile(filepath.Join(dir, ".hidden"), []byte("# Plan\n"), 0o644); err != nil {
		t.Fatalf("write dotfile plan: %v", err)
	}
	got, err := runScript(t, root, ws, "docs/specs/01-thing/.hidden")
	if err != nil {
		t.Fatalf("dotfile plan: %v\n%s", err, got)
	}
	if want := filepath.Join(dir, ".sdd/.hidden"); got != want {
		t.Errorf("dotfile plan workspace = %q, want %q (an empty stem collapses into the shared .sdd/)", got, want)
	}

	if err := os.Symlink(filepath.Join(dir, "stream-a-plan.md"), filepath.Join(dir, "alias.md")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	viaLink, err := runScript(t, root, ws, "docs/specs/01-thing/alias.md")
	if err != nil {
		t.Fatalf("symlinked plan: %v\n%s", err, viaLink)
	}
	if want := filepath.Join(dir, ".sdd/stream-a-plan"); viaLink != want {
		t.Errorf("symlinked plan workspace = %q, want %q; two spellings of one plan must share a ledger", viaLink, want)
	}
}

// TestWorkspaceRequiresAResolvablePlan proves the script fails loudly instead of
// falling back to a repo-root directory, which is the behaviour the per-plan
// workspace replaced.
func TestWorkspaceRequiresAResolvablePlan(t *testing.T) {
	root := planRepo(t)
	ws := scriptPath(t, "sdd-workspace")

	for _, tc := range []struct{ name, arg string }{
		{"no argument", ""},
		{"missing plan", "does/not/exist.md"},
	} {
		var args []string
		if tc.arg != "" {
			args = append(args, tc.arg)
		}
		out, err := runScript(t, root, ws, args...)
		if err == nil {
			t.Errorf("%s: sdd-workspace exited 0 and printed %q; it must fail rather than guess a workspace", tc.name, out)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".superpowers")); !os.IsNotExist(err) {
		t.Errorf("a failed resolution must not leave a root-level .superpowers/ behind")
	}
}

// TestHelpersWriteIntoTheirPlansWorkspace proves the threading actually holds:
// both callers resolve the SAME directory the workspace script does. They each
// re-derive it, so a signature that silently dropped the plan would put briefs
// and review packages somewhere else.
func TestHelpersWriteIntoTheirPlansWorkspace(t *testing.T) {
	root := planRepo(t)
	plan := "docs/specs/01-thing/stream-a-plan.md"
	want := filepath.Join(root, "docs/specs/01-thing/.sdd/stream-a-plan")

	out, err := runScript(t, root, scriptPath(t, "task-brief"), plan, "2")
	if err != nil {
		t.Fatalf("task-brief: %v\n%s", err, out)
	}
	if !strings.Contains(out, filepath.Join(want, "task-2-brief.md")) {
		t.Errorf("task-brief wrote to %q, want a file under %q", out, want)
	}

	head, err := runScript(t, root, "git", "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse: %v\n%s", err, head)
	}
	out, err = runScript(t, root, scriptPath(t, "review-package"), plan, head, head)
	if err != nil {
		t.Fatalf("review-package: %v\n%s", err, out)
	}
	if !strings.Contains(out, want+string(filepath.Separator)) {
		t.Errorf("review-package wrote to %q, want a file under %q", out, want)
	}

	// review-package takes the plan as its FIRST argument now. Called with the
	// old BASE HEAD signature it must refuse, not treat a sha as a plan path.
	if out, err := runScript(t, root, scriptPath(t, "review-package"), head, head); err == nil {
		t.Errorf("review-package accepted the old two-argument signature and printed %q", out)
	}
}
