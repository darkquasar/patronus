package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestCoreResolvesToPlanExecute proves the profile swap through the lockfile:
// core RESOLVES to plan-execute and to NEITHER upstream execution skill, while
// dispatching-parallel-agents survives (it is not part of this fork).
//
// Resolution, not deployment. `core` carries binary recipes (gitleaks via
// github-release, ccusage via npm, plus hosted MCP entries), and builtRegistry
// deliberately serves NO binary bodies: its own comment at
// cmd/patronus/integration_test.go:50-55 says a test that reaches it and then
// installs a binary "will FETCH, miss in the served bodies, and fail loudly".
// So this asserts the profile's membership through the lockfile, exactly as
// TestOrchestrationLock already does at orchestration_integration_test.go:133.
// Placement is proved separately, by name, in TestPlanExecutePlacesOnAllTargets.
func TestCoreResolvesToPlanExecute(t *testing.T) {
	f := builtRegistry(t)
	withRemoteEnv(t, f)

	if _, _, err := runLock(t, "--profile", "core", "--target", "claude"); err != nil {
		t.Fatalf("lock core: %v", err)
	}
	wd, _ := os.Getwd()
	s := string(mustRead(t, filepath.Join(wd, "patronus.lock")))

	for _, want := range []string{"plan-execute", "dispatching-parallel-agents"} {
		if !strings.Contains(s, want) {
			t.Errorf("core must resolve %q:\n%s", want, s)
		}
	}
	// Word-bounded: "executing-plans" is a substring of nothing here, but
	// "plan-execute" must not be read as a hit for either removed name.
	for _, gone := range []string{"executing-plans", "subagent-driven-development"} {
		if regexp.MustCompile(`"` + regexp.QuoteMeta(gone) + `"`).MatchString(s) {
			t.Errorf("core must no longer resolve %q; plan-execute replaces it:\n%s", gone, s)
		}
	}
}

// TestPlanExecutePlacesOnAllTargets proves the artifact actually lands, on all
// three targets: the spec's verification says "on all three targets", and each
// adapter places skills under a different global root, so a claude-only
// assertion would not prove the swap on codex or opencode.
//
// Installs plan-execute BY NAME rather than through the profile, so no binary
// recipe is ever reached. Its requires: edge pulls requesting-code-review, which
// is also an artifact.
func TestPlanExecutePlacesOnAllTargets(t *testing.T) {
	// Global skill root per target, from adapters/<target>.yaml `skill.global`.
	for _, tc := range []struct {
		target   string
		skillDir []string // path segments under the temp HOME
	}{
		{"claude", []string{".claude", "skills"}},
		{"codex", []string{".codex", "skills"}},
		{"opencode", []string{".config", "opencode", "skills"}},
	} {
		t.Run(tc.target, func(t *testing.T) {
			f := builtRegistry(t)
			home := withRemoteEnv(t, f)

			if _, errOut, err := runInstall(t,
				"plan-execute", "--target", tc.target, "--global", "--deploy", "--yes"); err != nil {
				t.Fatalf("install plan-execute for %s: %v\n%s", tc.target, err, errOut)
			}

			skills := filepath.Join(append([]string{home}, tc.skillDir...)...)
			if _, err := os.Stat(filepath.Join(skills, "plan-execute", "SKILL.md")); err != nil {
				t.Errorf("plan-execute not placed on %s: %v", tc.target, err)
			}
			// The sidecars the router loads must be packed, or the mode it picks is a
			// dangling reference.
			for _, rel := range []string{"solo.md", "sdd.md", "implementer-prompt.md",
				filepath.Join("scripts", "review-package")} {
				if _, err := os.Stat(filepath.Join(skills, "plan-execute", rel)); err != nil {
					t.Errorf("plan-execute sidecar %q not packed on %s: %v", rel, tc.target, err)
				}
			}
		})
	}
}

// TestPlanExecuteRequiresClosure proves the requires: edge on a STANDALONE
// install: asking for plan-execute alone also installs requesting-code-review,
// whose code-reviewer.md both modes fill for the final whole-branch review. An
// unresolved edge here means the review step points at a file that is not there.
func TestPlanExecuteRequiresClosure(t *testing.T) {
	f := builtRegistry(t)
	home := withRemoteEnv(t, f)

	out, errOut, err := runInstall(t, "plan-execute", "--target", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run install plan-execute: %v\n%s", err, errOut)
	}
	if !strings.Contains(errOut, "requesting-code-review") && !strings.Contains(out, "requesting-code-review") {
		t.Errorf("the requires closure should pull requesting-code-review:\nstdout:\n%s\nstderr:\n%s", out, errOut)
	}

	if _, e, err := runInstall(t, "plan-execute", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install plan-execute: %v\n%s", err, e)
	}
	p := filepath.Join(home, ".claude", "skills", "requesting-code-review", "code-reviewer.md")
	if _, err := os.Stat(p); err != nil {
		t.Errorf("code-reviewer.md not placed by the closure at %s: %v", p, err)
	}
}
