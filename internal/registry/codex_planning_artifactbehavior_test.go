//go:build artifactbehavior

package registry

// Artifact-behaviour fixture retained intact from codex_planning_test.go. It
// executes the shipped spec-brainstorming-cx validator program, so it is
// excluded from the default application gate and is NOT RUN there. Results
// are historical evidence, not current artifact-behaviour acceptance.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func codexPlanningValidator(t *testing.T, meta string, files []string) (string, error) {
	t.Helper()
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "meta.yaml"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(folder, name), []byte("invented fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "-mod=readonly", ".", folder)
	cmd.Dir = "../../artifacts/skills/spec-brainstorming-cx/scripts"
	cmd.Env = append(os.Environ(), "GOPROXY=off", "GOTOOLCHAIN=local")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestCodexSpecFolderMetaInvariants(t *testing.T) {
	meta := "research: invented-research.md\nstreams:\n  - slug: invented\n    spec: invented-spec.md\n    plan: null\n"
	t.Run("missing referenced research and plan", func(t *testing.T) {
		for _, field := range []string{"research", "plan"} {
			t.Run(field, func(t *testing.T) {
				fixture := "research: absent-research.md\nstreams: []\n"
				name := "absent-research.md"
				if field == "plan" {
					fixture = "research: null\nstreams: [{slug: absent, spec: null, spec_declined: 'owner decision', plan: absent-plan.md}]\n"
					name = "absent-plan.md"
				}
				out, err := codexPlanningValidator(t, fixture, nil)
				if err == nil || !strings.Contains(out, "missing referenced file: "+name) {
					t.Fatalf("expected missing reference rejection: %v\n%s", err, out)
				}
			})
		}
	})
	t.Run("quoted inline YAML and declined spec", func(t *testing.T) {
		out, err := codexPlanningValidator(t, "research: null\nstreams: [{slug: quick, spec: null, spec_declined: 'owner declined', plan: 'quick-plan.md'}]\n", []string{"quick-plan.md"})
		if err != nil || !strings.Contains(out, "ADR-0003 OK") {
			t.Fatalf("valid plan-only stream: %v\n%s", err, out)
		}
	})
	t.Run("malformed YAML", func(t *testing.T) {
		out, err := codexPlanningValidator(t, "streams: [\n", nil)
		if err == nil || !strings.Contains(out, "parse meta.yaml") {
			t.Fatalf("expected parse rejection: %v\n%s", err, out)
		}
	})
	t.Run("boolean completeness and traversal refused", func(t *testing.T) {
		for _, value := range []string{"true", "../outside.md"} {
			out, err := codexPlanningValidator(t, "research: "+value+"\n", nil)
			if err == nil || !strings.Contains(out, "invalid document filename") {
				t.Fatalf("expected invalid filename rejection: %v\n%s", err, out)
			}
		}
	})
	t.Run("valid YAML ignores filenames in comments", func(t *testing.T) {
		out, err := codexPlanningValidator(t, meta+"# absent-plan.md is not a reference\n", []string{"invented-research.md", "invented-spec.md"})
		if err != nil || !strings.Contains(out, "ADR-0003 OK") {
			t.Fatalf("valid folder: %v\n%s", err, out)
		}
	})
	t.Run("missing referenced spec", func(t *testing.T) {
		out, err := codexPlanningValidator(t, meta, []string{"invented-research.md"})
		if err == nil || !strings.Contains(out, "missing referenced file: invented-spec.md") {
			t.Fatalf("expected missing-file rejection: %v\n%s", err, out)
		}
	})
	t.Run("orphan spec and plan", func(t *testing.T) {
		for _, name := range []string{"orphan-spec.md", "orphan-plan.md"} {
			t.Run(name, func(t *testing.T) {
				out, err := codexPlanningValidator(t, meta, []string{"invented-research.md", "invented-spec.md", name})
				if err == nil || !strings.Contains(out, "orphan spec/plan file: "+name) {
					t.Fatalf("expected orphan rejection: %v\n%s", err, out)
				}
			})
		}
	})
}
