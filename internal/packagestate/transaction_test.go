package packagestate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/packagebundle"
)

func transactionFixture(t *testing.T, home string) *Transaction {
	r := receiptFixture(t, home)
	base := filepath.Join(filepath.Dir(r.Root), ".txn", r.Recipe)
	return &Transaction{SchemaVersion: 1, Recipe: r.Recipe, Root: r.Root, Stage: filepath.Join(base, "stage-123"), Backup: filepath.Join(base, "backup-123"), Phase: Prepared, Operation: "replace", Intent: "move-old", Candidate: r}
}
func TestTransactionScopedMemberRoundTrip(t *testing.T) {
	home := t.TempDir()
	r := cp05ScopedReceipt(t, home)
	tx := &Transaction{
		SchemaVersion: 1, Recipe: r.Recipe, Root: r.Root,
		Phase: RecoveryRequired, ResumePhase: Prepared, Operation: "remove", Intent: "unlink",
		Previous: r, Observed: r.Files,
		PendingRemove: []string{"node_modules/@example/tool/index.js"},
		Removed:       []string{"node_modules/@example/tool/old.js"},
	}
	if err := WriteTransaction(home, tx); err != nil {
		t.Fatal(err)
	}
	got, err := ReadTransaction(home, tx.Recipe)
	if err != nil || !reflect.DeepEqual(got, tx) {
		t.Fatalf("scoped transaction: %#v, %v", got, err)
	}
}

func TestTransactionRoundTrip(t *testing.T) {
	home := t.TempDir()
	tx := transactionFixture(t, home)
	if err := WriteTransaction(home, tx); err != nil {
		t.Fatal(err)
	}
	got, err := ReadTransaction(home, tx.Recipe)
	if err != nil || !reflect.DeepEqual(got, tx) {
		t.Fatalf("got %#v %v", got, err)
	}
	if err := ClearTransaction(home, tx.Recipe); err != nil {
		t.Fatal(err)
	}
	if err := ClearTransaction(home, tx.Recipe); err != nil {
		t.Fatal(err)
	}
}
func TestTransactionRejectsEscape(t *testing.T) {
	for _, path := range []string{"/tmp/escape", "../escape", "", "other"} {
		t.Run(path, func(t *testing.T) {
			home := t.TempDir()
			tx := transactionFixture(t, home)
			tx.Stage = path
			if err := WriteTransaction(home, tx); err == nil {
				t.Fatal("accepted escape")
			}
		})
	}
}

