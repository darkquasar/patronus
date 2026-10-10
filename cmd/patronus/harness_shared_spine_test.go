package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/state"
)

// TestHarnessTargetsUseSharedInstallSpine (OC-R10): no target has its own
// install command, and an invented profile (a skill plus a setting) installed
// with --target <t> --local --deploy records state items under that tool, and
// the ordinary lock command writes a v3 lock naming it.
func TestHarnessTargetsUseSharedInstallSpine(t *testing.T) {
	for _, tc := range []struct {
		target, skillRel, settingFile, settingPath string
	}{
		{"codex", ".agents/skills/fixture-spine/SKILL.md", ".codex/config.toml", "invented.flag"},
		{"opencode", ".opencode/skills/fixture-spine/SKILL.md", "opencode.json", "invented.flag"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			root := newRootCmd()
			for _, cmd := range root.Commands() {
				for _, name := range append([]string{cmd.Name()}, cmd.Aliases...) {
					if name == "install-"+tc.target {
						t.Fatalf("%s has its own install command", tc.target)
					}
				}
			}
			project := fixtureCatalog(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
			t.Chdir(project)
			codexWrite(t, filepath.Join(project, "artifacts/skills/fixture-spine/patronus.yaml"), []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: fixture-spine\ndescription: \"Invented spine skill.\"\nversion: 1.0.0\nentry: SKILL.md\ntargets: ["+tc.target+"]\ndefaults:\n  scope: project\n"))
			codexWrite(t, filepath.Join(project, "artifacts/skills/fixture-spine/SKILL.md"), []byte("---\nname: fixture-spine\ndescription: invented spine skill\n---\nbody\n"))
			codexWrite(t, filepath.Join(project, "artifacts/settings/fixture-spine-setting/patronus.yaml"), []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: setting\nrole: guardrail\nname: fixture-spine-setting\ndescription: \"Invented spine setting.\"\nversion: 1.0.0\ntargets: ["+tc.target+"]\ndefaults:\n  scope: project\nsetting:\n  path: "+tc.settingPath+"\n  value: true\n"))
			codexWrite(t, filepath.Join(project, "profiles/fixture-spine-profile.yaml"), []byte("apiVersion: patronus/v2\nfamily: profile\nname: fixture-spine-profile\nversion: 1.0.0\nrole: lifecycle\ntargets: ["+tc.target+"]\nlayers:\n  capabilities: [fixture-spine]\n  guardrails: [fixture-spine-setting]\n"))
			out, errOut, err := runInstall(t, "--profile", "fixture-spine-profile", "--target", tc.target, "--local", "--deploy")
			if err != nil {
				t.Fatalf("install: %v\n%s\n%s", err, out, errOut)
			}
			if _, err := os.Stat(filepath.Join(project, tc.skillRel)); err != nil {
				t.Fatalf("skill not placed: %v", err)
			}
			s, err := state.Load(filepath.Join(project, ".patronus/state.json"))
			if err != nil {
				t.Fatal(err)
			}
			tools := map[string]int{}
			for _, it := range s.Items {
				tools[it.Tool]++
			}
			if tools[tc.target] != 2 || len(tools) != 1 {
				t.Fatalf("state items by tool: %v", tools)
			}
			if out, errOut, err := runLock(t, "--profile", "fixture-spine-profile", "--target", tc.target, "--local-registry"); err != nil {
				t.Fatalf("lock: %v\n%s\n%s", err, out, errOut)
			}
			l, err := lock.Load(filepath.Join(project, "patronus.lock"))
			if err != nil {
				t.Fatal(err)
			}
			if l.Version != lock.TargetVersion || l.Target != tc.target {
				t.Fatalf("lock = version %d target %q", l.Version, l.Target)
			}
		})
	}
}
