package adapter

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/toolpath"
)

func TestCodexSkillRoot(t *testing.T) {
	home, project, config := t.TempDir(), t.TempDir(), t.TempDir()
	env := func(k string) (string, bool) {
		if k == "CODEX_HOME" {
			return config, true
		}
		return "", false
	}
	res := toolpath.New(env, home, project)
	ad := loadAdapter(t, "codex")
	for _, tc := range []struct{ scope, want string }{{"global", filepath.Join(home, ".agents/skills/demo-cx/SKILL.md")}, {"local", filepath.Join(project, ".agents/skills/demo-cx/SKILL.md")}} {
		marker := ad.Layout.Skill.ForScope(tc.scope).Path
		got := res.ResolveMarker(strings.ReplaceAll(marker, "{name}", "demo-cx"), "codex", tc.scope)
		if got != tc.want {
			t.Errorf("%s skill root = %s, want %s", tc.scope, got, tc.want)
		}
	}
	if got := res.ResolveMarker(ad.Layout.Setting.ForScope("global").File, "codex", "global"); got != filepath.Join(config, "config.toml") {
		t.Fatal(got)
	}
}

func TestCodexSettingRemovalPreservesSiblingFields(t *testing.T) {
	home := t.TempDir()
	eng := New(toolpath.New(testEnv(home), home, t.TempDir()))
	prior := []byte("model = 'user-model'\n[mcp_servers.fixture]\nbearer_token_env_var = 'INVENTED_TOKEN_ENV'\n[mcp_servers.fixture.http_headers]\nX-Invented = 'fixture-reference'\n")
	art := settingArtifact("fixture-url-cx", "mcp_servers.fixture.url", "https://fixture.invalid/mcp")
	ds, err := eng.Transform(art, loadAdapter(t, "codex"), "global", "", existingBytes(prior))
	if err != nil {
		t.Fatal(err)
	}
	if len(ds) != 1 || ds[0].Setting == nil || ds[0].Setting.Dotted != "mcp_servers.fixture.url" {
		t.Fatalf("no structured MCP key ownership: %+v", ds)
	}
	again, err := eng.Transform(art, loadAdapter(t, "codex"), "global", "", existingBytes(ds[0].After))
	if err != nil {
		t.Fatal(err)
	}
	if string(again[0].After) != string(ds[0].After) {
		t.Fatal("reinstall changed sibling/auth")
	}
	removed, found, err := RemoveSettingEdit(ds[0].After, ds[0].Setting)
	if err != nil || !found {
		t.Fatalf("remove: %v %t", err, found)
	}
	target := loadAdapter(t, "codex").Layout.Setting.ForScope("global")
	for _, key := range []string{"model", "mcp_servers.fixture.bearer_token_env_var", "mcp_servers.fixture.http_headers.X-Invented"} {
		before, _, err := ReadDotted(prior, target, key)
		if err != nil {
			t.Fatal(err)
		}
		after, present, err := ReadDotted(removed, target, key)
		if err != nil || !present || !SettingValuesEqual(before, after) {
			t.Fatalf("lost user field %s: %v %v", key, after, err)
		}
	}
}
