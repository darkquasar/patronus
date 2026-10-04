package packagestate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/darkquasar/patronus/internal/packagebundle"
)

func removalFixture(t *testing.T, home string, count int) *Transaction {
	t.Helper()
	r := receiptFixture(t, home)
	r.Files = nil
	tx := &Transaction{SchemaVersion: 1, Recipe: r.Recipe, Root: r.Root, Operation: "remove", Phase: Prepared, Previous: r}
	for i := 0; i < count; i++ {
		e := packagebundle.Entry{Path: fmt.Sprintf("file-%05d", i), Mode: 0644, SHA256: "sha256:" + strings.Repeat("a", 64)}
		r.Files = append(r.Files, e)
		tx.Observed = append(tx.Observed, e)
		tx.PendingRemove = append(tx.PendingRemove, e.Path)
	}
	return tx
}

func stateMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemovalCheckpointBoundedAndManifestImmutable(t *testing.T) {
	home := t.TempDir()
	tx := removalFixture(t, home, 1000)
	stateMust(t, PrepareRemoval(home, tx))
	stateMust(t, WriteTransaction(home, tx))
	_, _, path, err := transactionPath(home, tx.Recipe)
	stateMust(t, err)
	manifest := filepath.Join(filepath.Dir(path), "removal.json")
	before, err := os.ReadFile(manifest)
	stateMust(t, err)
	info, err := os.Stat(manifest)
	stateMust(t, err)
	// Caller-owned input and full-reader inventories cannot mutate private state.
	tx.Previous.Files[0].Path = "unknown"
	tx.Observed[0].Path = "unknown"
	tx.PendingRemove[0] = "unknown"
	for i := 0; i < 1000; i++ {
		e, ok := tx.NextRemoval()
		if !ok || e.Path != fmt.Sprintf("file-%05d", i) {
			t.Fatalf("invalid private selection: %+v", e)
		}
		tx.Intent = "unlink"
		stateMust(t, WriteTransaction(home, tx))
		stateMust(t, tx.CompleteRemoval(e.Path))
		stateMust(t, WriteTransaction(home, tx))
	}
	checkpoint, err := os.ReadFile(path)
	stateMust(t, err)
	if len(checkpoint) > 2048 || bytes.Contains(checkpoint, []byte("file-")) || bytes.Contains(checkpoint, []byte("previous")) {
		t.Fatalf("checkpoint contains inventory or exceeds bound: %d bytes", len(checkpoint))
	}
	after, err := os.ReadFile(manifest)
	stateMust(t, err)
	latest, err := os.Stat(manifest)
	stateMust(t, err)
	if !bytes.Equal(before, after) || !os.SameFile(info, latest) || !info.ModTime().Equal(latest.ModTime()) {
		t.Fatal("manifest was rewritten")
	}
	got, err := ReadTransaction(home, tx.Recipe)
	stateMust(t, err)
	if len(got.Removed) != 1000 || len(got.PendingRemove) != 0 || got.RemovalReceipt() != nil {
		t.Fatal("full reader did not reconstruct completed prefix")
	}
	got.Previous.Files[0].Path = "unknown"
	if got.RemovalPrevious().Files[0].Path != "file-00000" {
		t.Fatal("reader view aliases private manifest")
	}
}

func TestRemovalCheckpointRejectsCorruption(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Transaction)
	}{
		{"negative", func(tx *Transaction) { tx.Completed = -1 }},
		{"overflow", func(tx *Transaction) { tx.Completed = 3 }},
		{"id", func(tx *Transaction) { tx.RemovalID = strings.Repeat("0", 64) }},
		{"digest", func(tx *Transaction) { tx.ManifestHash = "sha256:" + strings.Repeat("0", 64) }},
		{"root", func(tx *Transaction) { tx.Root += "-other" }},
		{"recipe", func(tx *Transaction) { tx.Recipe = "other" }},
		{"phase", func(tx *Transaction) { tx.Phase = PackagePlaced }},
		{"early-receipt", func(tx *Transaction) { tx.Intent = "save-receipt" }},
		{"early-commit", func(tx *Transaction) { tx.Phase = Committed; tx.Intent = "drop-reference" }},
		{"detail", func(tx *Transaction) { tx.Detail = strings.Repeat("x", 1025) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			tx := removalFixture(t, home, 2)
			stateMust(t, PrepareRemoval(home, tx))
			stateMust(t, WriteTransaction(home, tx))
			_, _, path, err := transactionPath(home, tx.Recipe)
			stateMust(t, err)
			tc.mutate(tx)
			stateMust(t, (storage{}).writeJSON(path, tx.removalCheckpoint()))
			if _, err := ReadTransaction(home, "pi-sandbox"); err == nil {
				t.Fatal("accepted corrupt checkpoint")
			}
		})
	}
}

