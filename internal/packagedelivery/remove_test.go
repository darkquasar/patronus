package packagedelivery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagestate"
)

func TestPackageRemovalClean(t *testing.T) {
	s, req := installed(t)
	result, err := s.Remove(context.Background(), req.Recipe, false)
	must(t, err)
	if result.Receipt != nil || !result.Mutated {
		t.Fatalf("result: %+v", result)
	}
	if _, err := os.Stat(req.Root); !os.IsNotExist(err) {
		t.Fatalf("root survives: %v", err)
	}
	tx, err := packagestate.ReadTransaction(s.Home, req.Recipe)
	must(t, err)
	if tx == nil || tx.Phase != packagestate.Committed || tx.Operation != "remove" {
		t.Fatalf("missing acknowledgement journal: %+v", tx)
	}
}

func TestPackageRemovalRetainsEditsAndUnknown(t *testing.T) {
	s, req := installed(t)
	must(t, os.WriteFile(filepath.Join(req.Root, "dir/tool"), []byte("edit"), 0755))
	must(t, os.WriteFile(filepath.Join(req.Root, "mine"), []byte("unknown"), 0644))
	result, err := s.Remove(context.Background(), req.Recipe, false)
	must(t, err)
	if len(result.Retained) != 1 || result.Retained[0] != "dir/tool" || len(result.Leftovers) != 1 || result.Leftovers[0] != "mine" {
		t.Fatalf("result: %+v", result)
	}
	if result.Receipt == nil || len(result.Receipt.Files) != 1 {
		t.Fatalf("receipt: %+v", result.Receipt)
	}
	must(t, packagestate.ClearTransaction(s.Home, req.Recipe))
	result, err = s.Remove(context.Background(), req.Recipe, true)
	must(t, err)
	if result.Receipt != nil || len(result.Retained) != 0 {
		t.Fatalf("result: %+v", result)
	}
	data, err := os.ReadFile(filepath.Join(req.Root, "mine"))
	must(t, err)
	if string(data) != "unknown" {
		t.Fatal("unknown changed")
	}
}

func TestPackageRemovalHardCrashRecovery(t *testing.T) {
	for _, point := range []string{"after-unlink-intent", "after-unlink", "after-reduced-receipt-write", "after-removal-commit"} {
		t.Run(point, func(t *testing.T) {
			home := t.TempDir()
			setupCrashBaseline(t, home, true)
			runCrash(t, home, point, "PATRONUS_REMOVE=1")
			locked(t, home)
			s := Service{Home: home}
			must(t, s.Recover(context.Background(), "kit"))
			r, err := packagestate.Load(home, "kit")
			must(t, err)
			if r != nil {
				t.Fatalf("receipt survives: %+v", r)
			}
			tx, err := packagestate.ReadTransaction(home, "kit")
			must(t, err)
			if tx == nil || tx.Phase != packagestate.Committed {
				t.Fatalf("journal: %+v", tx)
			}
			if _, err := os.Stat(tx.Root); !os.IsNotExist(err) {
				t.Fatalf("root survives: %v", err)
			}
		})
	}
}

func TestPackageRemovalCrashPreservesLaterEdit(t *testing.T) {
	home := t.TempDir()
	setupCrashBaseline(t, home, true)
	runCrash(t, home, "after-unlink-intent", "PATRONUS_REMOVE=1")
	locked(t, home)
	path := filepath.Join(home, ".patronus", "packages", "kit", "dir/tool")
	must(t, os.WriteFile(path, []byte("later edit"), 0755))
	s := Service{Home: home}
	err := s.Recover(context.Background(), "kit")
	if !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("expected recovery conflict: %v", err)
	}
	data, err := os.ReadFile(path)
	must(t, err)
	if string(data) != "later edit" {
		t.Fatal("later edit overwritten")
	}
	tx, err := packagestate.ReadTransaction(home, "kit")
	must(t, err)
	if tx.Phase != packagestate.RecoveryRequired || tx.ResumePhase != packagestate.Prepared {
		t.Fatalf("journal: %+v", tx)
	}
	// Reconciliation to the force-authorized baseline permits retry, not rollback.
	must(t, os.WriteFile(path, []byte("actual user preimage"), 0755))
	must(t, s.Recover(context.Background(), "kit"))
}

