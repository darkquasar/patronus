package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/state"
	toml "github.com/pelletier/go-toml/v2"
)

// The invented auth material below is a reference to an environment variable
// name and an opaque header label. No live secret is used or persisted.
const (
	codexMCPServer   = "fix-mcp-two"
	codexMCPOldURL   = "https://fixture.invalid/mcp/"
	codexMCPNewURL   = "https://fixture.invalid/mcp/v2"
	codexMCPTokenEnv = "INVENTED_TOKEN_ENV"
)

var codexMCPTarget = manifest.FileTarget{File: "~/.codex/config.toml", Format: "toml"}

// codexMCPFixture installs a fixture catalog with Codex isolated under a temp
// HOME and seeds unrelated user TOML siblings.
func codexMCPFixture(t *testing.T) (root, home, config string) {
	t.Helper()
	root = fixtureCatalog(t)
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Chdir(root)
	config = filepath.Join(home, ".codex/config.toml")
	codexWrite(t, config, []byte("model = 'user-model'\n[profiles.invented]\nmodel = 'user-profile-model'\n"))
	return root, home, config
}

// codexAddUserAuth simulates the user adding opaque auth/header/env children
// to the managed server table after Patronus wired it.
func codexAddUserAuth(t *testing.T, config string) {
	t.Helper()
	doc := map[string]any{}
	if err := toml.Unmarshal(mustRead(t, config), &doc); err != nil {
		t.Fatal(err)
	}
	server := doc["mcp_servers"].(map[string]any)[codexMCPServer].(map[string]any)
	server["bearer_token_env_var"] = codexMCPTokenEnv
	server["http_headers"] = map[string]any{"X-Invented": "fixture-reference"}
	server["env_http_headers"] = map[string]any{"X-Invented-Env": "INVENTED_HEADER_ENV"}
	out, err := toml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	codexWrite(t, config, out)
}

func codexAssertUserFields(t *testing.T, config string) {
	t.Helper()
	raw := mustRead(t, config)
	for key, want := range map[string]any{
		"model":                   "user-model",
		"profiles.invented.model": "user-profile-model",
		"mcp_servers." + codexMCPServer + ".bearer_token_env_var":            codexMCPTokenEnv,
		"mcp_servers." + codexMCPServer + ".http_headers.X-Invented":         "fixture-reference",
		"mcp_servers." + codexMCPServer + ".env_http_headers.X-Invented-Env": "INVENTED_HEADER_ENV",
	} {
		got, present, err := adapter.ReadDotted(raw, codexMCPTarget, key)
		if err != nil || !present || !adapter.SettingValuesEqual(got, want) {
			t.Fatalf("user field %s = %v (present %t, err %v), want %v:\n%s", key, got, present, err, want, raw)
		}
	}
}

func codexMCPURL(t *testing.T, config string) (any, bool) {
	t.Helper()
	got, present, err := adapter.ReadDotted(mustRead(t, config), codexMCPTarget, "mcp_servers."+codexMCPServer+".url")
	if err != nil {
		t.Fatal(err)
	}
	return got, present
}

