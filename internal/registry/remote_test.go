package registry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/archive"
	"github.com/darkquasar/patronus/internal/manifest"
)

// fakeFetcher serves canned bytes per URL and counts hits, so a test can assert a
// warm cache performs ZERO network.
type fakeFetcher struct {
	bodies map[string][]byte
	hits   int
}

func (f *fakeFetcher) Fetch(_ context.Context, url string) (io.ReadCloser, error) {
	f.hits++
	b, ok := f.bodies[url]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}

// buildServed constructs a discovery index + one artifact tarball and returns a
// fakeFetcher serving them at the R2 base URLs (DefaultRegistryURL/catalog/...).
func buildServed(t *testing.T) (*fakeFetcher, string) {
	t.Helper()
	src := map[string][]byte{
		"patronus.yaml": []byte("kind: Skill\nname: demo\nversion: 1.0.0\n"),
		"SKILL.md":      []byte("# demo body"),
	}
	tgz, err := archive.CreateTarGz(src)
	if err != nil {
		t.Fatal(err)
	}
	tarURL := DefaultRegistryURL + "/catalog/demo/1.0.0/demo-1.0.0.tar.gz"

	ix := &Index{
		SchemaVersion: IndexSchemaVersion,
		Artifacts: []IndexArtifact{{
			Manifest: &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: "demo", Version: "1.0.0"}, Type: manifest.TypeSkill, Entry: "SKILL.md"},
			Tarball:  Tarball{URL: tarURL, SHA256: sha(tgz)},
		}},
	}
	data, _ := ix.Marshal()
	idxURL := DefaultRegistryURL + "/catalog/index.json"
	f := &fakeFetcher{bodies: map[string][]byte{
		idxURL:             data,
		idxURL + ".sha256": []byte(sha(data) + "\n"),
		tarURL:             tgz,
	}}
	return f, "SKILL.md"
}

func TestRemoteColdBootstrapThenWarm(t *testing.T) {
	f, _ := buildServed(t)
	cache := t.TempDir()
	r := NewRemoteRegistry(f, cache, "")

	cat, err := r.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Artifacts) != 1 || cat.Artifacts[0].Manifest.Name != "demo" {
		t.Fatalf("catalog: %+v", cat)
	}
	if cat.Artifacts[0].Source.TarballURL == "" {
		t.Fatal("remote source not set")
	}
	coldHits := f.hits
	if coldHits == 0 {
		t.Fatal("cold cache should have fetched")
	}

	// Warm: a second Catalog reads the cache with ZERO additional network.
	if _, err := r.Catalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.hits != coldHits {
		t.Fatalf("warm cache hit network: %d → %d", coldHits, f.hits)
	}
}

func TestRemoteShaMismatchFatal(t *testing.T) {
	f, _ := buildServed(t)
	idxURL := DefaultRegistryURL + "/catalog/index.json"
	f.bodies[idxURL+".sha256"] = []byte("sha256:deadbeef\n") // wrong
	r := NewRemoteRegistry(f, t.TempDir(), "")
	if _, err := r.Catalog(context.Background()); err == nil {
		t.Fatal("expected sha256 mismatch error")
	}
}

func TestRemoteShaSidecarAbsentFallsBack(t *testing.T) {
	f, _ := buildServed(t)
	idxURL := DefaultRegistryURL + "/catalog/index.json"
	delete(f.bodies, idxURL+".sha256") // absent → TLS-trust fallback
	r := NewRemoteRegistry(f, t.TempDir(), "")
	if _, err := r.Catalog(context.Background()); err != nil {
		t.Fatalf("absent sidecar should fall back, got %v", err)
	}
}

func TestRemoteIndexURLAndBase(t *testing.T) {
	// Default base.
	r := NewRemoteRegistry(&fakeFetcher{}, t.TempDir(), "")
	if r.Base() != DefaultRegistryURL {
		t.Fatalf("default base = %q", r.Base())
	}
	if r.indexURL() != DefaultRegistryURL+"/catalog/index.json" {
		t.Fatalf("index URL = %q", r.indexURL())
	}
	// Custom base (fork/mirror), trailing slash trimmed.
	r2 := NewRemoteRegistry(&fakeFetcher{}, t.TempDir(), "https://mirror.example.com/")
	if r2.Base() != "https://mirror.example.com" {
		t.Fatalf("custom base = %q", r2.Base())
	}
	// Different bases cache to different files (no clobber).
	if r.cacheKey() == r2.cacheKey() {
		t.Fatal("distinct bases must have distinct cache keys")
	}
}

func TestRemoteOfflineColdErrors(t *testing.T) {
	f := &fakeFetcher{bodies: map[string][]byte{}} // serves nothing
	r := NewRemoteRegistry(f, t.TempDir(), "")
	if _, err := r.Catalog(context.Background()); err == nil {
		t.Fatal("offline + cold cache should error")
	}
}

func TestRemoteRefreshKeepsCacheOnFailure(t *testing.T) {
	f, _ := buildServed(t)
	cache := t.TempDir()
	r := NewRemoteRegistry(f, cache, "")
	// Warm the cache.
	if _, err := r.Catalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Now make the fetcher fail and Refresh: cache must be kept, no error.
	var warned bool
	r.Warnf = func(string, ...any) { warned = true }
	f.bodies = map[string][]byte{} // serve nothing
	cat, err := r.Refresh(context.Background())
	if err != nil {
		t.Fatalf("refresh should keep cache, got %v", err)
	}
	if cat == nil || len(cat.Artifacts) != 1 {
		t.Fatalf("expected cached catalog, got %+v", cat)
	}
	if !warned {
		t.Error("expected a warning on kept-cache refresh")
	}
}

func TestMaterializeRoundTripAndIdempotent(t *testing.T) {
	f, body := buildServed(t)
	cache := t.TempDir()
	r := NewRemoteRegistry(f, cache, "")
	cat, err := r.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	e := &cat.Artifacts[0]

	dir, err := r.Materialize(context.Background(), e)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, body))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "# demo body" {
		t.Fatalf("materialized body = %q", got)
	}
	if e.Source.LocalDir != dir {
		t.Fatal("LocalDir not set after materialize")
	}

	// Idempotent: second call hits cache, no extra tarball fetch.
	hits := f.hits
	e2 := &ArtifactEntry{Manifest: e.Manifest, Source: Source{TarballURL: e.Source.TarballURL, SHA256: e.Source.SHA256}}
	if _, err := r.Materialize(context.Background(), e2); err != nil {
		t.Fatal(err)
	}
	if f.hits != hits {
		t.Fatalf("idempotent materialize hit network: %d → %d", hits, f.hits)
	}
}

func TestMaterializeShaMismatchWritesNothing(t *testing.T) {
	f, _ := buildServed(t)
	cache := t.TempDir()
	r := NewRemoteRegistry(f, cache, "")
	cat, _ := r.Catalog(context.Background())
	e := &cat.Artifacts[0]
	e.Source.SHA256 = "sha256:deadbeef" // corrupt the pin

	if _, err := r.Materialize(context.Background(), e); err == nil {
		t.Fatal("expected sha mismatch error")
	}
	if _, err := os.Stat(filepath.Join(cache, "items", "demo-1.0.0", "patronus.yaml")); err == nil {
		t.Fatal("materialize wrote files despite sha mismatch")
	}
}
