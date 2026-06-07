package adapter

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	toml "github.com/pelletier/go-toml/v2"
)

func loadAdapter(t *testing.T, tool string) *manifest.Adapter {
	t.Helper()
	ad, err := manifest.LoadAdapter(filepath.Join("..", "..", "adapters", tool+".yaml"))
	if err != nil {
		t.Fatalf("load %s adapter: %v", tool, err)
	}
	return ad
}

func TestMergeClaudeStdioHasTypeField(t *testing.T) {
	ad := loadAdapter(t, "claude")
	ft := ad.Layout.Mcp.ForScope("local")
	tr := ad.Layout.Mcp.Transports["stdio"]
	spec := ServerSpec{
		Name:      "memory",
		Transport: "stdio",
		Values: map[string]any{
			"command": "npx",
			"args":    []string{"-y", "server-memory"},
			"env":     map[string]any{"KEY": "v"},
		},
	}
	out, err := MergeConfig(nil, ft, tr, spec)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	srv := doc["mcpServers"].(map[string]any)["memory"].(map[string]any)
	if srv["type"] != "stdio" {
		t.Errorf("claude stdio must carry type:stdio, got %v", srv["type"])
	}
	if srv["command"] != "npx" {
		t.Errorf("command = %v", srv["command"])
	}
	if _, ok := srv["args"].([]any); !ok {
		t.Errorf("args should be an array, got %T", srv["args"])
	}
}