// codexMCPOwned returns the recorded managed leaves for the fixture server.
func codexMCPOwned(t *testing.T, home string) []*diff.SettingEdit {
	t.Helper()
	s, err := state.Load(filepath.Join(home, ".patronus/state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out []*diff.SettingEdit
	for _, it := range s.Items {
		if it.Artifact != codexMCPServer || it.Tool != "codex" {
			continue
		}
		for _, f := range it.Files {
			if f.Setting != nil {
				out = append(out, f.Setting)
			}
		}
	}
	return out
}

func codexBumpMCPRecipe(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "recipes", codexMCPServer+".yaml")
	body := string(mustRead(t, path))
	body = strings.ReplaceAll(body, "version: 1.0.0", "version: 2.0.0")
	body = strings.ReplaceAll(body, "url: \""+codexMCPOldURL+"\"", "url: \""+codexMCPNewURL+"\"")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCodexMCPReinstallUpdatePreservesUserAuth(t *testing.T) {
	root, home, config := codexMCPFixture(t)
	if _, _, err := runInstall(t, codexMCPServer, "--target", "codex", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
	owned := codexMCPOwned(t, home)
	if len(owned) != 1 || owned[0].Dotted != "mcp_servers."+codexMCPServer+".url" || owned[0].PriorPresent {
		t.Fatalf("fresh Codex wiring must own only the transport leaf with an absent prior: %+v", owned)
	}
	codexAddUserAuth(t, config)

	if _, _, err := runInstall(t, codexMCPServer, "--target", "codex", "--global", "--deploy"); err != nil {
		t.Fatalf("reinstall with user auth refused: %v", err)
	}
	codexAssertUserFields(t, config)
	if url, _ := codexMCPURL(t, config); url != codexMCPOldURL {
		t.Fatalf("reinstall url = %v", url)
	}

	codexBumpMCPRecipe(t, root)
	if _, _, err := runUpdate(t, codexMCPServer, "--deploy"); err != nil {
		t.Fatalf("update with user auth refused: %v", err)
	}
	codexAssertUserFields(t, config)
	if url, _ := codexMCPURL(t, config); url != codexMCPNewURL {
		t.Fatalf("update did not change managed leaf: %v", url)
	}
	owned = codexMCPOwned(t, home)
	if len(owned) != 1 || owned[0].PriorPresent || owned[0].PriorValue != nil || owned[0].ScalarValue != codexMCPNewURL {
		t.Fatalf("managed leaf prior must stay immutable across update: %+v", owned)
	}
	if strings.Contains(string(mustRead(t, filepath.Join(home, ".patronus/state.json"))), codexMCPTokenEnv) {
		t.Fatal("user auth reference adopted into ownership state")
	}
}

func TestCodexMCPRemovalPreservesUserAuth(t *testing.T) {
	t.Run("user auth survives", func(t *testing.T) {
		root, home, config := codexMCPFixture(t)
		if _, _, err := runInstall(t, codexMCPServer, "--target", "codex", "--global", "--deploy"); err != nil {
			t.Fatal(err)
		}
		codexAddUserAuth(t, config)
		codexBumpMCPRecipe(t, root)
		if _, _, err := runUpdate(t, codexMCPServer, "--deploy"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := execRemove(t, codexMCPServer, "--target", "codex", "--deploy"); err != nil {
			t.Fatal(err)
		}
		codexAssertUserFields(t, config)
		if url, present := codexMCPURL(t, config); present {
			t.Fatalf("owned leaf survived removal (or prior resurrected): %v", url)
		}
		if owned := codexMCPOwned(t, home); len(owned) != 0 {
			t.Fatalf("ownership survived removal: %+v", owned)
		}
	})

	t.Run("no user fields leaves no empty server", func(t *testing.T) {
		_, _, config := codexMCPFixture(t)
		before := mustRead(t, config)
		if _, _, err := runInstall(t, codexMCPServer, "--target", "codex", "--global", "--deploy"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := execRemove(t, codexMCPServer, "--target", "codex", "--deploy"); err != nil {
			t.Fatal(err)
		}
		raw := mustRead(t, config)
		if _, present, err := adapter.ReadDotted(raw, codexMCPTarget, "mcp_servers."+codexMCPServer); err != nil || present {
			t.Fatalf("removal left an empty managed server table (%v):\n%s", err, raw)
		}
		for _, key := range []string{"model", "profiles.invented.model"} {
			want, _, _ := adapter.ReadDotted(before, codexMCPTarget, key)
			got, present, err := adapter.ReadDotted(raw, codexMCPTarget, key)
			if err != nil || !present || !adapter.SettingValuesEqual(got, want) {
				t.Fatalf("unrelated sibling %s lost:\n%s", key, raw)
			}
		}
	})

	t.Run("equal unowned server is not adopted", func(t *testing.T) {
		_, home, config := codexMCPFixture(t)
		user := []byte("model = 'user-model'\n[mcp_servers." + codexMCPServer + "]\nurl = '" + codexMCPOldURL + "'\nbearer_token_env_var = '" + codexMCPTokenEnv + "'\n")
		codexWrite(t, config, user)
		if _, _, err := runInstall(t, codexMCPServer, "--target", "codex", "--global", "--deploy"); err != nil {
			t.Fatal(err)
		}
		if owned := codexMCPOwned(t, home); len(owned) != 0 {
			t.Fatalf("equal user server adopted into ownership: %+v", owned)
		}
		_, _, _ = execRemove(t, codexMCPServer, "--target", "codex", "--deploy")
		if string(mustRead(t, config)) != string(user) {
			t.Fatalf("unowned user server changed:\n%s", mustRead(t, config))
		}
	})

	t.Run("incompatible unowned server blocks even with force", func(t *testing.T) {
		_, _, config := codexMCPFixture(t)
		user := []byte("[mcp_servers." + codexMCPServer + "]\nurl = 'https://user.invalid/mcp'\nbearer_token_env_var = '" + codexMCPTokenEnv + "'\n")
		codexWrite(t, config, user)
		if _, _, err := runInstall(t, codexMCPServer, "--target", "codex", "--global", "--deploy", "--force"); err == nil {
			t.Fatal("incompatible unowned server overwritten")
		}
		if string(mustRead(t, config)) != string(user) {
			t.Fatal("refusal mutated user config")
		}
	})

	t.Run("legacy whole-map ownership stays refused", func(t *testing.T) {
		_, home, config := codexMCPFixture(t)
		server := map[string]any{"url": codexMCPOldURL}
		legacy := []byte("[mcp_servers." + codexMCPServer + "]\nurl = '" + codexMCPOldURL + "'\nbearer_token_env_var = '" + codexMCPTokenEnv + "'\n")
		codexWrite(t, config, legacy)
		sp := filepath.Join(home, ".patronus/state.json")
		s := &state.State{Version: state.Version, Items: []state.Item{{
			Artifact: codexMCPServer, ItemVersion: "1.0.0", Type: string(manifest.ShapeWireOnly), Tool: "codex", Scope: "global",
			Files: []state.FileState{{Path: config, Action: string(diff.Merge), Checksum: shaState(legacy), Setting: &diff.SettingEdit{
				Target: diff.FileTargetRef{File: codexMCPTarget.File, Format: "toml"}, Dotted: "mcp_servers." + codexMCPServer, ScalarValue: server,
			}}},
		}}}
		if err := state.Save(sp, s); err != nil {
			t.Fatal(err)
		}
		ownership := mustRead(t, sp)
		for _, force := range []bool{false, true} {
			args := []string{codexMCPServer, "--target", "codex", "--global", "--deploy"}
			if force {
				args = append(args, "--force")
			}
			if _, _, err := runInstall(t, args...); err == nil {
				t.Fatalf("legacy whole-map ownership silently converted (force=%t)", force)
			}
			if string(mustRead(t, config)) != string(legacy) || string(mustRead(t, sp)) != string(ownership) {
				t.Fatalf("legacy refusal mutated config/state (force=%t)", force)
			}
		}
		_, _, _ = execRemove(t, codexMCPServer, "--target", "codex", "--deploy")
		if string(mustRead(t, config)) != string(legacy) {
			t.Fatal("legacy whole-map removal dropped user auth")
		}
	})
}

const codexStdioServer = "fix-mcp-stdio"

// codexStdioRecipe writes a wire-only stdio recipe into the fixture catalog.
// The command is an invented bare name; nothing is fetched or executed.
func codexStdioRecipe(t *testing.T, root, version, arg string) {
	t.Helper()
	body := `apiVersion: patronus/v2
family: recipe
name: ` + codexStdioServer + `
version: ` + version + `
role: tools
summary: "Invented stdio MCP fixture for Codex local lifecycle."
wire:
  method: merge
  actor: patronus
  mcp:
    transport: stdio
    command: "invented-mcp-server"
    args: ["` + arg + `"]
  tools: [codex]
`
	codexWrite(t, filepath.Join(root, "recipes", codexStdioServer+".yaml"), []byte(body))
}

func TestCodexMCPStdioLocalLifecyclePreservesUserEnv(t *testing.T) {
	prefix := "mcp_servers." + codexStdioServer + "."
	localTarget := manifest.FileTarget{File: ".codex/config.toml", Format: "toml"}
	read := func(t *testing.T, config, key string) (any, bool) {
		t.Helper()
		got, present, err := adapter.ReadDotted(mustRead(t, config), localTarget, key)
		if err != nil {
			t.Fatal(err)
		}
		return got, present
	}
	setup := func(t *testing.T) (root, config string) {
		root = fixtureCatalog(t)
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
		t.Chdir(root)
		codexStdioRecipe(t, root, "1.0.0", "serve")
		config = filepath.Join(root, ".codex/config.toml")
		codexWrite(t, config, []byte("model = 'user-model'\n[profiles.invented]\nmodel = 'user-profile-model'\n[mcp_servers.user-other]\ncommand = 'user-other-cmd'\n"))
		return root, config
	}
	assertUnrelated := func(t *testing.T, config string) {
		t.Helper()
		for key, want := range map[string]any{"model": "user-model", "profiles.invented.model": "user-profile-model", "mcp_servers.user-other.command": "user-other-cmd"} {
			if got, present := read(t, config, key); !present || !adapter.SettingValuesEqual(got, want) {
				t.Fatalf("unrelated %s = %v (present %t):\n%s", key, got, present, mustRead(t, config))
			}
		}
	}

	t.Run("user env survives reinstall update remove", func(t *testing.T) {
		root, config := setup(t)
		if _, _, err := runInstall(t, codexStdioServer, "--target", "codex", "--local", "--deploy"); err != nil {
			t.Fatal(err)
		}
		if got, _ := read(t, config, prefix+"command"); got != "invented-mcp-server" {
			t.Fatalf("local stdio command = %v:\n%s", got, mustRead(t, config))
		}
		doc := map[string]any{}
		if err := toml.Unmarshal(mustRead(t, config), &doc); err != nil {
			t.Fatal(err)
		}
		server := doc["mcp_servers"].(map[string]any)[codexStdioServer].(map[string]any)
		server["env"] = map[string]any{"INVENTED_KEY": "invented-literal"}
		server["env_vars"] = []any{"INVENTED_FORWARD_ENV"}
		server["startup_timeout_sec"] = int64(17)
		raw, err := toml.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		codexWrite(t, config, raw)
		assertUser := func(t *testing.T) {
			t.Helper()
			assertUnrelated(t, config)
			for key, want := range map[string]any{prefix + "env.INVENTED_KEY": "invented-literal", prefix + "env_vars": []any{"INVENTED_FORWARD_ENV"}, prefix + "startup_timeout_sec": int64(17)} {
				if got, present := read(t, config, key); !present || !adapter.SettingValuesEqual(got, want) {
					t.Fatalf("user %s = %v (present %t):\n%s", key, got, present, mustRead(t, config))
				}
			}
		}
		if _, _, err := runInstall(t, codexStdioServer, "--target", "codex", "--local", "--deploy"); err != nil {
			t.Fatalf("reinstall refused: %v", err)
		}
		assertUser(t)
		codexStdioRecipe(t, root, "2.0.0", "serve-v2")
		if _, _, err := runUpdate(t, codexStdioServer, "--deploy"); err != nil {
			t.Fatalf("update refused: %v", err)
		}
		assertUser(t)
		if got, _ := read(t, config, prefix+"args"); !adapter.SettingValuesEqual(got, []any{"serve-v2"}) {
			t.Fatalf("update did not move managed args: %v", got)
		}
		state := string(mustRead(t, filepath.Join(root, ".patronus/state.json")))
		if strings.Contains(state, "INVENTED_FORWARD_ENV") || strings.Contains(state, "invented-literal") {
			t.Fatal("user env adopted into ownership state")
		}
		if _, _, err := execRemove(t, codexStdioServer, "--target", "codex", "--deploy"); err != nil {
			t.Fatal(err)
		}
		assertUser(t)
		for _, leaf := range []string{"command", "args"} {
			if got, present := read(t, config, prefix+leaf); present {
				t.Fatalf("owned %s survived removal: %v", leaf, got)
			}
		}
	})

	t.Run("no user fields removes only the managed server", func(t *testing.T) {
		_, config := setup(t)
		if _, _, err := runInstall(t, codexStdioServer, "--target", "codex", "--local", "--deploy"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := execRemove(t, codexStdioServer, "--target", "codex", "--deploy"); err != nil {
			t.Fatal(err)
		}
		if _, present := read(t, config, "mcp_servers."+codexStdioServer); present {
			t.Fatalf("empty managed server table left:\n%s", mustRead(t, config))
		}
		assertUnrelated(t, config)
	})
}
