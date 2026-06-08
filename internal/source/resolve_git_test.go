package source

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/archive"
)

type fakeFetcher struct {
	bodies map[string][]byte
}

func (f fakeFetcher) Fetch(_ context.Context, url string) (io.ReadCloser, error) {
	b, ok := f.bodies[url]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// hostTarball builds a GitHub-style source archive: every member nested under a
// top-level <repo>-<ref>/ prefix.
func hostTarball(t *testing.T, repo, ref string, members map[string]string) []byte {
	t.Helper()
	files := map[string][]byte{}
	prefix := repo + "-" + ref + "/"
	for name, content := range members {
		files[prefix+name] = []byte(content)
	}
	tgz, err := archive.CreateTarGz(files)
	if err != nil {
		t.Fatal(err)
	}
	return tgz
}

func TestResolveGitArtifactItem(t *testing.T) {
	tgz := hostTarball(t, "agent-kit", "v2", map[string]string{
		"pattern-internal/patronus.yaml": "apiVersion: patronus/v1\nkind: Skill\nrole: pattern\nname: pattern-internal\ndescription: d\nversion: 1.0.0\nentry: SKILL.md\ntargets: [claude]\ndefaults:\n  scope: project\n",
		"pattern-internal/SKILL.md":      "# internal",
		"README.md":                      "ignore me",
	})
	url := "https://github.com/me/agent-kit/archive/v2.tar.gz"
	rs := &Resolver{Fetcher: fakeFetcher{bodies: map[string][]byte{url: tgz}}, CacheDir: t.TempDir()}

	ref, _ := Parse("git:github.com/me/agent-kit@v2#pattern-internal")
	got, err := rs.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Artifact == nil || got.Artifact.Manifest.Name != "pattern-internal" {
		t.Fatalf("got %+v", got)
	}
	if got.ResolvedRef != ref.Raw {
		t.Errorf("resolvedRef = %q", got.ResolvedRef)
	}
	// Materialized: patronus.yaml + SKILL.md on disk under LocalDir.
	if _, err := os.Stat(filepath.Join(got.Artifact.Source.LocalDir, "SKILL.md")); err != nil {
		t.Errorf("SKILL.md not materialized: %v", err)
	}
}

func TestResolveGitRecipeItem(t *testing.T) {
	tgz := hostTarball(t, "kit", "main", map[string]string{
		"foo-mcp.yaml": "apiVersion: patronus/v1\nkind: Recipe\nname: foo-mcp\ncapability: tools\nsummary: s\nwire:\n  mcp:\n    transport: http\n    url: https://x/\n",
	})
	url := "https://github.com/me/kit/archive/main.tar.gz"
	rs := &Resolver{Fetcher: fakeFetcher{bodies: map[string][]byte{url: tgz}}, CacheDir: t.TempDir()}

	ref, _ := Parse("git:github.com/me/kit@main#foo-mcp")
	got, err := rs.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Recipe == nil || got.Recipe.Manifest.Name != "foo-mcp" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveGitMissingItem(t *testing.T) {
	tgz := hostTarball(t, "kit", "v1", map[string]string{"README.md": "x"})
	url := "https://github.com/me/kit/archive/v1.tar.gz"
	rs := &Resolver{Fetcher: fakeFetcher{bodies: map[string][]byte{url: tgz}}, CacheDir: t.TempDir()}
	ref, _ := Parse("git:github.com/me/kit@v1#nope")
	if _, err := rs.Resolve(context.Background(), ref); err == nil {
		t.Fatal("expected missing-item error")
	}
}

func TestResolveHTTPSArtifact(t *testing.T) {
	manifestYAML := "apiVersion: patronus/v1\nkind: Skill\nrole: capability\nname: web-skill\ndescription: d\nversion: 1.0.0\nentry: SKILL.md\ntargets: [claude]\ndefaults:\n  scope: project\n"
	base := "https://example.com/skills"
	rs := &Resolver{Fetcher: fakeFetcher{bodies: map[string][]byte{
		base + "/web-skill.yaml": []byte(manifestYAML),
		base + "/SKILL.md":       []byte("# web body"),
	}}, CacheDir: t.TempDir()}

	ref, _ := Parse(base + "/web-skill.yaml")
	got, err := rs.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Artifact == nil || got.Artifact.Manifest.Name != "web-skill" {
		t.Fatalf("got %+v", got)
	}
	if _, err := os.Stat(filepath.Join(got.Artifact.Source.LocalDir, "SKILL.md")); err != nil {
		t.Errorf("entry not materialized: %v", err)
	}
}

func TestResolveHTTPSRecipe(t *testing.T) {
	recipeYAML := "apiVersion: patronus/v1\nkind: Recipe\nname: hosted-mcp\ncapability: tools\nsummary: s\nwire:\n  mcp:\n    transport: http\n    url: https://x/\n"
	url := "https://example.com/hosted-mcp.yaml"
	rs := &Resolver{Fetcher: fakeFetcher{bodies: map[string][]byte{url: []byte(recipeYAML)}}, CacheDir: t.TempDir()}
	ref, _ := Parse(url)
	got, err := rs.Resolve(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Recipe == nil || got.Recipe.Manifest.Name != "hosted-mcp" {
		t.Fatalf("got %+v", got)
	}
}
