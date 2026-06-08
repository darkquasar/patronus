package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
)

func runLock(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newLockCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

func TestLockRequiresProfile(t *testing.T) {
	_, _, err := runLock(t)
	if err == nil {
		t.Fatal("expected error when --profile is omitted")
	}
}

func TestLockRejectsPositionalArgs(t *testing.T) {
	_, _, err := runLock(t, "extra", "--profile", "cloudflare")
	if err == nil {
		t.Fatal("expected error for positional args")
	}
}

// TestLockBuildsFromRealCatalog drives the lock machinery against the real repo
// catalog (resolved from this package's dir) and writes to a temp path, mirroring
// the install deploy tests that exercise machinery directly rather than the cwd
// write the cobra command performs.
func TestLockBuildsFromRealCatalog(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := registry.DiscoverRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := registry.NewLocalRegistry(root).Catalog(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	res, err := profile.Resolve(cat, "cloudflare")
	if err != nil {
		t.Fatal(err)
	}
	l, err := lock.FromResolved(cat, res, "2026-06-07T00:00:00Z", "v0.6.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Entries) == 0 {
		t.Fatal("expected entries for cloudflare profile")
	}
	for _, e := range l.Entries {
		if e.Source != "registry" {
			t.Errorf("%s: source %q, want registry", e.Name, e.Source)
		}
		if e.SHA256 == "" {
			t.Errorf("%s: empty sha256", e.Name)
		}
	}

	path := filepath.Join(t.TempDir(), "patronus.lock")
	if err := lock.Save(path, l); err != nil {
		t.Fatal(err)
	}
	got, err := lock.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Profile != "cloudflare" || len(got.Entries) != len(l.Entries) {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}
