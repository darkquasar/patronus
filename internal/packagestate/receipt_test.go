package packagestate

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/packagebundle"
)

func receiptFixture(t *testing.T, home string) *Receipt {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatal(err)
	}
	return &Receipt{SchemaVersion: 1, Recipe: "pi-sandbox", RecipeVersion: "1.0.0", Root: filepath.Join(canonical, ".patronus/packages/pi-sandbox"), URL: "https://example.com/package.tar.gz", ArchiveSHA256: "sha256:" + strings.Repeat("a", 64), Identity: packagebundle.Identity{Name: "pi-sandbox", Version: "1.0.0", OS: "linux", Arch: "amd64"}, Files: []packagebundle.Entry{{Path: "package.json", Mode: 0644, SHA256: "sha256:" + strings.Repeat("b", 64)}}, Directories: []string{"."}}
}
func TestReceiptAbsent(t *testing.T) {
	home := t.TempDir()
	got, err := Load(home, "pi-sandbox")
	if err != nil || got != nil {
		t.Fatalf("got %#v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".patronus")); !os.IsNotExist(err) {
		t.Fatalf("read mutated home: %v", err)
	}
}
func TestReceiptRoundTrip(t *testing.T) {
	home := t.TempDir()
	want := receiptFixture(t, home)
	if err := Save(home, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(home, want.Recipe)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v %v", got, err)
	}
	list, err := List(home)
	if err != nil || len(list) != 1 {
		t.Fatalf("list %#v %v", list, err)
	}
	for _, p := range []string{".patronus/package-state", ".patronus/package-state/pi-sandbox.json"} {
		info, err := os.Stat(filepath.Join(home, p))
		if err != nil {
			t.Fatal(err)
		}
		wantMode := os.FileMode(0700)
		if !info.IsDir() {
			wantMode = 0600
		}
		if info.Mode().Perm() != wantMode {
			t.Fatalf("%s mode %v", p, info.Mode())
		}
	}
	if err := DeleteReceipt(home, want.Recipe); err != nil {
		t.Fatal(err)
	}
	if err := DeleteReceipt(home, want.Recipe); err != nil {
		t.Fatal(err)
	}
}
func TestReceiptRejectsForgedRecords(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Receipt)
	}{
		{"schema", func(r *Receipt) { r.SchemaVersion = 2 }}, {"recipe", func(r *Receipt) { r.Recipe = "../escape" }}, {"root", func(r *Receipt) { r.Root = "/tmp/escape" }}, {"digest", func(r *Receipt) { r.ArchiveSHA256 = "abc" }}, {"file", func(r *Receipt) { r.Files[0].Path = "../escape" }}, {"directory", func(r *Receipt) { r.Directories = []string{"../escape"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			r := receiptFixture(t, home)
			tc.mutate(r)
			if err := Save(home, r); err == nil {
				t.Fatal("accepted forged receipt")
			}
		})
	}
}
func TestReceiptStrictJSON(t *testing.T) {
	for _, data := range []string{`{`, `{"schemaVersion":2}`, `{"schemaVersion":1,"unknown":1}`, `{} {}`, `null`} {
		t.Run(data, func(t *testing.T) {
			home := t.TempDir()
			r := receiptFixture(t, home)
			if err := Save(home, r); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(home, ".patronus/package-state/pi-sandbox.json"), []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(home, r.Recipe); err == nil {
				t.Fatal("accepted invalid JSON")
			}
		})
	}
}
func TestReceiptSymlinkAncestors(t *testing.T) {
	for _, rel := range []string{".patronus", ".patronus/package-state", ".patronus/packages", ".patronus/packages/pi-sandbox", ".patronus/package-state/pi-sandbox.json"} {
		t.Run(rel, func(t *testing.T) {
			home := t.TempDir()
			target := t.TempDir()
			path := filepath.Join(home, rel)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
			if err := Save(home, receiptFixture(t, home)); err == nil || !strings.Contains(err.Error(), "unsupported symlink") || !strings.Contains(err.Error(), path) {
				t.Fatalf("got %v", err)
			}
		})
	}
}
func TestReceiptDurabilityFaults(t *testing.T) {
	for _, point := range []string{"before-temp-sync", "after-temp-sync", "before-rename", "after-rename", "before-parent-sync", "after-parent-sync"} {
		t.Run(point, func(t *testing.T) {
			home := t.TempDir()
			r := receiptFixture(t, home)
			if err := Save(home, r); err != nil {
				t.Fatal(err)
			}
			r.RecipeVersion = "2.0.0"
			injected := errors.New("injected")
			s := storage{fault: func(p, _ string) error {
				if p == point {
					return injected
				}
				return nil
			}}
			err := s.save(home, r)
			var de *DurabilityError
			if !errors.As(err, &de) || !errors.Is(err, injected) {
				t.Fatalf("got %v", err)
			}
			visible := point == "after-rename" || point == "before-parent-sync" || point == "after-parent-sync"
			if de.MayBeVisible != visible {
				t.Fatalf("visibility %v", de)
			}
			got, err := Load(home, r.Recipe)
			if err != nil {
				t.Fatal(err)
			}
			want := "1.0.0"
			if visible {
				want = "2.0.0"
			}
			if got.RecipeVersion != want {
				t.Fatalf("version %s", got.RecipeVersion)
			}
		})
	}
}

