package packagedelivery

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagestate"
)

type memoryFetcher []byte

func (f memoryFetcher) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f)), nil
}
func fixture(t *testing.T, home, version string) (Request, memoryFetcher) {
	t.Helper()
	identity := packagebundle.Identity{Name: "kit", Version: version, OS: "linux", Arch: "amd64"}
	data, err := packagebundle.Build(identity, "sha256:"+fmt.Sprintf("%064d", 0), []packagebundle.File{{Path: "dir/tool", Mode: 0755, Data: []byte(version)}})
	if err != nil {
		t.Fatal(err)
	}
	return Request{Recipe: "kit", RecipeVersion: version, URL: "https://example.test/kit.tar.gz", SHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(data)), Root: filepath.Join(home, ".patronus", "packages", "kit"), Identity: identity}, data
}
func locked(t *testing.T, home string) {
	t.Helper()
	release, err := packagestate.Acquire(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := release(); err != nil {
			t.Error(err)
		}
	})
}
func TestReplaceAndVerify(t *testing.T) {
	home := t.TempDir()
	locked(t, home)
	req, data := fixture(t, home, "1.0.0")
	s := Service{Home: home, Fetcher: data}
	result, err := s.Replace(context.Background(), req, false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Mutated {
		t.Fatal("first install skipped")
	}
	got, err := os.ReadFile(filepath.Join(req.Root, "dir/tool"))
	if err != nil || string(got) != "1.0.0" {
		t.Fatalf("content %q: %v", got, err)
	}
	result, err = s.Replace(context.Background(), req, false)
	if err != nil || result.Mutated {
		t.Fatalf("unchanged: %+v %v", result, err)
	}
	in, err := s.Inspect(req)
	if err != nil || len(in.Changed)+len(in.Missing)+len(in.Unknown) != 0 {
		t.Fatalf("inspection: %+v %v", in, err)
	}
}

func TestPartialReceiptRefetchesSamePin(t *testing.T) {
	s, req := installed(t)
	r, err := packagestate.Load(s.Home, req.Recipe)
	must(t, err)
	must(t, os.Remove(filepath.Join(req.Root, "dir/tool")))
	r.Files = r.Files[1:]
	r.Directories = []string{"."}
	must(t, os.Remove(filepath.Join(req.Root, "dir")))
	must(t, packagestate.Save(s.Home, r))
	result, err := s.Replace(context.Background(), req, false)
	must(t, err)
	if !result.Mutated {
		t.Fatal("partial receipt mistaken for complete inventory")
	}
	data, err := os.ReadFile(filepath.Join(req.Root, "dir/tool"))
	must(t, err)
	if string(data) != "1.0.0" {
		t.Fatal("missing payload not repaired")
	}
}

func TestMetadataOnlyNoDownload(t *testing.T) {
	s, req := installed(t)
	before, err := os.Stat(req.Root)
	must(t, err)
	s.Fetcher = failingFetcher{}
	req.RecipeVersion = "2.0.0"
	result, err := s.Replace(context.Background(), req, false)
	must(t, err)
	if !result.Mutated || result.Receipt.RecipeVersion != "2.0.0" {
		t.Fatal(result)
	}
	after, err := os.Stat(req.Root)
	must(t, err)
	if !os.SameFile(before, after) {
		t.Fatal("metadata update replaced root")
	}
	req.SHA256 = strings.ToUpper(strings.TrimPrefix(req.SHA256, "sha256:"))
	result, err = s.Replace(context.Background(), req, false)
	must(t, err)
	if result.Mutated {
		t.Fatal("equivalent digest caused mutation")
	}
}

type failingFetcher struct{}

func (failingFetcher) Open(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("unexpected download")
}
func TestMissingMemberRepairsAndForceRefetches(t *testing.T) {
	for _, kind := range []string{"missing", "edited"} {
		t.Run(kind, func(t *testing.T) {
			s, req := installed(t)
			path := filepath.Join(req.Root, "dir/tool")
			if kind == "missing" {
				must(t, os.Remove(path))
			} else {
				must(t, os.WriteFile(path, []byte("edit"), 0755))
			}
			s.Fetcher = failingFetcher{}
			_, err := s.Replace(context.Background(), req, kind == "edited")
			if err == nil || !strings.Contains(err.Error(), "unexpected download") {
				t.Fatal("repair did not reacquire archive")
			}
			_, data := fixture(t, s.Home, "1.0.0")
			s.Fetcher = data
			result, err := s.Replace(context.Background(), req, kind == "edited")
			must(t, err)
			if !result.Mutated {
				t.Fatal("repair skipped")
			}
			assertVersion(t, s.Home, "1.0.0")
		})
	}
}
func TestReplacementRemovesObsoleteFile(t *testing.T) {
	s, req := installed(t)
	req.Identity.Version = "2.0.0"
	req.RecipeVersion = "2.0.0"
	data, err := packagebundle.Build(req.Identity, "sha256:"+strings.Repeat("0", 64), []packagebundle.File{{Path: "replacement.txt", Mode: 0644, Data: []byte("new")}})
	must(t, err)
	req.SHA256 = bytesDigest(data)
	s.Fetcher = memoryFetcher(data)
	_, err = s.Replace(context.Background(), req, false)
	must(t, err)
	if _, err := os.Lstat(filepath.Join(req.Root, "dir/tool")); !os.IsNotExist(err) {
		t.Fatal("obsolete file survived")
	}
	info, err := os.Stat(filepath.Join(req.Root, "replacement.txt"))
	must(t, err)
	if info.Mode().Perm() != 0644 {
		t.Fatal(info.Mode())
	}
}
