//go:build artifactbehavior

package main

import (
	"github.com/darkquasar/patronus/internal/registry"
	toml "github.com/pelletier/go-toml/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexGuardInstallScriptAndRegistration(t *testing.T) {
	realRoot, err := registry.DiscoverRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, scope := range []string{"global", "local"} {
		t.Run(scope, func(t *testing.T) {
			root := fixtureCatalog(t)
			name := "block-secrets-cx"
			copyDir(t, filepath.Join(realRoot, "artifacts", "hooks", name), filepath.Join(root, "artifacts", "hooks", name))
			home, config := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("CODEX_HOME", config)
			t.Setenv("PATRONUS_REGISTRY_URL", "")
			t.Chdir(root)
			out, errOut, err := runInstall(t, name, "--target", "codex", "--"+scope, "--deploy", "--yes")
			if err != nil {
				t.Fatalf("install: %v\n%s\n%s", err, out, errOut)
			}
			base := config
			if scope == "local" {
				base = filepath.Join(root, ".codex")
			}
			script := filepath.Join(base, "hooks", name+".py")
			info, err := os.Stat(script)
			if err != nil || info.Mode().Perm() != 0755 {
				t.Fatalf("placed executable: %v %v", info, err)
			}
			var settings map[string]any
			if err := toml.Unmarshal(mustRead(t, filepath.Join(base, "config.toml")), &settings); err != nil {
				t.Fatal(err)
			}
			groups := settings["hooks"].(map[string]any)["PreToolUse"].([]any)
			if len(groups) != 1 {
				t.Fatalf("registration: %#v", groups)
			}
			group := groups[0].(map[string]any)
			handler := group["hooks"].([]any)[0].(map[string]any)
			if handler["command"] != "\""+script+"\"" || group["matcher"] != "Write|Edit|MultiEdit|apply_patch" {
				t.Fatalf("registration: %#v", group)
			}
			code, output := runCodexGuard(t, script, home, root, guardPayload(t, "Write", map[string]any{"content": "fixture-nonce"}))
			if code != 0 {
				t.Fatalf("installed script exit=%d output=%q", code, output)
			}
			out, errOut, err = runInstall(t, name, "--target", "codex", "--"+scope, "--dry-run")
			if err != nil || !strings.Contains(out, "SKIP") {
				t.Fatalf("reinstall: %v\n%s\n%s", err, out, errOut)
			}
		})
	}
}