func TestPackageRemovalMissingOwnedFile(t *testing.T) {
	s, req := installed(t)
	must(t, os.Remove(filepath.Join(req.Root, "dir/tool")))
	result, err := s.Remove(context.Background(), req.Recipe, false)
	must(t, err)
	if result.Receipt != nil || len(result.Retained) > 0 {
		t.Fatalf("result: %+v", result)
	}
}

func TestPackageRemovalMissingReceiptAuthorizesNothing(t *testing.T) {
	s, req := installed(t)
	must(t, packagestate.DeleteReceipt(s.Home, req.Recipe))
	if _, err := s.Remove(context.Background(), req.Recipe, true); err == nil {
		t.Fatal("missing receipt accepted")
	}
	if _, err := os.Stat(filepath.Join(req.Root, "dir/tool")); err != nil {
		t.Fatal(err)
	}
}

func TestPackageRemovalRejectsSymlinkAncestor(t *testing.T) {
	s, req := installed(t)
	outside := t.TempDir()
	must(t, os.WriteFile(filepath.Join(outside, "tool"), []byte("outside"), 0755))
	must(t, os.Remove(filepath.Join(req.Root, "dir/tool")))
	must(t, os.Remove(filepath.Join(req.Root, "dir")))
	must(t, os.Symlink(outside, filepath.Join(req.Root, "dir")))
	if _, err := s.Remove(context.Background(), req.Recipe, true); err == nil {
		t.Fatal("symlink accepted")
	}
	data, err := os.ReadFile(filepath.Join(outside, "tool"))
	must(t, err)
	if string(data) != "outside" {
		t.Fatal("outside content changed")
	}
}

func TestPackageRemovalRejectsUnownedJournalIntent(t *testing.T) {
	s, req := installed(t)
	r, err := packagestate.Load(s.Home, req.Recipe)
	must(t, err)
	path := filepath.Join(req.Root, "mine")
	must(t, os.WriteFile(path, []byte("unknown"), 0644))
	tx := &packagestate.Transaction{SchemaVersion: 1, Recipe: req.Recipe, Root: req.Root, Operation: "remove", Phase: packagestate.Prepared, Previous: r, Observed: []packagebundle.Entry{{Path: "mine", Mode: 0644, SHA256: bytesDigest([]byte("unknown"))}}, PendingRemove: []string{"mine"}}
	must(t, packagestate.WriteTransaction(s.Home, tx))
	if err := s.Recover(context.Background(), req.Recipe); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("unowned intent accepted: %v", err)
	}
	data, err := os.ReadFile(path)
	must(t, err)
	if string(data) != "unknown" {
		t.Fatal("unowned file changed")
	}
}

func TestPackageRemovalPartialPersistenceRetry(t *testing.T) {
	s, req := installed(t)
	sentinel := errors.New("disk error")
	s.persistence = &persistence{save: func(string, *packagestate.Receipt) error { return sentinel }}
	_, err := s.Remove(context.Background(), req.Recipe, false)
	if !errors.Is(err, sentinel) {
		t.Fatalf("persistence error lost: %v", err)
	}
	if _, err := os.Stat(filepath.Join(req.Root, "dir/tool")); !os.IsNotExist(err) {
		t.Fatalf("first deletion not completed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(req.Root, "package.json")); err != nil {
		t.Fatal("continued after receipt error", err)
	}
	s.persistence = nil
	must(t, s.Recover(context.Background(), req.Recipe))
	if _, err := os.Stat(req.Root); !os.IsNotExist(err) {
		t.Fatalf("retry did not finish: %v", err)
	}
}

func TestPackageRemovalAmbiguousReceiptStops(t *testing.T) {
	for _, visibleWrite := range []bool{false, true} {
		t.Run(fmt.Sprint(visibleWrite), func(t *testing.T) {
			s, req := installed(t)
			s.persistence = &persistence{save: func(home string, r *packagestate.Receipt) error {
				if visibleWrite {
					must(t, packagestate.Save(home, r))
				}
				return &packagestate.DurabilityError{Path: "kit.json", Stage: "test-sync", MayBeVisible: true, Err: errors.New("ambiguous")}
			}}
			_, err := s.Remove(context.Background(), req.Recipe, false)
			if !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("ambiguous error lost: %v", err)
			}
			if _, err := os.Stat(filepath.Join(req.Root, "package.json")); err != nil {
				t.Fatal("destructive follow-up", err)
			}
			s.persistence = nil
			must(t, s.Recover(context.Background(), req.Recipe))
		})
	}
}

