package adapter_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/plan"
	"github.com/darkquasar/patronus/internal/remove"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func runnerAdapter(t *testing.T) *manifest.Adapter {
	t.Helper()
	ad, err := manifest.DecodeAdapter([]byte(`apiVersion: patronus/v2
family: adapter
tool: toy
layout:
  hook:
    projectScriptDir: ".toy/patronus/hooks"
    projectRunnerDir: ".toy/plugins"
    project: {file: ".toy/patronus/hooks.json", format: json, path: "hooks.{event}", action: merge}
    runnerDialect:
      schemaVersion: 1
      toolMap: {write: Write, edit: Edit, apply_patch: apply_patch, patch: apply_patch, bash: Bash, read: Read, grep: Grep, glob: Glob, webfetch: WebFetch, websearch: WebSearch, task: Task, todowrite: TodoWrite, skill: Skill, invented: Invented}
      argMap:
        apply_patch: {patchText: patch}
        write: {filePath: file_path, content: content}
        edit: {filePath: file_path, oldString: old_string, newString: new_string}
        bash: {command: command}
        invented: {source: destination}
`))
	if err != nil {
		t.Fatal(err)
	}
	return ad
}

func runnerDiffs(t *testing.T, root, name string) []diff.FileDiff {
	t.Helper()
	src := t.TempDir()
	body, err := os.ReadFile("testdata/opencode-runner/fixture-guard.sh")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "guard.sh"), body, 0644); err != nil {
		t.Fatal(err)
	}
	art := &manifest.Artifact{Meta: manifest.Meta{Name: name, Version: "1.0.0", Family: manifest.FamilyArtifact, Role: manifest.RoleGuardrail}, Type: manifest.TypeHook, Hook: &manifest.HookSpec{Event: "PreToolUse", Matcher: "Write|Edit|apply_patch|Bash|Invented", Intent: manifest.HookGate, Script: "guard.sh", Command: `sh "{script}" "` + filepath.Join(root, "capture.json") + `"`}}
	eng := adapter.New(toolpath.New(func(string) (string, bool) { return "", false }, t.TempDir(), root))
	ds, err := eng.Transform(art, runnerAdapter(t), "local", src, func(string) ([]byte, bool, error) { return nil, false, nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 3 {
		t.Fatalf("want script, registry, runner: %+v", ds)
	}
	return ds
}

func TestHookRunnerDeterministicAndSharedLifecycle(t *testing.T) {
	root := t.TempDir()
	alpha, beta := runnerDiffs(t, root, "alpha"), runnerDiffs(t, root, "beta")
	if !bytes.Equal(alpha[2].After, beta[2].After) {
		t.Fatal("runner bytes differ between hooks")
	}
	absent := func(string) ([]byte, bool, error) { return nil, false, nil }
	cs, err := plan.Finalize(append(alpha, beta...), absent)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, d := range cs.Diffs {
		files[d.Path] = d.After
		if d.Path == alpha[2].Path && (len(d.CreateContrib) != 1 || d.CreateContrib[0].Artifact != "beta") {
			t.Fatalf("missing runner contributor: %+v", d)
		}
	}
	items := state.FromChangeSet(cs.Diffs, "2026-01-01T00:00:00Z")
	if len(items) != 2 {
		t.Fatalf("items: %+v", items)
	}
	occ := remove.Occupancy{}
	for _, it := range items {
		for _, f := range it.Files {
			occ[f.Path] = append(occ[f.Path], remove.Contributor{Artifact: it.Artifact, Tool: it.Tool, Scope: it.Scope, Created: f.Action == "CREATE", Setting: f.Setting})
		}
	}
	read := func(p string) ([]byte, bool, error) { b, ok := files[p]; return b, ok, nil }
	first, err := remove.Compute(items[:1], read, occ)
	if err != nil {
		t.Fatal(err)
	}
	apply := func(cs *diff.ChangeSet) {
		for _, d := range cs.Diffs {
			switch d.Action {
			case diff.Delete:
				delete(files, d.Path)
			case diff.Restore, diff.Merge:
				files[d.Path] = d.After
			}
		}
	}
	retained := false
	for _, e := range first.Ledger {
		if e.Path == alpha[2].Path && e.Outcome == remove.SharedRetained {
			retained = true
		}
	}
	if !retained {
		t.Fatalf("runner not retained: %+v", first.Ledger)
	}
	apply(first.ChangeSet)
	if _, ok := files[alpha[2].Path]; !ok {
		t.Fatal("non-last removal deleted runner")
	}
	for p, cs := range occ {
		var kept []remove.Contributor
		for _, c := range cs {
			if c.Artifact != items[0].Artifact {
				kept = append(kept, c)
			}
		}
		occ[p] = kept
	}
	last, err := remove.Compute(items[1:], read, occ)
	if err != nil {
		t.Fatal(err)
	}
	apply(last.ChangeSet)
	if _, ok := files[alpha[2].Path]; ok {
		t.Fatal("last removal retained runner")
	}
	raw := files[alpha[1].Path]
	var registry struct {
		Hooks map[string][]any `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &registry); err != nil {
		t.Fatal(err)
	}
	if len(registry.Hooks["PreToolUse"]) != 0 {
		t.Fatalf("registrations survived removal: %s", raw)
	}
	eng := adapter.New(toolpath.New(func(string) (string, bool) { return "", false }, t.TempDir(), root))
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "guard.sh"), []byte("exit 0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	art := &manifest.Artifact{Meta: manifest.Meta{Name: "alpha"}, Type: manifest.TypeHook, Hook: &manifest.HookSpec{Event: "PreToolUse", Matcher: "Write", Intent: manifest.HookGate, Script: "guard.sh", Command: "sh {script}"}}
	ds, err := eng.Transform(art, runnerAdapter(t), "local", src, read)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Finalize(ds, read); err != nil {
		t.Fatalf("reinstall: %v", err)
	}
}

func TestHookRunnerNodeFixtures(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node >= 22.18 with type stripping not on PATH")
	}
	version, err := exec.Command(node, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	var major, minor int
	if _, err := fmt.Sscanf(string(version), "v%d.%d", &major, &minor); err != nil {
		t.Fatal(err)
	}
	if major < 22 || major == 22 && minor < 18 {
		t.Skip("node >= 22.18 with type stripping not on PATH")
	}
	probe := filepath.Join(t.TempDir(), "probe.ts")
	if err := os.WriteFile(probe, []byte("const value: number = 1; export default value;\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(node, "--experimental-strip-types", "--eval", `import(require("node:url").pathToFileURL(process.argv[1]).href)`, probe).CombinedOutput(); err != nil {
		t.Skipf("node >= 22.18 with type stripping not on PATH (actual .ts import probe): %v %s", err, out)
	}
	for _, tc := range []struct {
		name, tool, args, matcher, command, message string
		blocked, capture                            bool
	}{
		{"patch benign", "apply_patch", `{"patchText":"*** Begin Patch\n*** Add File: demo.txt\n+benign fixture\n*** End Patch"}`, "apply_patch", "", "", false, true},
		{"patch blocked", "apply_patch", `{"patchText":"*** Begin Patch\n*** Add File: demo.txt\n+PLANTED_FIXTURE_SECRET\n*** End Patch"}`, "apply_patch", "", "fixture guard blocked", true, true},
		{"edit", "edit", `{"filePath":"demo","oldString":"old","newString":"new"}`, "Edit", "", "", false, true},
		{"write", "write", `{"filePath":"demo","content":"benign"}`, "Write", "", "", false, true},
		{"edit blocked", "edit", `{"newString":"PLANTED_FIXTURE_SECRET"}`, "Edit", "", "fixture guard blocked", true, true},
		{"stdout discarded", "edit", `{}`, "Edit", "printf PLANTED_FIXTURE_SECRET", "", false, false},
		{"bash", "bash", `{"command":"echo benign"}`, "Bash", "", "", false, true},
		{"injected dialect", "invented", `{"source":"value","unlisted":1}`, "Invented", "", "", false, true},
		{"exact matcher", "edit", `{"newString":"PLANTED_FIXTURE_SECRET"}`, "MultiEdit", "", "", false, false},
		{"unmapped tool", "unknown", `{"value":"PLANTED_FIXTURE_SECRET"}`, "", "", "", false, false},
		{"unmapped prototype tool", "toString", `{}`, "", "", "", false, false},
		{"unlisted prototype argument", "edit", `{"toString":"benign"}`, "Edit", "", "", false, true},
		{"missing command", "edit", `{}`, "Edit", "/invented/absent-command", "call blocked", true, false},
		{"other exit", "edit", `{}`, "Edit", "exit 3", "call blocked", true, false},
		{"timeout", "edit", `{}`, "Edit", "sleep 2", "call blocked", true, false},
		{"empty stderr", "edit", `{}`, "Edit", "exit 2", "blocked by fixture", true, false},
		{"missing registry", "edit", `{}`, "Edit", "MISSING", "", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			ds := runnerDiffs(t, root, "fixture")
			for _, d := range ds {
				if err := os.MkdirAll(filepath.Dir(d.Path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(d.Path, d.After, 0644); err != nil {
					t.Fatal(err)
				}
			}
			command := `sh "` + ds[0].Path + `" "` + filepath.Join(root, "capture.json") + `"`
			if tc.command != "" {
				command = tc.command
			}
			if tc.command == "MISSING" {
				if err := os.Remove(ds[1].Path); err != nil {
					t.Fatal(err)
				}
			} else {
				timeout := 60.0
				if tc.name == "timeout" {
					timeout = 0.01
				}
				registry := map[string]any{"hooks": map[string]any{"PreToolUse": []any{map[string]any{"matcher": tc.matcher, "patronusId": "fixture", "hooks": []any{map[string]any{"type": "command", "command": command, "timeout": timeout}}}}}}
				b, _ := json.Marshal(registry)
				if err := os.WriteFile(ds[1].Path, b, 0644); err != nil {
					t.Fatal(err)
				}
			}
			payload := []byte(`{"input":{"tool":"` + tc.tool + `","sessionID":"fixture-session"},"output":{"args":` + tc.args + `}}`)
			if tc.name == "patch benign" {
				payload, err = os.ReadFile("testdata/opencode-runner/payloads/apply-patch.json")
				if err != nil {
					t.Fatal(err)
				}
			}
			p := filepath.Join(root, "payload.json")
			if err := os.WriteFile(p, payload, 0644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(node, "--experimental-strip-types", "testdata/opencode-runner/run.mjs", ds[2].Path, p, root)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			stdout, err := cmd.Output()
			if err != nil {
				t.Fatalf("node: %v %s", err, stderr.String())
			}
			var result struct {
				Blocked bool   `json:"blocked"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(stdout, &result); err != nil {
				t.Fatalf("output: %s: %v", stdout, err)
			}
			if result.Blocked != tc.blocked || !strings.Contains(result.Message, tc.message) {
				t.Fatalf("outcome %+v, want blocked=%v message=%q", result, tc.blocked, tc.message)
			}
			if bytes.Contains(stdout, []byte("PLANTED_FIXTURE_SECRET")) || bytes.Contains(stderr.Bytes(), []byte("PLANTED_FIXTURE_SECRET")) {
				t.Fatal("payload echoed")
			}
			if tc.command == "MISSING" && (strings.Count(stderr.String(), "no guards active") != 1 || strings.Count(stderr.String(), "patronus-hook-runner:") != 1) {
				t.Fatalf("notice: %q", stderr.String())
			}
			capture, err := os.ReadFile(filepath.Join(root, "capture.json"))
			if !tc.capture {
				if err == nil {
					t.Fatalf("unmatched call ran guard: %s", capture)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				Tool    string         `json:"tool_name"`
				Args    map[string]any `json:"tool_input"`
				Session string         `json:"session_id"`
				Cwd     string         `json:"cwd"`
				Event   string         `json:"hook_event_name"`
			}
			if err := json.Unmarshal(capture, &got); err != nil {
				t.Fatal(err)
			}
			ad := runnerAdapter(t)
			if got.Tool != ad.Layout.Hook.RunnerDialect.ToolMap[tc.tool] || got.Session != "fixture-session" || got.Cwd != root || got.Event != "PreToolUse" {
				t.Fatalf("payload: %s", capture)
			}
			var original struct {
				Output struct {
					Args map[string]any `json:"args"`
				} `json:"output"`
			}
			if err := json.Unmarshal(payload, &original); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{}
			for k, v := range original.Output.Args {
				mapped := ad.Layout.Hook.RunnerDialect.ArgMap[tc.tool][k]
				if mapped == "" {
					mapped = k
				}
				want[mapped] = v
			}
			a, _ := json.Marshal(got.Args)
			b, _ := json.Marshal(want)
			if !bytes.Equal(a, b) {
				t.Fatalf("args: %s want %s", a, b)
			}
		})
	}
}
