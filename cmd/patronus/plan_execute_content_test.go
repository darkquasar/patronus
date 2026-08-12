package main

import (
	"os"
	"strings"
	"testing"
)

// TestSoloModeEndsWithIndependentReview pins the load-bearing property of the
// two-mode design. A third "reviewed-solo" mode was considered and rejected on
// the grounds that the whole-branch review is independent in BOTH modes. If
// solo's final review ever became a self-review, that rejected third mode would
// reappear as a real gap, so this is the assertion that makes two modes valid.
func TestSoloModeEndsWithIndependentReview(t *testing.T) {
	b, err := os.ReadFile("../../artifacts/skills/plan-execute/solo.md")
	if err != nil {
		t.Fatalf("read solo.md: %v", err)
	}
	body := string(b)

	for _, want := range []string{
		"requesting-code-review",
		"code-reviewer.md",
		"Do not review your own work",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("solo.md must end with an independent whole-branch review; missing %q", want)
		}
	}
}

// TestSDDModeSafetyRules pins the three ways this sdd.md is deliberately not a
// copy of upstream's subagent-driven-development. Each is a correctness rule the
// spec calls out, and each is invisible to any structural check, so it gets a
// content assertion.
func TestSDDModeSafetyRules(t *testing.T) {
	b, err := os.ReadFile("../../artifacts/skills/plan-execute/sdd.md")
	if err != nil {
		t.Fatalf("read sdd.md: %v", err)
	}
	body := string(b)

	cases := []struct {
		rule string
		want string
	}{
		{"three-round fix-loop ceiling", "three failed review rounds"},
		{"prior-task contracts in the brief", "contracts established by earlier tasks"},
		{"stop and report on dispatch failure", "Stop and report"},
		{"reviewer context is never the transcript", "never from the implementer's transcript"},
	}
	for _, c := range cases {
		if !strings.Contains(body, c.want) {
			t.Errorf("sdd.md is missing the %s rule (want substring %q)", c.rule, c.want)
		}
	}
}

// TestSDDAuxFilesCarriedOver proves the prompt templates and script helpers the
// sdd mode invokes actually ship in this artifact's directory, executable where
// they need to be. sdd.md references them by relative path, so an unpacked helper
// is a broken skill, not a missing nicety.
func TestSDDAuxFilesCarriedOver(t *testing.T) {
	dir := "../../artifacts/skills/plan-execute/"
	for _, rel := range []string{"implementer-prompt.md", "task-reviewer-prompt.md"} {
		if _, err := os.Stat(dir + rel); err != nil {
			t.Errorf("prompt template %q not carried over: %v", rel, err)
		}
	}
	for _, rel := range []string{"scripts/task-brief", "scripts/review-package", "scripts/sdd-workspace"} {
		fi, err := os.Stat(dir + rel)
		if err != nil {
			t.Errorf("script helper %q not carried over: %v", rel, err)
			continue
		}
		if fi.Mode().Perm()&0o111 == 0 {
			t.Errorf("script helper %q mode = %v, want executable", rel, fi.Mode().Perm())
		}
	}
}

// TestPlanReviewForkNamesBothArms pins that plan-review's fork keeps BOTH arms
// after being rewritten. The fork's criterion is PARALLELISM and is unchanged;
// only the destination names move, and the proportionality gate is noted as
// living inside the non-parallel arm. A one-arm pointer would collapse two
// orthogonal axes into one.
func TestPlanReviewForkNamesBothArms(t *testing.T) {
	b, err := os.ReadFile("../../artifacts/skills/plan-review/SKILL.md")
	if err != nil {
		t.Fatalf("read plan-review SKILL.md: %v", err)
	}
	body := string(b)

	for _, want := range []string{
		"plan-execute-parallel",
		"disjoint file-owning boundaries",
		"proportionality",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("plan-review fork missing %q", want)
		}
	}
	// The old destinations must be gone from the fork.
	for _, unwanted := range []string{"`executing-plans` (solo)", "`team-implement` (parallel team)"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("plan-review still routes to the retired destination %q", unwanted)
		}
	}
}

// TestWritingPlansHandsOffToPlanExecute pins the hand-off rename AND the
// reviewer-dispatch fix from 79f9b64 that must survive it. Task 1 reverted that
// commit's executing-plans half; its writing-plans half is a wanted fix, and
// editing the neighbouring hand-off text is exactly where it could be lost.
func TestWritingPlansHandsOffToPlanExecute(t *testing.T) {
	b, err := os.ReadFile("../../artifacts/skills/writing-plans/SKILL.md")
	if err != nil {
		t.Fatalf("read writing-plans SKILL.md: %v", err)
	}
	body := string(b)

	if !strings.Contains(body, "plan-execute") {
		t.Error("writing-plans must hand off to plan-execute")
	}
	if strings.Contains(body, "executing-plans") {
		t.Error("writing-plans still names executing-plans; the hand-off moved to plan-execute")
	}
	// The 79f9b64 reviewer-dispatch fix stays.
	if !strings.Contains(body, "plan-review") {
		t.Error("the plan-review reviewer-dispatch pointer from 79f9b64 was lost")
	}
}
