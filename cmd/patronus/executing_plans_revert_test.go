package main

import (
	"os"
	"strings"
	"testing"
)

// TestExecutingPlansHedgeRestored pins the partial revert of 79f9b64 on the
// executing-plans artifact. The two lines that commit rewrote turned upstream's
// capability HEDGE into an instruction, which erased the boundary between this
// skill and subagent-driven-development. plan-execute now owns that fork, so the
// hedge belongs back here.
//
// This artifact is ADAPTED, not verbatim-vendored: the second half of the test
// pins the Patronus tk-close re-coupling recorded in its attribution.note, so a
// future "restore upstream verbatim" edit reds instead of silently dropping it.
func TestExecutingPlansHedgeRestored(t *testing.T) {
	b, err := os.ReadFile("../../artifacts/skills/executing-plans/SKILL.md")
	if err != nil {
		t.Fatalf("read executing-plans SKILL.md: %v", err)
	}
	body := string(b)

	for _, want := range []string{
		"This discipline works much better with access to subagents.",
		"If your platform supports dispatching a fresh subagent per task with review between tasks, prefer that",
		"If your platform supports it, dispatch a fresh subagent per task and review between tasks instead of executing inline.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("pre-79f9b64 hedge text missing:\n  want substring: %q", want)
		}
	}

	// The 79f9b64 rewrite's phrasing must be gone from both sites.
	for _, unwanted := range []string{
		"This discipline depends on fresh context per task.",
		"**If subagents are available, dispatch one per task**",
		"**If subagents are available, dispatch a fresh subagent per task and review between tasks**",
	} {
		if strings.Contains(body, unwanted) {
			t.Errorf("79f9b64 rewrite text still present: %q", unwanted)
		}
	}

	// The Patronus adaptation is NOT reverted: tk close stays.
	if !strings.Contains(body, "tk close <id>") {
		t.Error("the Patronus tk-close re-coupling was dropped by the revert; it must survive")
	}
}