func TestReceiptHomeSymlinkResolved(t *testing.T) {
	home := t.TempDir()
	link := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(home, link); err != nil {
		t.Fatal(err)
	}
	r := receiptFixture(t, home)
	if err := Save(link, r); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(link, r.Recipe); err != nil {
		t.Fatal(err)
	}
}
func TestReceiptLoadRejectsMismatchedName(t *testing.T) {
	home := t.TempDir()
	r := receiptFixture(t, home)
	if err := Save(home, r); err != nil {
		t.Fatal(err)
	}
	from := filepath.Join(home, ".patronus/package-state/pi-sandbox.json")
	to := filepath.Join(home, ".patronus/package-state/other.json")
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(home, "other"); err == nil {
		t.Fatal("accepted mismatched recipe")
	}
	if _, err := List(home); err == nil {
		t.Fatal("listing accepted mismatched recipe")
	}
}
func TestReceiptPartialRemovalInventory(t *testing.T) {
	home := t.TempDir()
	r := receiptFixture(t, home)
	r.Files = nil
	r.Directories = nil
	if err := Save(home, r); err != nil {
		t.Fatal(err)
	}
}
func TestReceiptDeleteDurability(t *testing.T) {
	for _, point := range []string{"before-unlink", "after-unlink", "before-parent-sync", "after-parent-sync"} {
		t.Run(point, func(t *testing.T) {
			home := t.TempDir()
			r := receiptFixture(t, home)
			if err := Save(home, r); err != nil {
				t.Fatal(err)
			}
			injected := errors.New("disk fault")
			s := storage{fault: func(p, _ string) error {
				if p == point {
					return injected
				}
				return nil
			}}
			err := s.deleteReceipt(home, r.Recipe)
			var de *DurabilityError
			if !errors.As(err, &de) || !errors.Is(err, injected) || de.MayBeVisible != (point != "before-unlink") {
				t.Fatalf("got %#v %v", de, err)
			}
			got, err := Load(home, r.Recipe)
			if err != nil {
				t.Fatal(err)
			}
			if (got == nil) != (point != "before-unlink") {
				t.Fatalf("receipt %#v", got)
			}
			if point != "before-unlink" {
				if err := s.deleteReceipt(home, r.Recipe); err == nil {
					t.Fatal("absent deletion omitted parent sync")
				}
			}
		})
	}
}
func TestReceiptRealRenameFailurePreservesOld(t *testing.T) {
	home := t.TempDir()
	r := receiptFixture(t, home)
	if err := Save(home, r); err != nil {
		t.Fatal(err)
	}
	r.RecipeVersion = "2.0.0"
	s := storage{fault: func(point, path string) error {
		if point == "before-rename" {
			files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".state-*"))
			if err != nil {
				return err
			}
			for _, f := range files {
				if err := os.Remove(f); err != nil {
					return err
				}
			}
		}
		return nil
	}}
	err := s.save(home, r)
	var de *DurabilityError
	if !errors.As(err, &de) || de.MayBeVisible {
		t.Fatalf("got %v", err)
	}
	got, err := Load(home, r.Recipe)
	if err != nil || got.RecipeVersion != "1.0.0" {
		t.Fatalf("got %#v %v", got, err)
	}
}

func TestReceiptRejectsOversizedJSON(t *testing.T) {
	home := t.TempDir()
	r := receiptFixture(t, home)
	r.URL = strings.Repeat("x", 8<<20)
	if err := Save(home, r); err == nil {
		t.Fatal("saved oversized record")
	}
	r = receiptFixture(t, home)
	if err := Save(home, r); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(home, ".patronus/package-state/pi-sandbox.json")
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(strings.Repeat(" ", 8<<20)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := Load(home, r.Recipe); err == nil {
		t.Fatal("accepted oversized trailing bytes")
	}
}
func TestReceiptMakesExistingStateDirectoryPrivate(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".patronus/package-state")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := Save(home, receiptFixture(t, home)); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0700 {
		t.Fatalf("state directory remains %v", info.Mode())
	}
}
