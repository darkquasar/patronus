package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBareName(t *testing.T) {
	r, err := Parse("pattern-cloudflare")
	if err != nil {
		t.Fatal(err)
	}
	if r.Scheme != Registry || r.Name != "pattern-cloudflare" {
		t.Fatalf("got %+v", r)
	}
	if r.LockSource() != "registry" {
		t.Fatalf("lock source = %q, want registry", r.LockSource())
	}
}

func TestParseFile(t *testing.T) {
	r, err := Parse("file:../local-skills/my-skill")
	if err != nil {
		t.Fatal(err)
	}
	if r.Scheme != File || r.Path != "../local-skills/my-skill" {
		t.Fatalf("got %+v", r)
	}
	if r.LockSource() != "file:../local-skills/my-skill" {
		t.Fatalf("lock source = %q", r.LockSource())
	}
}

func TestParseGit(t *testing.T) {
	r, err := Parse("git:github.com/me/agent-kit@v2#pattern-internal")
	if err != nil {
		t.Fatal(err)
	}
	if r.Scheme != Git || r.Host != "github.com" || r.Owner != "me" ||
		r.Repo != "agent-kit" || r.GitRef != "v2" || r.Item != "pattern-internal" {
		t.Fatalf("got %+v", r)
	}
	if r.LockSource() != r.Raw {
		t.Fatalf("git lock source should be raw ref")
	}
}

func TestParseGitMinimal(t *testing.T) {
	r, err := Parse("git:github.com/me/agent-kit")
	if err != nil {
		t.Fatal(err)
	}
	if r.GitRef != "" || r.Item != "" || r.Repo != "agent-kit" {
		t.Fatalf("got %+v", r)
	}
}

func TestParseHTTPS(t *testing.T) {
	r, err := Parse("https://example.com/recipes/foo-mcp.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if r.Scheme != HTTPS {
		t.Fatalf("got %+v", r)
	}
}

func TestParseMalformed(t *testing.T) {
	cases := []string{
		"",
		"file:",
		"git:github.com/me", // too few path parts
		"https://example.com/no-extension",
		"http://example.com/x.yaml", // insecure
		"gti:typo/owner/repo",       // mistyped scheme
	}
	for _, c := range cases {
		if _, err := Parse(c); err == nil {
			t.Errorf("Parse(%q) = nil error, want error", c)
		}
	}
}

func TestResolveRegistryNoop(t *testing.T) {
	r, _ := Parse("some-name")
	got, err := Resolve(r)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("registry resolve should be nil entry, got %+v", got)
	}
}

func TestResolveGitRejected(t *testing.T) {
	r, _ := Parse("git:github.com/me/repo@v1")
	if _, err := Resolve(r); err == nil || !strings.Contains(err.Error(), "Phase 6") {
		t.Fatalf("want Phase 6 rejection, got %v", err)
	}
}

func TestResolveHTTPSRejected(t *testing.T) {
	r, _ := Parse("https://example.com/x.yaml")
	if _, err := Resolve(r); err == nil || !strings.Contains(err.Error(), "Phase 6") {
		t.Fatalf("want Phase 6 rejection, got %v", err)
	}
}

func TestResolveFileArtifact(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifestYAML := `apiVersion: patronus/v1
kind: Skill
role: capability
name: my-skill
description: A local skill.
version: 1.0.0
entry: SKILL.md
targets: [claude]
defaults:
  scope: project
`
	writeFile(t, filepath.Join(skillDir, "patronus.yaml"), manifestYAML)
	writeFile(t, filepath.Join(skillDir, "SKILL.md"), "# body")

	r, _ := Parse("file:" + skillDir)
	got, err := Resolve(r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Artifact == nil || got.Artifact.Manifest.Name != "my-skill" {
		t.Fatalf("got %+v", got)
	}
	if got.Artifact.Source.LocalDir != skillDir {
		t.Fatalf("LocalDir = %q, want %q", got.Artifact.Source.LocalDir, skillDir)
	}
}

func TestResolveFileRecipe(t *testing.T) {
	dir := t.TempDir()
	recipePath := filepath.Join(dir, "foo-mcp.yaml")
	recipeYAML := `apiVersion: patronus/v1
kind: Recipe
name: foo-mcp
capability: tools
summary: Local recipe.
wire:
  mcp:
    transport: http
    url: "https://example.com/mcp/"
`
	writeFile(t, recipePath, recipeYAML)

	r, _ := Parse("file:" + recipePath)
	got, err := Resolve(r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Recipe == nil || got.Recipe.Manifest.Name != "foo-mcp" {
		t.Fatalf("got %+v", got)
	}
	if got.Recipe.Source.LocalDir != dir {
		t.Fatalf("LocalDir = %q, want %q", got.Recipe.Source.LocalDir, dir)
	}
}

func TestResolveFileMissing(t *testing.T) {
	r, _ := Parse("file:/no/such/path")
	if _, err := Resolve(r); err == nil {
		t.Fatal("want error for missing path")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
