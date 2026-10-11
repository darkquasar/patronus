package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A synthetic routing profile resolves a requires closure into its lock.
func TestCoreResolvesToPlanExecute(t *testing.T) {
	f := serveFixtureFrom(t, fixtureSkillBundle(t))
	withRemoteEnv(t, f)

	if _, _, err := runLock(t, "--profile", "fix-routing", "--target", "claude"); err != nil {
		t.Fatalf("lock core: %v", err)
	}
	wd, _ := os.Getwd()
	s := string(mustRead(t, filepath.Join(wd, "patronus.lock")))

	for _, want := range []string{"fix-router", "fix-review"} {
		if !strings.Contains(s, want) {
			t.Errorf("core must resolve %q:\n%s", want, s)
		}
	}
	// Word-bounded: "executing-plans" is a substring of nothing here, but
	// "fix-router" must not be read as a hit for either removed name.
	for _, gone := range []string{"fix-unselected", "fix-skill"} {
		if regexp.MustCompile(`"` + regexp.QuoteMeta(gone) + `"`).MatchString(s) {
			t.Errorf("core must no longer resolve %q; fix-router replaces it:\n%s", gone, s)
		}
	}
}

// TestPlanExecutePlacesOnAllTargets proves the artifact actually lands, on all
// three targets: the spec's verification says "on all three targets", and each
// adapter places skills under a different global root, so a claude-only
// assertion would not prove the swap on codex or opencode.
//
// Installs fix-router BY NAME rather than through the profile, so no binary
// recipe is ever reached. Its requires: edge pulls fix-review, which
// is also an artifact.
func TestPlanExecutePlacesOnAllTargets(t *testing.T) {
	// Global skill root per target, from adapters/<target>.yaml `skill.global`.
	for _, tc := range []struct {
		target   string
		skillDir []string // path segments under the temp HOME
	}{
		{"claude", []string{".claude", "skills"}},
		{"codex", []string{".agents", "skills"}},
		{"opencode", []string{".config", "opencode", "skills"}},
	} {
		t.Run(tc.target, func(t *testing.T) {
			f := serveFixtureFrom(t, fixtureSkillBundle(t))
			home := withRemoteEnv(t, f)

			if _, errOut, err := runInstall(t,
				"fix-router", "--target", tc.target, "--global", "--deploy", "--yes"); err != nil {
				t.Fatalf("install fix-router for %s: %v\n%s", tc.target, err, errOut)
			}

			skills := filepath.Join(append([]string{home}, tc.skillDir...)...)
			if _, err := os.Stat(filepath.Join(skills, "fix-router", "SKILL.md")); err != nil {
				t.Errorf("fix-router not placed on %s: %v", tc.target, err)
			}
			// The sidecars the router loads must be packed, or the mode it picks is a
			// dangling reference.
			for _, rel := range []string{"mode.md", "NOTICE", filepath.Join("scripts", "helper")} {
				if _, err := os.Stat(filepath.Join(skills, "fix-router", rel)); err != nil {
					t.Errorf("fix-router sidecar %q not packed on %s: %v", rel, tc.target, err)
				}
			}
		})
	}
}

// TestPlanExecuteRequiresClosure proves the requires: edge on a STANDALONE
// install: asking for fix-router alone also installs fix-review,
// whose review.md both modes fill for the final whole-branch review. An
// unresolved edge here means the review step points at a file that is not there.
func TestPlanExecuteRequiresClosure(t *testing.T) {
	f := serveFixtureFrom(t, fixtureSkillBundle(t))
	home := withRemoteEnv(t, f)

	out, errOut, err := runInstall(t, "fix-router", "--target", "claude", "--global", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run install fix-router: %v\n%s", err, errOut)
	}
	if !strings.Contains(errOut, "fix-review") && !strings.Contains(out, "fix-review") {
		t.Errorf("the requires closure should pull fix-review:\nstdout:\n%s\nstderr:\n%s", out, errOut)
	}

	if _, e, err := runInstall(t, "fix-router", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install fix-router: %v\n%s", err, e)
	}
	p := filepath.Join(home, ".claude", "skills", "fix-review", "review.md")
	if _, err := os.Stat(p); err != nil {
		t.Errorf("review.md not placed by the closure at %s: %v", p, err)
	}
}
