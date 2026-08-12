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