func TestTransactionRecoveryAndRemoval(t *testing.T) {
	home := t.TempDir()
	tx := transactionFixture(t, home)
	tx.Operation = "remove"
	tx.Stage = ""
	tx.Backup = ""
	tx.Candidate = nil
	tx.Previous = receiptFixture(t, home)
	tx.Observed = tx.Previous.Files
	tx.Removed = []string{"old.txt"}
	tx.PendingRemove = []string{"package.json"}
	tx.Phase = RecoveryRequired
	tx.ResumePhase = Prepared
	tx.Intent = "unlink"
	tx.Detail = "retry removal"
	if err := WriteTransaction(home, tx); err != nil {
		t.Fatal(err)
	}
	got, err := ReadTransaction(home, tx.Recipe)
	if err != nil || !reflect.DeepEqual(got, tx) {
		t.Fatalf("got %#v %v", got, err)
	}
}
func TestTransactionSymlinkPaths(t *testing.T) {
	for _, rel := range []string{".patronus/package-state/transactions", ".patronus/package-state/transactions/pi-sandbox", ".patronus/package-state/transactions/pi-sandbox/transaction.json", ".patronus/packages/.txn", ".patronus/packages/.txn/pi-sandbox", ".patronus/packages/.txn/pi-sandbox/stage-123"} {
		t.Run(rel, func(t *testing.T) {
			home := t.TempDir()
			p := filepath.Join(home, rel)
			if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(t.TempDir(), p); err != nil {
				t.Fatal(err)
			}
			if err := WriteTransaction(home, transactionFixture(t, home)); err == nil || !strings.Contains(err.Error(), "unsupported symlink") {
				t.Fatalf("got %v", err)
			}
		})
	}
}
func TestTransactionInvalidRecords(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Transaction)
	}{
		{"schema", func(tx *Transaction) { tx.SchemaVersion = 2 }}, {"root", func(tx *Transaction) { tx.Root = "/tmp/escape" }}, {"backup", func(tx *Transaction) {
			tx.Backup = filepath.Join(filepath.Dir(filepath.Dir(tx.Backup)), "other", "backup")
		}}, {"phase", func(tx *Transaction) { tx.Phase = "future" }}, {"resume", func(tx *Transaction) { tx.Phase = RecoveryRequired; tx.ResumePhase = "future" }}, {"operation", func(tx *Transaction) { tx.Operation = "execute" }}, {"intent", func(tx *Transaction) { tx.Intent = "unlink" }}, {"candidate", func(tx *Transaction) { tx.Candidate.Root = "/tmp/escape" }}, {"observed", func(tx *Transaction) { tx.Observed = []packagebundle.Entry{{Path: "../outside", Mode: 0644}} }}, {"pending", func(tx *Transaction) { tx.PendingRemove = []string{"../outside"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			tx := transactionFixture(t, home)
			tc.mutate(tx)
			if err := WriteTransaction(home, tx); err == nil {
				t.Fatal("accepted invalid transaction")
			}
		})
	}
}
func TestTransactionDurabilityFaults(t *testing.T) {
	for _, point := range []string{"before-temp-sync", "after-temp-sync", "before-rename", "after-rename", "before-parent-sync", "after-parent-sync"} {
		t.Run(point, func(t *testing.T) {
			home := t.TempDir()
			tx := transactionFixture(t, home)
			if err := WriteTransaction(home, tx); err != nil {
				t.Fatal(err)
			}
			tx.Phase = PackagePlaced
			injected := errors.New("disk fault")
			s := storage{fault: func(p, _ string) error {
				if p == point {
					return injected
				}
				return nil
			}}
			err := s.writeTransaction(home, tx)
			var de *DurabilityError
			visible := point == "after-rename" || point == "before-parent-sync" || point == "after-parent-sync"
			if !errors.As(err, &de) || !errors.Is(err, injected) || de.MayBeVisible != visible {
				t.Fatalf("got %#v %v", de, err)
			}
			got, err := ReadTransaction(home, tx.Recipe)
			if err != nil {
				t.Fatal(err)
			}
			want := Prepared
			if visible {
				want = PackagePlaced
			}
			if got.Phase != want {
				t.Fatalf("phase %s", got.Phase)
			}
		})
	}
}
func TestTransactionClearDurability(t *testing.T) {
	for _, point := range []string{"before-unlink", "after-unlink", "before-parent-sync", "after-parent-sync"} {
		t.Run(point, func(t *testing.T) {
			home := t.TempDir()
			tx := transactionFixture(t, home)
			if err := WriteTransaction(home, tx); err != nil {
				t.Fatal(err)
			}
			s := storage{fault: func(p, _ string) error {
				if p == point {
					return errors.New("disk fault")
				}
				return nil
			}}
			err := s.clearTransaction(home, tx.Recipe)
			var de *DurabilityError
			if !errors.As(err, &de) || de.MayBeVisible != (point != "before-unlink") {
				t.Fatalf("got %#v %v", de, err)
			}
			got, err := ReadTransaction(home, tx.Recipe)
			if err != nil {
				t.Fatal(err)
			}
			if (got == nil) != (point != "before-unlink") {
				t.Fatalf("got %#v", got)
			}
		})
	}
}
func TestTransactionStrictJSON(t *testing.T) {
	for _, data := range []string{`null`, `{"schemaVersion":2}`, `{"unknown":1}`, `{} {}`} {
		t.Run(data, func(t *testing.T) {
			home := t.TempDir()
			tx := transactionFixture(t, home)
			if err := WriteTransaction(home, tx); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(home, ".patronus/package-state/transactions/pi-sandbox/transaction.json")
			if err := os.WriteFile(p, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadTransaction(home, tx.Recipe); err == nil {
				t.Fatal("accepted invalid JSON")
			}
		})
	}
}

func TestTransactionObservedEditedMode(t *testing.T) {
	home := t.TempDir()
	tx := transactionFixture(t, home)
	tx.Observed = append(tx.Observed, tx.Candidate.Files...)
	tx.Observed[0].Mode = 0600
	if err := WriteTransaction(home, tx); err != nil {
		t.Fatal(err)
	}
	got, err := ReadTransaction(home, tx.Recipe)
	if err != nil || got.Observed[0].Mode != 0600 {
		t.Fatalf("got %#v %v", got, err)
	}
}

func TestTransactionObservedModeEncoding(t *testing.T) {
	for _, mode := range []uint32{0600, 0700, 04755, 02755, 01755, 07777} {
		t.Run(fmt.Sprintf("%o", mode), func(t *testing.T) {
			home := t.TempDir()
			tx := transactionFixture(t, home)
			tx.Observed = append(tx.Observed, tx.Candidate.Files...)
			tx.Observed[0].Mode = mode
			if err := WriteTransaction(home, tx); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, mode := range []uint32{010000, uint32(os.ModeSymlink), uint32(os.ModeDir)} {
		t.Run(fmt.Sprintf("invalid-%o", mode), func(t *testing.T) {
			home := t.TempDir()
			tx := transactionFixture(t, home)
			tx.Observed = append(tx.Observed, tx.Candidate.Files...)
			tx.Observed[0].Mode = mode
			if err := WriteTransaction(home, tx); err == nil {
				t.Fatal("accepted non-permission bits")
			}
		})
	}
}
func TestTransactionCandidateDoesNotReplaceReceipt(t *testing.T) {
	home := t.TempDir()
	r := receiptFixture(t, home)
	if err := Save(home, r); err != nil {
		t.Fatal(err)
	}
	tx := transactionFixture(t, home)
	tx.Candidate.RecipeVersion = "2.0.0"
	if err := WriteTransaction(home, tx); err != nil {
		t.Fatal(err)
	}
	got, err := Load(home, r.Recipe)
	if err != nil || got.RecipeVersion != "1.0.0" {
		t.Fatalf("committed receipt changed: %#v %v", got, err)
	}
}

func TestTransactionRejectsCompoundIntents(t *testing.T) {
	for _, tc := range []struct{ operation, intent string }{
		{"replace", "move-old promote-new"},
		{"replace", "commit cleanup"},
		{"metadata", "save-receipt commit"},
		{"remove", "commit cleanup"},
	} {
		t.Run(tc.operation+"/"+tc.intent, func(t *testing.T) {
			home := t.TempDir()
			tx := transactionFixture(t, home)
			tx.Operation, tx.Intent = tc.operation, tc.intent
			if err := WriteTransaction(home, tx); err == nil {
				t.Fatal("accepted compound intent")
			}
		})
	}
}

func TestTransactionRetryRepairsAncestorDurability(t *testing.T) {
	for _, rel := range []string{".patronus", ".patronus/package-state", ".patronus/package-state/transactions", ".patronus/package-state/transactions/pi-sandbox"} {
		t.Run(rel, func(t *testing.T) {
			home, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			tx := transactionFixture(t, home)
			ancestor := filepath.Join(home, rel)
			injected := errors.New("ancestor sync failed")
			s := storage{fault: func(point, path string) error {
				if point == "before-ancestor-sync" && path == ancestor {
					return injected
				}
				return nil
			}}
			for attempt := 0; attempt < 2; attempt++ {
				err := s.writeTransaction(home, tx)
				var de *DurabilityError
				if !errors.Is(err, injected) || !errors.As(err, &de) || !de.MayBeVisible {
					t.Fatalf("attempt %d: expected uncertain ancestor durability, got %v", attempt, err)
				}
				if info, err := os.Lstat(ancestor); err != nil || !info.IsDir() {
					t.Fatalf("ancestor not visible: %v", err)
				}
				if got, err := ReadTransaction(home, tx.Recipe); err != nil || got != nil {
					t.Fatalf("journal written before ancestor repair: %#v %v", got, err)
				}
			}
			repaired := false
			s.fault = func(point, path string) error {
				if point == "after-ancestor-sync" && path == ancestor {
					repaired = true
				}
				if point == "before-rename" && !repaired {
					t.Fatal("journal published before ancestor repair")
				}
				return nil
			}
			if err := s.writeTransaction(home, tx); err != nil {
				t.Fatal(err)
			}
			if !repaired {
				t.Fatal("retry skipped existing ancestor link")
			}
			if got, err := ReadTransaction(home, tx.Recipe); err != nil || !reflect.DeepEqual(got, tx) {
				t.Fatalf("retry journal: %#v %v", got, err)
			}
		})
	}
}

func TestTransactionRootPresence(t *testing.T) {
	home := t.TempDir()
	tx := transactionFixture(t, home)
	tx.RootExisted = true
	if err := WriteTransaction(home, tx); err == nil {
		t.Fatal("existing unowned root accepted")
	}
	tx.Previous = tx.Candidate
	if err := WriteTransaction(home, tx); err != nil {
		t.Fatal(err)
	}
	got, err := ReadTransaction(home, tx.Recipe)
	if err != nil || !got.RootExisted {
		t.Fatalf("root presence lost: %+v %v", got, err)
	}
	tx.RootExisted = false
	if err := WriteTransaction(home, tx); err != nil {
		t.Fatal(err)
	}
	got, err = ReadTransaction(home, tx.Recipe)
	if err != nil || got.RootExisted {
		t.Fatalf("receipt incorrectly implies root presence: %+v %v", got, err)
	}
}