func TestPackageRemovalModeDrift(t *testing.T) {
	s, req := installed(t)
	path := filepath.Join(req.Root, "dir/tool")
	must(t, os.Chmod(path, 0700))
	result, err := s.Remove(context.Background(), req.Recipe, false)
	must(t, err)
	if len(result.Retained) != 1 || result.Retained[0] != "dir/tool" {
		t.Fatalf("mode drift not retained: %+v", result)
	}
	must(t, packagestate.ClearTransaction(s.Home, req.Recipe))
	result, err = s.Remove(context.Background(), req.Recipe, true)
	must(t, err)
	if result.Receipt != nil {
		t.Fatal("force did not remove owned mode drift")
	}
}

func TestPackageRemovalRevalidatesAfterIntent(t *testing.T) {
	s, req := installed(t)
	path := filepath.Join(req.Root, "dir/tool")
	s.Fault = func(point string) error {
		if point == "after-unlink-intent" {
			return os.WriteFile(path, []byte("intervening"), 0755)
		}
		return nil
	}
	_, err := s.Remove(context.Background(), req.Recipe, true)
	if !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("expected conflict: %v", err)
	}
	if data := mustReadRemoval(t, path); string(data) != "intervening" {
		t.Fatal("intervening edit deleted")
	}
}

func mustReadRemoval(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	must(t, err)
	return data
}

func TestPackageRemovalCorruptReceipt(t *testing.T) {
	s, req := installed(t)
	must(t, os.WriteFile(filepath.Join(s.Home, ".patronus", "package-state", "kit.json"), []byte("broken"), 0600))
	if _, err := s.Remove(context.Background(), req.Recipe, true); err == nil {
		t.Fatal("corrupt receipt accepted")
	}
	if string(mustReadRemoval(t, filepath.Join(req.Root, "dir/tool"))) != "1.0.0" {
		t.Fatal("payload changed")
	}
}

func TestPackageRemovalDoesNotDeleteRecreatedCompletedPath(t *testing.T) {
	home := t.TempDir()
	setupCrashBaseline(t, home, false)
	runCrash(t, home, "after-reduced-receipt-write", "PATRONUS_REMOVE=1")
	locked(t, home)
	path := filepath.Join(home, ".patronus", "packages", "kit", "dir/tool")
	must(t, os.WriteFile(path, []byte("new user file"), 0755))
	s := Service{Home: home}
	must(t, s.Recover(context.Background(), "kit"))
	if string(mustReadRemoval(t, path)) != "new user file" {
		t.Fatal("completed deletion was replayed")
	}
	result, err := s.Remove(context.Background(), "kit", true)
	must(t, err)
	if len(result.Leftovers) != 1 || result.Leftovers[0] != "dir/tool" {
		t.Fatalf("unowned recreated path not reported: %+v", result)
	}
}

func TestPackageRemovalRejectsTypeSubstitutionAfterIntent(t *testing.T) {
	for _, kind := range []string{"directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			s, req := installed(t)
			path := filepath.Join(req.Root, "dir/tool")
			outside := filepath.Join(t.TempDir(), "outside")
			must(t, os.WriteFile(outside, []byte("outside"), 0644))
			s.Fault = func(point string) error {
				if point != "after-unlink-intent" {
					return nil
				}
				if err := os.Remove(path); err != nil {
					return err
				}
				if kind == "directory" {
					return os.Mkdir(path, 0755)
				}
				return os.Symlink(outside, path)
			}
			_, err := s.Remove(context.Background(), req.Recipe, true)
			if !errors.Is(err, ErrRecoveryRequired) {
				t.Fatalf("type substitution accepted: %v", err)
			}
			info, err := os.Lstat(path)
			must(t, err)
			if info.Mode().IsRegular() {
				t.Fatal("substitution overwritten")
			}
			if string(mustReadRemoval(t, outside)) != "outside" {
				t.Fatal("outside changed")
			}
		})
	}
}