func TestRemovalManifestRejectsCorruption(t *testing.T) {
	for _, corruption := range []string{"missing", "truncated", "null", "unknown-field", "changed-baseline", "symlink"} {
		t.Run(corruption, func(t *testing.T) {
			home := t.TempDir()
			tx := removalFixture(t, home, 2)
			stateMust(t, PrepareRemoval(home, tx))
			stateMust(t, WriteTransaction(home, tx))
			_, _, path, err := transactionPath(home, tx.Recipe)
			stateMust(t, err)
			path = filepath.Join(filepath.Dir(path), "removal.json")
			switch corruption {
			case "missing":
				stateMust(t, os.Remove(path))
			case "truncated":
				stateMust(t, os.WriteFile(path, []byte(`{"schemaVersion":`), 0600))
			case "null":
				stateMust(t, os.WriteFile(path, []byte(`null`), 0600))
			case "unknown-field":
				stateMust(t, os.WriteFile(path, []byte(`{"unknown":true}`), 0600))
			case "changed-baseline":
				tx.removal.manifest.Selected[0].Mode = 0700
				stateMust(t, (storage{}).writeJSON(path, tx.removal.manifest))
			case "symlink":
				stateMust(t, os.Remove(path))
				stateMust(t, os.Symlink(filepath.Join(t.TempDir(), "outside"), path))
			}
			if _, err := ReadTransaction(home, tx.Recipe); err == nil {
				t.Fatal("accepted invalid manifest")
			}
		})
	}
}

func TestRemovalManifestValidatesOwnershipOnce(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Transaction)
	}{
		{"unowned", func(tx *Transaction) { tx.Observed[0].Path = "unknown"; tx.PendingRemove[0] = "unknown" }},
		{"missing-unowned", func(tx *Transaction) { tx.Removed = []string{"unknown"} }},
		{"overlap", func(tx *Transaction) { tx.Removed = []string{tx.Observed[0].Path} }},
		{"order", func(tx *Transaction) {
			tx.PendingRemove[0], tx.PendingRemove[1] = tx.PendingRemove[1], tx.PendingRemove[0]
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := removalFixture(t, t.TempDir(), 2)
			tc.mutate(tx)
			home := filepath.Dir(filepath.Dir(filepath.Dir(tx.Root)))
			if err := PrepareRemoval(home, tx); err == nil {
				t.Fatal("accepted invalid ownership")
			}
		})
	}
}

func TestRemovalPublicationDurability(t *testing.T) {
	for _, target := range []string{"removal.json", "transaction.json"} {
		for _, point := range []string{"before-temp-sync", "after-temp-sync", "before-rename", "after-rename", "before-parent-sync", "after-parent-sync"} {
			t.Run(target+"/"+point, func(t *testing.T) {
				home := t.TempDir()
				tx := removalFixture(t, home, 2)
				s := storage{fault: func(p, path string) error {
					if p == point && filepath.Base(path) == target {
						return syscall.ENOSPC
					}
					return nil
				}}
				err := s.prepareRemoval(home, tx)
				if err == nil {
					err = s.writeTransaction(home, tx)
				}
				var de *DurabilityError
				visible := point == "after-rename" || point == "before-parent-sync" || point == "after-parent-sync"
				if !errors.Is(err, syscall.ENOSPC) || !errors.As(err, &de) || de.MayBeVisible != visible {
					t.Fatalf("fault not preserved: %v", err)
				}
				got, err := ReadTransaction(home, tx.Recipe)
				stateMust(t, err)
				if (got != nil) != (target == "transaction.json" && visible) {
					t.Fatal("unexpected publication visibility")
				}
			})
		}
	}
}

func TestRemovalCheckpointNeedsPrivateStateAndRejectsLegacyReader(t *testing.T) {
	home := t.TempDir()
	tx := removalFixture(t, home, 2)
	stateMust(t, PrepareRemoval(home, tx))
	stateMust(t, WriteTransaction(home, tx))
	_, root, path, err := transactionPath(home, tx.Recipe)
	stateMust(t, err)
	var oldView *Transaction
	_, err = readJSON(path, &oldView)
	stateMust(t, err)
	// Schema-1's validator rejects the discovery marker before mutation. Its
	// strict decoder additionally rejects the new fields in the actual old binary.
	if err := validateTransaction(home, tx.Recipe, root, oldView); err == nil {
		t.Fatal("legacy validator accepted new marker")
	}
	if err := WriteTransaction(home, oldView); err == nil {
		t.Fatal("unvalidated caller struct authorized progress")
	}
	if err := tx.CompleteRemoval("file-00001"); err == nil {
		t.Fatal("out-of-order completion accepted")
	}
}

