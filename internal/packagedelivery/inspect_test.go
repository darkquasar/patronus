package packagedelivery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/darkquasar/patronus/internal/packagestate"
)

func installed(t *testing.T) (*Service, Request) {
	t.Helper()
	home := t.TempDir()
	locked(t, home)
	req, data := fixture(t, home, "1.0.0")
	s := &Service{Home: home, Fetcher: data}
	if _, err := s.Replace(context.Background(), req, false); err != nil {
		t.Fatal(err)
	}
	return s, req
}
func TestInspectDrift(t *testing.T) {
	for _, kind := range []string{"bytes", "mode", "missing", "unknown-file", "unknown-directory", "symlink", "directory-type"} {
		t.Run(kind, func(t *testing.T) {
			s, req := installed(t)
			path := filepath.Join(req.Root, "dir/tool")
			switch kind {
			case "bytes":
				must(t, os.WriteFile(path, []byte("edited"), 0755))
			case "mode":
				must(t, os.Chmod(path, 0600))
			case "missing":
				must(t, os.Remove(path))
			case "unknown-file":
				must(t, os.WriteFile(filepath.Join(req.Root, "extra"), []byte("user"), 0644))
			case "unknown-directory":
				must(t, os.Mkdir(filepath.Join(req.Root, "empty"), 0755))
			case "symlink":
				must(t, os.Remove(path))
				must(t, os.Symlink(t.TempDir(), path))
			case "directory-type":
				must(t, os.Remove(path))
				must(t, os.Mkdir(path, 0755))
			}
			in, err := s.Inspect(req)
			switch kind {
			case "bytes", "mode":
				if err != nil || !reflect.DeepEqual(in.Changed, []string{"dir/tool"}) {
					t.Fatalf("%+v %v", in, err)
				}
			case "missing":
				if err != nil || !reflect.DeepEqual(in.Missing, []string{"dir/tool"}) {
					t.Fatalf("%+v %v", in, err)
				}
			case "unknown-file", "unknown-directory":
				if err != nil || len(in.Unknown) != 1 {
					t.Fatalf("%+v %v", in, err)
				}
			default:
				var conflict *ConflictError
				if !errors.As(err, &conflict) || conflict.Kind != "path-type" {
					t.Fatal(err)
				}
			}
			if kind != "missing" {
				_, err = s.Replace(context.Background(), req, false)
				if err == nil {
					t.Fatal("drift accepted")
				}
			}
			if kind == "unknown-file" || kind == "unknown-directory" || kind == "symlink" || kind == "directory-type" {
				_, err = s.Replace(context.Background(), req, true)
				if err == nil {
					t.Fatal("force bypassed ownership/type")
				}
			}
		})
	}
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func TestInspectUnknownAndSymlinkRoots(t *testing.T) {
	for _, kind := range []string{"unknown", "root-link", "ancestor-link"} {
		t.Run(kind, func(t *testing.T) {
			home := t.TempDir()
			req, data := fixture(t, home, "1.0.0")
			s := Service{Home: home, Fetcher: data}
			must(t, os.MkdirAll(filepath.Dir(req.Root), 0755))
			switch kind {
			case "unknown":
				must(t, os.Mkdir(req.Root, 0755))
			case "root-link":
				must(t, os.Symlink(t.TempDir(), req.Root))
			case "ancestor-link":
				must(t, os.Remove(filepath.Dir(req.Root)))
				must(t, os.Symlink(t.TempDir(), filepath.Dir(req.Root)))
			}
			_, err := s.Inspect(req)
			var conflict *ConflictError
			if !errors.As(err, &conflict) {
				t.Fatalf("expected typed conflict: %v", err)
			}
		})
	}
}
func TestInspectAbsentDoesNotWrite(t *testing.T) {
	home := t.TempDir()
	req, _ := fixture(t, home, "1.0.0")
	s := Service{Home: home}
	_, err := s.Inspect(req)
	must(t, err)
	entries, err := os.ReadDir(home)
	must(t, err)
	if len(entries) != 0 {
		t.Fatal(entries)
	}
}

func TestMatchingUnownedRootIsNotAdopted(t *testing.T) {
	s, req := installed(t)
	must(t, packagestate.DeleteReceipt(s.Home, req.Recipe))
	_, err := s.Replace(context.Background(), req, true)
	var conflict *ConflictError
	if !errors.As(err, &conflict) || conflict.Kind != "unowned-root" {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(req.Root, "dir/tool"))
	must(t, err)
	if string(data) != "1.0.0" {
		t.Fatal("matching unowned bytes mutated")
	}
}