func TestMergeClaudeHTTP(t *testing.T) {
	ad := loadAdapter(t, "claude")
	ft := ad.Layout.Mcp.ForScope("local")
	tr := ad.Layout.Mcp.Transports["http"]
	spec := ServerSpec{Name: "remote", Transport: "http", Values: map[string]any{
		"url": "https://x", "headers": map[string]any{"A": "b"},
	}}
	out, err := MergeConfig(nil, ft, tr, spec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"type": "http"`) || !strings.Contains(string(out), "https://x") {
		t.Errorf("claude http unexpected:\n%s", out)
	}
}

func TestMergeCodexShapeByKeyNoType(t *testing.T) {
	ad := loadAdapter(t, "codex")
	ft := ad.Layout.Mcp.ForScope("global")
	tr := ad.Layout.Mcp.Transports["stdio"]
	spec := ServerSpec{Name: "memory", Transport: "stdio", Values: map[string]any{
		"command": "uvx", "args": []string{"mem"}, "env": map[string]any{"K": "v"},
	}}
	out, err := MergeConfig(nil, ft, tr, spec)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := toml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("invalid toml: %v\n%s", err, out)
	}
	srv := doc["mcp_servers"].(map[string]any)["memory"].(map[string]any)
	// §9.9: NO type key for codex.
	if _, ok := srv["type"]; ok {
		t.Errorf("codex stdio must NOT carry a type key:\n%s", out)
	}
	if srv["command"] != "uvx" {
		t.Errorf("command = %v", srv["command"])
	}
}

func TestMergeCodexHTTPKeys(t *testing.T) {
	ad := loadAdapter(t, "codex")
	ft := ad.Layout.Mcp.ForScope("global")
	tr := ad.Layout.Mcp.Transports["http"]
	spec := ServerSpec{Name: "gh", Transport: "http", Values: map[string]any{
		"url": "https://api", "bearerTokenEnvVar": "GH_TOKEN", "headers": map[string]any{"X": "1"},
	}}
	out, err := MergeConfig(nil, ft, tr, spec)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "bearer_token_env_var") || !strings.Contains(s, "GH_TOKEN") {
		t.Errorf("codex http missing bearer mapping:\n%s", s)
	}
	if strings.Contains(s, "type =") {
		t.Errorf("codex http must not carry type:\n%s", s)
	}
}

func TestMergeOpencodeLocalRemoteAndArray(t *testing.T) {
	ad := loadAdapter(t, "opencode")
	ft := ad.Layout.Mcp.ForScope("local")
	tr := ad.Layout.Mcp.Transports["stdio"]
	spec := ServerSpec{Name: "memory", Transport: "stdio", Values: map[string]any{
		"commandArray": []string{"npx", "-y", "mem"},
		"env":          map[string]any{"K": "v"},
	}}
	out, err := MergeConfig(nil, ft, tr, spec)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	srv := doc["mcp"].(map[string]any)["memory"].(map[string]any)
	if srv["type"] != "local" {
		t.Errorf("opencode stdio type must be local, got %v", srv["type"])
	}
	if _, ok := srv["command"].([]any); !ok {
		t.Errorf("opencode command must be an array, got %T (%v)", srv["command"], srv["command"])
	}
	// http -> remote
	trh := ad.Layout.Mcp.Transports["http"]
	out2, err := MergeConfig(nil, ft, trh, ServerSpec{Name: "r", Transport: "http", Values: map[string]any{"url": "https://y"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out2), `"type": "remote"`) {
		t.Errorf("opencode http must be type:remote:\n%s", out2)
	}
}

func TestMergePreservesExistingKeys(t *testing.T) {
	ad := loadAdapter(t, "claude")
	ft := ad.Layout.Mcp.ForScope("local")
	tr := ad.Layout.Mcp.Transports["stdio"]
	existing := []byte(`{"otherTop":"keep","mcpServers":{"existing":{"type":"stdio","command":"old"}}}`)
	out, err := MergeConfig(existing, ft, tr, ServerSpec{Name: "new", Transport: "stdio", Values: map[string]any{"command": "c"}})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["otherTop"] != "keep" {
		t.Error("top-level sibling key lost")
	}
	servers := doc["mcpServers"].(map[string]any)
	if _, ok := servers["existing"]; !ok {
		t.Error("existing server entry lost")
	}
	if _, ok := servers["new"]; !ok {
		t.Error("new server not added")
	}
}

func TestMergeJSONCStripsComments(t *testing.T) {
	ad := loadAdapter(t, "opencode")
	ft := ad.Layout.Mcp.ForScope("local")
	tr := ad.Layout.Mcp.Transports["http"]
	existing := []byte("{\n  // a comment\n  \"keep\": true /* inline */\n}")
	out, err := MergeConfig(existing, ft, tr, ServerSpec{Name: "r", Transport: "http", Values: map[string]any{"url": "https://z"}})
	if err != nil {
		t.Fatalf("jsonc merge failed: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("re-emitted json invalid: %v\n%s", err, out)
	}
	if doc["keep"] != true {
		t.Error("jsonc sibling key lost")
	}
}

func TestStripJSONCommentsPreservesStrings(t *testing.T) {
	in := []byte(`{"url":"http://x//y","note":"/* not a comment */"}`)
	got := stripJSONComments(in)
	var doc map[string]any
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("invalid after strip: %v\n%s", err, got)
	}
	if doc["url"] != "http://x//y" || doc["note"] != "/* not a comment */" {
		t.Errorf("string content corrupted: %v", doc)
	}
}

func TestBuildTransportObjectOmitsMissing(t *testing.T) {
	ad := loadAdapter(t, "codex")
	tr := ad.Layout.Mcp.Transports["stdio"]
	// Only command provided; args/env absent -> omitted, no type key ever.
	obj := buildTransportObject(tr, map[string]any{"command": "c"})
	if _, ok := obj["type"]; ok {
		t.Error("codex stdio must never emit type")
	}
	if _, ok := obj["args"]; ok {
		t.Error("absent args must be omitted")
	}
	if obj["command"] != "c" {
		t.Errorf("command = %v", obj["command"])
	}
}

func TestSetDottedDeep(t *testing.T) {
	root := map[string]any{}
	if err := setDotted(root, "a.b.c", 42); err != nil {
		t.Fatal(err)
	}
	if root["a"].(map[string]any)["b"].(map[string]any)["c"] != 42 {
		t.Errorf("deep set failed: %v", root)
	}
}