func TestRemovalPendingReceiptReadersAndClear(t *testing.T) {
	home := t.TempDir()
	tx := removalFixture(t, home, 2)
	stateMust(t, Save(home, tx.Previous))
	stateMust(t, PrepareRemoval(home, tx))
	stateMust(t, WriteTransaction(home, tx))
	tx.Intent = "unlink"
	stateMust(t, WriteTransaction(home, tx))
	stateMust(t, tx.CompleteRemoval("file-00000"))
	stateMust(t, WriteTransaction(home, tx))
	r, err := Load(home, tx.Recipe)
	stateMust(t, err)
	list, err := List(home)
	stateMust(t, err)
	if !reflect.DeepEqual(r, tx.Previous) || len(list) != 1 || len(list[0].Files) != 2 {
		t.Fatal("committed reader silently published intermediate receipt")
	}
	got, err := ReadTransaction(home, tx.Recipe)
	stateMust(t, err)
	if got.Completed != 1 || len(got.Removed) != 1 || len(got.PendingRemove) != 1 || len(got.RemovalReceipt().Files) != 1 {
		t.Fatal("pending view lost prefix")
	}
	// Keep the marker authoritative until its durable deletion, then clean the
	// manifest. A cleanup fault cannot leave a live marker with no manifest.
	s := storage{fault: func(point, path string) error {
		if point == "before-unlink" && filepath.Base(path) == "removal.json" {
			return syscall.ENOSPC
		}
		return nil
	}}
	if err := s.clearTransaction(home, tx.Recipe); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("lost cleanup error: %v", err)
	}
	got, err = ReadTransaction(home, tx.Recipe)
	stateMust(t, err)
	if got != nil {
		t.Fatal("marker remains after manifest cleanup started")
	}
	stateMust(t, ClearTransaction(home, tx.Recipe))
	// No custom MarshalJSON is involved in checkpoint persistence.
	data, err := json.Marshal(tx.removalCheckpoint())
	stateMust(t, err)
	if bytes.Contains(data, []byte("observed")) {
		t.Fatal("checkpoint contains observation inventory")
	}
}

func TestRemovalProgressDurabilityFaults(t *testing.T) {
	for _, point := range []string{"before-temp-sync", "after-temp-sync", "before-rename", "after-rename", "before-parent-sync", "after-parent-sync"} {
		t.Run(point, func(t *testing.T) {
			home := t.TempDir()
			tx := removalFixture(t, home, 2)
			stateMust(t, PrepareRemoval(home, tx))
			tx.Intent = "unlink"
			stateMust(t, WriteTransaction(home, tx))
			stateMust(t, tx.CompleteRemoval("file-00000"))
			s := storage{fault: func(p, _ string) error {
				if p == point {
					return syscall.ENOSPC
				}
				return nil
			}}
			err := s.writeTransaction(home, tx)
			var de *DurabilityError
			visible := point == "after-rename" || point == "before-parent-sync" || point == "after-parent-sync"
			if !errors.Is(err, syscall.ENOSPC) || !errors.As(err, &de) || de.MayBeVisible != visible {
				t.Fatalf("lost progress fault: %v", err)
			}
			got, err := ReadTransaction(home, tx.Recipe)
			stateMust(t, err)
			want := 0
			if visible {
				want = 1
			}
			if got.Completed != want || len(got.PendingRemove) != 2-want {
				t.Fatalf("progress view: %+v", got)
			}
		})
	}
}

func TestRemovalCheckpointBoundsDiagnosticWithoutCorruptingUTF8(t *testing.T) {
	home := t.TempDir()
	tx := removalFixture(t, home, 1)
	stateMust(t, PrepareRemoval(home, tx))
	tx.Phase, tx.ResumePhase = RecoveryRequired, Prepared
	tx.Detail = strings.Repeat("\xff€", 1024)
	stateMust(t, WriteTransaction(home, tx))
	got, err := ReadTransaction(home, tx.Recipe)
	stateMust(t, err)
	if len(got.Detail) > 1024 || len(got.Detail) == 0 {
		t.Fatal("diagnostic not bounded")
	}
}
