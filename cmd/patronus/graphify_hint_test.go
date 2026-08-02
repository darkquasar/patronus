package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/registry"
)

// The graphify-hint nudge fires on a SEARCH: a Bash grep/rg/find command, or the
// native Grep/Glob tools (which carry the search term in .tool_input.pattern). The
// manifest inlines its logic; artifacts/hooks/graphify-hint/graphify-hint.sh is the
// reference form kept in sync with it, and is what this test executes. The nudge
// fails open (exit 0) and only speaks when a graph exists and graphify is on PATH,
// so the test stages both in a temp workdir.
func TestGraphifyHintFiresOnSearchPayloads(t *testing.T) {
	root, err := registry.DiscoverRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "artifacts", "hooks", "graphify-hint", "graphify-hint.sh")

	// Stage the two preconditions the hint checks: a graph file (relative to cwd)
	// and a graphify binary on PATH. Without both, the hint stays silent by design.
	work := t.TempDir()
	if err := os.MkdirAll(filepath.Join(work, "graphify-out"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "graphify-out", "graph.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(work, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "graphify"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		payload  string
		wantHint bool
	}{
		{"bash grep", `{"tool_input":{"command":"grep -r foo ."}}`, true},
		{"bash rg", `{"tool_input":{"command":"rg needle"}}`, true},
		{"bash find", `{"tool_input":{"command":"find . -name '*.go'"}}`, true},
		{"native Grep tool", `{"tool_input":{"pattern":"func main"}}`, true},
		{"native Glob tool", `{"tool_input":{"pattern":"**/*.go"}}`, true},
		{"bash non-search stays silent", `{"tool_input":{"command":"ls -la"}}`, false},
		{"empty input stays silent", `{"tool_input":{}}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.CommandContext(context.Background(), "bash", script)
			cmd.Dir = work
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			cmd.Stdin = strings.NewReader(tt.payload)
			var stderr strings.Builder
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("hint script must always exit 0 (fail open), got %v\n%s", err, stderr.String())
			}
			gotHint := strings.Contains(stderr.String(), "graphify query")
			if gotHint != tt.wantHint {
				t.Errorf("hint fired = %v, want %v (stderr: %q)", gotHint, tt.wantHint, stderr.String())
			}
		})
	}
}
