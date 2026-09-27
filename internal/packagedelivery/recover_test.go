package packagedelivery

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/darkquasar/patronus/internal/packagestate"
)

func faultPoints() []string {
	return []string{"after-prepared", "after-intent-move-old", "after-backup-rename", "after-intent-promote-new", "after-stage-rename", "after-package-placed", "before-receipt-save", "after-receipt-save", "before-commit-write", "after-committed", "during-backup-cleanup"}
}
func isCommittedPoint(point string) bool {
	return point == "after-committed" || point == "during-backup-cleanup"
}
func assertVersion(t *testing.T, home, want string) {
	t.Helper()
	receipt, err := packagestate.Load(home, "kit")
	must(t, err)
	root := filepath.Join(home, ".patronus", "packages", "kit")
	if want == "" {
		if receipt != nil {
			t.Fatal("receipt survives initial rollback")
		}
		if _, err := os.Lstat(root); !os.IsNotExist(err) {
			t.Fatalf("root survives initial rollback: %v", err)
		}
		return
	}
	if receipt == nil || receipt.RecipeVersion != want {
		t.Fatalf("receipt %+v, want %s", receipt, want)
	}
	data, err := os.ReadFile(filepath.Join(root, "dir/tool"))
	must(t, err)
	if string(data) != want {
		t.Fatalf("tree %q want %s", data, want)
	}
}
func TestReturnedFaultRollsBack(t *testing.T) {
	for _, initial := range []bool{false, true} {
		for _, point := range faultPoints() {
			t.Run(point+map[bool]string{false: "/update", true: "/initial"}[initial], func(t *testing.T) {
				home := t.TempDir()
				locked(t, home)
				old, oldData := fixture(t, home, "1.0.0")
				s := Service{Home: home, Fetcher: oldData}
				if !initial {
					_, err := s.Replace(context.Background(), old, false)
					must(t, err)
				}
				req, data := fixture(t, home, "2.0.0")
				s.Fetcher = data
				injected := errors.New("injected ordinary failure")
				s.Fault = func(p string) error {
					if p == point {
						return injected
					}
					return nil
				}
				result, err := s.Replace(context.Background(), req, false)
				if !errors.Is(err, injected) {
					t.Fatalf("missing fault: %v", err)
				}
				want := "1.0.0"
				if initial {
					want = ""
				}
				if isCommittedPoint(point) {
					want = "2.0.0"
					if !result.Mutated || result.Receipt == nil {
						t.Fatal("lost committed result")
					}
				}
				assertVersion(t, home, want)
				tx, err := packagestate.ReadTransaction(home, "kit")
				must(t, err)
				if isCommittedPoint(point) {
					if tx == nil {
						t.Fatal("cleanup fault lost journal")
					}
					fresh := Service{Home: home}
					must(t, fresh.Recover(context.Background(), "kit"))
				} else if tx != nil {
					t.Fatalf("ordinary rollback left journal: %+v", tx)
				}
			})
		}
	}
}
func TestCrashHelper(t *testing.T) {
	if os.Getenv("PATRONUS_CRASH_CHILD") != "1" {
		return
	}
	home := os.Getenv("PATRONUS_CRASH_HOME")
	locked(t, home)
	req, data := fixture(t, home, "2.0.0")
	s := Service{Home: home, Fetcher: data}
	point := os.Getenv("PATRONUS_CRASH_POINT")
	s.Fault = func(p string) error {
		if p == point {
			os.Exit(97)
		}
		return nil
	}
	if os.Getenv("PATRONUS_REMOVE") == "1" {
		_, err := s.Remove(context.Background(), "kit", true)
		t.Fatalf("removal crash hook missed: %v", err)
	}
	if os.Getenv("PATRONUS_METADATA") == "1" {
		old, oldData := fixture(t, home, "1.0.0")
		req = old
		req.RecipeVersion = "2.0.0"
		s.Fetcher = oldData
	}

	if storagePoint := os.Getenv("PATRONUS_STORAGE_POINT"); storagePoint != "" {
		visibleWrite := os.Getenv("PATRONUS_STORAGE_VISIBLE") == "1"
		injected := false
		fault := func(path string) error {
			injected = true
			return &packagestate.DurabilityError{Path: path, Stage: "injected-sync", MayBeVisible: true, Err: errors.New("storage interruption")}
		}
		s.persistence = &persistence{
			write: func(home string, tx *packagestate.Transaction) error {
				match := storagePoint == string(tx.Phase)+"/"+tx.Intent
				if match && !injected {
					if visibleWrite {
						must(t, packagestate.WriteTransaction(home, tx))
					}
					return fault("transaction.json")
				}
				return packagestate.WriteTransaction(home, tx)
			},
			save: func(home string, r *packagestate.Receipt) error {
				if storagePoint == "receipt" && !injected {
					if visibleWrite {
						must(t, packagestate.Save(home, r))
					}
					return fault("kit.json")
				}
				return packagestate.Save(home, r)
			},
		}
		_, err := s.Replace(context.Background(), req, true)
		if !injected || !errors.Is(err, ErrRecoveryRequired) {
			t.Fatalf("storage hook missed: %v", err)
		}
		// A second interruption after the ambiguous error must not destroy evidence.
		os.Exit(97)
	}
	_, err := s.Replace(context.Background(), req, true)
	t.Fatalf("crash hook missed: %v", err)
}
func runCrash(t *testing.T, home, point string, extra ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestCrashHelper$")
	cmd.Env = append(os.Environ(), "PATRONUS_CRASH_CHILD=1", "PATRONUS_CRASH_HOME="+home, "PATRONUS_CRASH_POINT="+point)
	cmd.Env = append(cmd.Env, extra...)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 97 {
		t.Fatalf("wanted hard exit97, got %v: %s", err, out)
	}
}
func setupCrashBaseline(t *testing.T, home string, forced bool) {
	t.Helper()
	release, err := packagestate.Acquire(home)
	must(t, err)
	req, data := fixture(t, home, "1.0.0")
	s := Service{Home: home, Fetcher: data}
	_, err = s.Replace(context.Background(), req, false)
	must(t, err)
	if forced {
		must(t, os.WriteFile(filepath.Join(req.Root, "dir/tool"), []byte("actual user preimage"), 0755))
		must(t, os.Chmod(filepath.Join(req.Root, "dir/tool"), 0600))
	}
	must(t, release())
}
func TestHardCrashRecovery(t *testing.T) {
	for _, baseline := range []string{"initial", "update", "forced"} {
		for _, point := range faultPoints() {
			t.Run(baseline+"/"+point, func(t *testing.T) {
				home := t.TempDir()
				if baseline != "initial" {
					setupCrashBaseline(t, home, baseline == "forced")
				}
				runCrash(t, home, point)
				// Assert crash evidence before constructing a fresh recovery service.
				tx, err := packagestate.ReadTransaction(home, "kit")
				must(t, err)
				if tx == nil {
					t.Fatal("crash did not leave pending transaction")
				}
				expected := packagestate.Prepared
				if point == "after-package-placed" || point == "before-receipt-save" || point == "after-receipt-save" || point == "before-commit-write" {
					expected = packagestate.PackagePlaced
				}
				if isCommittedPoint(point) {
					expected = packagestate.Committed
				}
				if tx.Phase != expected {
					t.Fatalf("phase %s want %s", tx.Phase, expected)
				}
				stage, err := exists(tx.Stage)
				must(t, err)
				root, err := exists(tx.Root)
				must(t, err)
				backup, err := exists(tx.Backup)
				must(t, err)
				placed := point == "after-stage-rename" || expected != packagestate.Prepared
				if stage == placed {
					t.Fatalf("stage=%v placed=%v", stage, placed)
				}
				moved := point != "after-prepared" && point != "after-intent-move-old"
				if backup != (baseline != "initial" && moved) {
					t.Fatalf("backup %v baseline %s moved %v", backup, baseline, moved)
				}
				if root != (placed || baseline != "initial" && !moved) {
					t.Fatalf("root=%v placed=%v moved=%v", root, placed, moved)
				}
				locked(t, home)
				fresh := Service{Home: home}
				must(t, fresh.Recover(context.Background(), "kit"))
				if isCommittedPoint(point) {
					assertVersion(t, home, "2.0.0")
				} else if baseline == "initial" {
					assertVersion(t, home, "")
				} else if baseline == "forced" {
					receipt, err := packagestate.Load(home, "kit")
					must(t, err)
					if receipt.RecipeVersion != "1.0.0" {
						t.Fatal(receipt)
					}
					p := filepath.Join(tx.Root, "dir/tool")
					data, err := os.ReadFile(p)
					must(t, err)
					info, err := os.Stat(p)
					must(t, err)
					if string(data) != "actual user preimage" || info.Mode().Perm() != 0600 {
						t.Fatalf("lost forced preimage %q %v", data, info.Mode())
					}
				} else {
					assertVersion(t, home, "1.0.0")
				}
				tx, err = packagestate.ReadTransaction(home, "kit")
				must(t, err)
				if tx != nil {
					t.Fatal("recovery left journal")
				}
			})
		}
	}
}
func TestRecoveryConflictRetry(t *testing.T) {
	for _, point := range []string{"after-stage-rename", "after-committed"} {
		for _, kind := range []string{"unknown", "edited"} {
			t.Run(point+"/"+kind, func(t *testing.T) {
				home := t.TempDir()
				setupCrashBaseline(t, home, false)
				runCrash(t, home, point)
				locked(t, home)
				tx, err := packagestate.ReadTransaction(home, "kit")
				must(t, err)
				location := tx.Root
				if isCommittedPoint(point) {
					location = tx.Backup
				}
				path := filepath.Join(location, "user-file")
				if kind == "edited" {
					path = filepath.Join(location, "dir/tool")
				}
				must(t, os.WriteFile(path, []byte("preserve me"), 0755))
				s := Service{Home: home}
				var first []string
				for attempt := 0; attempt < 2; attempt++ {
					err = s.Recover(context.Background(), "kit")
					var conflict *ConflictError
					if !errors.Is(err, ErrRecoveryRequired) || !errors.As(err, &conflict) {
						t.Fatalf("not typed recovery: %v", err)
					}
					if attempt == 0 {
						first = conflict.Paths
					} else if !reflect.DeepEqual(first, conflict.Paths) {
						t.Fatalf("conflict changed %v %v", first, conflict.Paths)
					}
					data, err := os.ReadFile(path)
					must(t, err)
					if string(data) != "preserve me" {
						t.Fatal("conflict lost")
					}
					pending, err := packagestate.ReadTransaction(home, "kit")
					must(t, err)
					if pending.Phase != packagestate.RecoveryRequired || pending.ResumePhase != tx.Phase || pending.Intent != expectedRecoveryIntent(tx) {
						t.Fatalf("lost resume state: %+v", pending)
					}
				}
				must(t, os.Rename(path, filepath.Join(home, "relocated")))
				must(t, s.Recover(context.Background(), "kit"))
				if isCommittedPoint(point) {
					assertVersion(t, home, "2.0.0")
				} else {
					assertVersion(t, home, "1.0.0")
				}
			})
		}
	}
}
func TestMetadataCrashRecovery(t *testing.T) {
	for _, point := range []string{"after-prepared", "before-receipt-save", "after-receipt-save", "before-commit-write", "after-committed"} {
		t.Run(point, func(t *testing.T) {
			home := t.TempDir()
			setupCrashBaseline(t, home, false)
			root := filepath.Join(home, ".patronus", "packages", "kit")
			before, err := os.Stat(root)
			must(t, err)
			runCrash(t, home, point, "PATRONUS_METADATA=1")
			tx, err := packagestate.ReadTransaction(home, "kit")
			must(t, err)
			if tx == nil || tx.Operation != "metadata" || tx.Stage != "" || tx.Backup != "" {
				t.Fatalf("metadata renamed: %+v", tx)
			}
			after, err := os.Stat(root)
			must(t, err)
			if !os.SameFile(before, after) {
				t.Fatal("metadata moved root")
			}
			locked(t, home)
			s := Service{Home: home}
			must(t, s.Recover(context.Background(), "kit"))
			r, err := packagestate.Load(home, "kit")
			must(t, err)
			want := "1.0.0"
			if isCommittedPoint(point) {
				want = "2.0.0"
			}
			if r.RecipeVersion != want {
				t.Fatal(r)
			}
		})
	}
}
func TestCanceledReplacement(t *testing.T) {
	for _, point := range []string{"before", "after-prepared", "after-backup-rename", "after-stage-rename", "after-receipt-save"} {
		t.Run(point, func(t *testing.T) {
			s, old := installed(t)
			req, data := fixture(t, s.Home, "2.0.0")
			s.Fetcher = data
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if point == "before" {
				cancel()
			}
			s.Fault = func(p string) error {
				if p == point {
					cancel()
				}
				return nil
			}
			_, err := s.Replace(ctx, req, false)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation lost: %v", err)
			}
			assertVersion(t, s.Home, old.RecipeVersion)
			tx, err := packagestate.ReadTransaction(s.Home, "kit")
			must(t, err)
			if tx != nil {
				t.Fatal("cancel abandoned journal")
			}
		})
	}
}
func TestRecoveryDoesNotDispatchRemovalAsReplacement(t *testing.T) {
	s, req := installed(t)
	r, err := packagestate.Load(s.Home, "kit")
	must(t, err)
	tx := &packagestate.Transaction{SchemaVersion: 1, Recipe: "kit", Root: req.Root, Phase: packagestate.Prepared, Operation: "remove", Previous: r}
	must(t, packagestate.WriteTransaction(s.Home, tx))
	err = s.Recover(context.Background(), "kit")
	if err != nil {
		t.Fatal(err)
	}
	assertVersion(t, s.Home, "1.0.0")
}

func TestMutationBoundaryRechecks(t *testing.T) {
	for _, point := range []string{"after-intent-move-old", "after-intent-promote-new"} {
		t.Run(point, func(t *testing.T) {
			s, old := installed(t)
			req, data := fixture(t, s.Home, "2.0.0")
			s.Fetcher = data
			introduced := filepath.Join(old.Root, "user-added")
			s.Fault = func(p string) error {
				if p == point {
					if p == "after-intent-promote-new" {
						must(t, os.Mkdir(old.Root, 0755))
					}
					must(t, os.WriteFile(introduced, []byte("intervening bytes"), 0644))
				}
				return nil
			}
			_, err := s.Replace(context.Background(), req, true)
			if err == nil {
				t.Fatal("intervening edit was not a conflict")
			}
			bytes, readErr := os.ReadFile(introduced)
			must(t, readErr)
			if string(bytes) != "intervening bytes" {
				t.Fatal("intervening content relocated or lost")
			}
		})
	}
}

func TestEmptyAndAbsentOwnedRoots(t *testing.T) {
	for _, present := range []bool{false, true} {
		for _, point := range []string{"after-prepared", "after-backup-rename", "after-stage-rename"} {
			t.Run(point+map[bool]string{false: "/absent", true: "/empty"}[present], func(t *testing.T) {
				home := t.TempDir()
				setupCrashBaseline(t, home, false)
				root := filepath.Join(home, ".patronus", "packages", "kit")
				must(t, os.Remove(filepath.Join(root, "dir/tool")))
				must(t, os.Remove(filepath.Join(root, "dir")))
				must(t, os.Remove(filepath.Join(root, "package.json")))
				if !present {
					must(t, os.Remove(root))
				}
				runCrash(t, home, point)
				tx, err := packagestate.ReadTransaction(home, "kit")
				must(t, err)
				if tx.RootExisted != present {
					t.Fatal("incorrect root presence")
				}
				locked(t, home)
				s := Service{Home: home}
				must(t, s.Recover(context.Background(), "kit"))
				got, err := exists(root)
				must(t, err)
				if got != present {
					t.Fatalf("root exists %v want %v", got, present)
				}
				r, err := packagestate.Load(home, "kit")
				must(t, err)
				if r == nil || r.RecipeVersion != "1.0.0" {
					t.Fatal("old receipt lost")
				}
			})
		}
	}
}
func TestMissingBackupBlocksRecovery(t *testing.T) {
	home := t.TempDir()
	setupCrashBaseline(t, home, false)
	runCrash(t, home, "after-stage-rename")
	tx, err := packagestate.ReadTransaction(home, "kit")
	must(t, err)
	must(t, os.Rename(tx.Backup, filepath.Join(home, "misplaced-backup")))
	locked(t, home)
	s := Service{Home: home}
	err = s.Recover(context.Background(), "kit")
	if !errors.Is(err, ErrRecoveryRequired) {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(tx.Root, "dir/tool"))
	must(t, err)
	if string(data) != "2.0.0" {
		t.Fatal("new tree destroyed without backup")
	}
	must(t, os.Rename(filepath.Join(home, "misplaced-backup"), tx.Backup))
	must(t, s.Recover(context.Background(), "kit"))
	assertVersion(t, home, "1.0.0")
}
func TestUnrelatedReceiptPreserved(t *testing.T) {
	home := t.TempDir()
	setupCrashBaseline(t, home, false)
	runCrash(t, home, "after-stage-rename")
	locked(t, home)
	r, err := packagestate.Load(home, "kit")
	must(t, err)
	r.Recipe = "other"
	r.Root = filepath.Join(filepath.Dir(r.Root), "other")
	must(t, packagestate.Save(home, r))
	s := Service{Home: home}
	must(t, s.Recover(context.Background(), "kit"))
	after, err := packagestate.Load(home, "other")
	must(t, err)
	if !reflect.DeepEqual(after, r) {
		t.Fatal("unrelated receipt changed")
	}
}

func TestAmbiguousStorageCrashRecovery(t *testing.T) {
	points := []string{"prepared/move-old", "prepared/promote-new", "package-placed/save-receipt", "receipt", "package-placed/commit", "committed/commit"}
	for _, point := range points {
		for _, visibleWrite := range []bool{false, true} {
			t.Run(point+map[bool]string{false: "/old-visible", true: "/new-visible"}[visibleWrite], func(t *testing.T) {
				home := t.TempDir()
				setupCrashBaseline(t, home, false)
				value := "0"
				if visibleWrite {
					value = "1"
				}
				runCrash(t, home, "no-fault-hook", "PATRONUS_STORAGE_POINT="+point, "PATRONUS_STORAGE_VISIBLE="+value)
				tx, err := packagestate.ReadTransaction(home, "kit")
				must(t, err)
				if tx == nil {
					t.Fatal("ambiguous error erased journal")
				}
				committed := point == "committed/commit" && visibleWrite
				if (tx.Phase == packagestate.Committed) != committed {
					t.Fatalf("visible phase %+v", tx)
				}
				// No rollback was attempted: before promotion stage remains; after it root is new.
				placed := point == "package-placed/save-receipt" || point == "receipt" || point == "package-placed/commit" || point == "committed/commit"
				if placed {
					data, err := os.ReadFile(filepath.Join(tx.Root, "dir/tool"))
					must(t, err)
					if string(data) != "2.0.0" {
						t.Fatal("ambiguous write triggered rollback")
					}
				}
				locked(t, home)
				fresh := Service{Home: home}
				attempts := 0
				fresh.persistence = &persistence{stabilize: func(path string) error { attempts++; return errors.New("still unsynced: " + path) }}
				err = fresh.Recover(context.Background(), "kit")
				if !errors.Is(err, ErrRecoveryRequired) || attempts != 1 {
					t.Fatalf("sync failure did not stop recovery: %v", err)
				}
				retained, err := packagestate.ReadTransaction(home, "kit")
				must(t, err)
				if !reflect.DeepEqual(retained, tx) {
					t.Fatal("sync failure rewrote journal")
				}
				fresh.persistence = nil
				must(t, fresh.Recover(context.Background(), "kit"))
				want := "1.0.0"
				if committed {
					want = "2.0.0"
				}
				assertVersion(t, home, want)
			})
		}
	}
}
func TestBeforeVisibleStorageErrorsRollback(t *testing.T) {
	for _, point := range []string{"prepared/move-old", "prepared/promote-new", "package-placed/save-receipt", "receipt", "package-placed/commit", "committed/commit"} {
		t.Run(point, func(t *testing.T) {
			s, _ := installed(t)
			req, data := fixture(t, s.Home, "2.0.0")
			s.Fetcher = data
			fired := false
			injected := errors.New("before storage write")
			fault := func() error {
				fired = true
				return &packagestate.DurabilityError{Stage: "before-rename", MayBeVisible: false, Err: injected}
			}
			s.persistence = &persistence{write: func(home string, tx *packagestate.Transaction) error {
				if !fired && point == string(tx.Phase)+"/"+tx.Intent {
					return fault()
				}
				return packagestate.WriteTransaction(home, tx)
			}, save: func(home string, r *packagestate.Receipt) error {
				if !fired && point == "receipt" {
					return fault()
				}
				return packagestate.Save(home, r)
			}}
			_, err := s.Replace(context.Background(), req, false)
			if !errors.Is(err, injected) {
				t.Fatalf("fault not returned: %v", err)
			}
			assertVersion(t, s.Home, "1.0.0")
			tx, err := packagestate.ReadTransaction(s.Home, "kit")
			must(t, err)
			if tx != nil {
				t.Fatal("before-visible failure abandoned transaction")
			}
		})
	}
}

func expectedRecoveryIntent(tx *packagestate.Transaction) string {
	if tx.Phase == packagestate.Committed {
		return "cleanup"
	}
	return tx.Intent
}

func TestReturnedFaultRestoresForcedPreimage(t *testing.T) {
	for _, point := range faultPoints() {
		t.Run(point, func(t *testing.T) {
			s, old := installed(t)
			path := filepath.Join(old.Root, "dir/tool")
			must(t, os.WriteFile(path, []byte("edited baseline"), 0755))
			must(t, os.Chmod(path, 0600))
			req, data := fixture(t, s.Home, "2.0.0")
			s.Fetcher = data
			injected := errors.New("ordinary forced failure")
			s.Fault = func(p string) error {
				if p == point {
					return injected
				}
				return nil
			}
			_, err := s.Replace(context.Background(), req, true)
			if !errors.Is(err, injected) {
				t.Fatal(err)
			}
			if isCommittedPoint(point) {
				assertVersion(t, s.Home, "2.0.0")
				return
			}
			got, err := os.ReadFile(path)
			must(t, err)
			info, err := os.Stat(path)
			must(t, err)
			if string(got) != "edited baseline" || info.Mode().Perm() != 0600 {
				t.Fatalf("preimage lost: %q %v", got, info.Mode())
			}
			r, err := packagestate.Load(s.Home, "kit")
			must(t, err)
			if r.RecipeVersion != "1.0.0" {
				t.Fatal(r)
			}
		})
	}
}
func TestMetadataReturnedFaults(t *testing.T) {
	for _, point := range []string{"after-prepared", "before-receipt-save", "after-receipt-save", "before-commit-write", "after-committed"} {
		t.Run(point, func(t *testing.T) {
			s, req := installed(t)
			req.RecipeVersion = "2.0.0"
			s.Fetcher = failingFetcher{}
			injected := errors.New("metadata failure")
			s.Fault = func(p string) error {
				if p == point {
					return injected
				}
				return nil
			}
			_, err := s.Replace(context.Background(), req, false)
			if !errors.Is(err, injected) {
				t.Fatal(err)
			}
			r, err := packagestate.Load(s.Home, "kit")
			must(t, err)
			want := "1.0.0"
			if isCommittedPoint(point) {
				want = "2.0.0"
			}
			if r.RecipeVersion != want {
				t.Fatal(r)
			}
		})
	}
}
func TestFirstInstallConflictRetry(t *testing.T) {
	home := t.TempDir()
	runCrash(t, home, "after-stage-rename")
	locked(t, home)
	tx, err := packagestate.ReadTransaction(home, "kit")
	must(t, err)
	path := filepath.Join(tx.Root, "unexpected")
	must(t, os.WriteFile(path, []byte("keep"), 0644))
	s := Service{Home: home}
	for i := 0; i < 2; i++ {
		if err := s.Recover(context.Background(), "kit"); !errors.Is(err, ErrRecoveryRequired) {
			t.Fatal(err)
		}
	}
	must(t, os.Rename(path, filepath.Join(home, "saved")))
	must(t, s.Recover(context.Background(), "kit"))
	assertVersion(t, home, "")
}
func TestInspectPendingIsReadOnly(t *testing.T) {
	home := t.TempDir()
	setupCrashBaseline(t, home, false)
	runCrash(t, home, "after-stage-rename")
	req, _ := fixture(t, home, "2.0.0")
	s := Service{Home: home}
	before, err := packagestate.ReadTransaction(home, "kit")
	must(t, err)
	in, err := s.Inspect(req)
	must(t, err)
	if in.Pending == nil {
		t.Fatal("pending not reported")
	}
	after, err := packagestate.ReadTransaction(home, "kit")
	must(t, err)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("read recovered transaction")
	}
	data, err := os.ReadFile(filepath.Join(req.Root, "dir/tool"))
	must(t, err)
	if string(data) != "2.0.0" {
		t.Fatal("read mutated tree")
	}
}

func TestCommittedCleanupPersistenceErrors(t *testing.T) {
	for _, operation := range []string{"cleanup-intent", "clear-journal"} {
		for _, visibleWrite := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: "/before", true: "/after"}[visibleWrite], func(t *testing.T) {
				s, _ := installed(t)
				req, data := fixture(t, s.Home, "2.0.0")
				s.Fetcher = data
				injected := errors.New("cleanup persistence error")
				fired := false
				fault := func() error {
					fired = true
					return &packagestate.DurabilityError{Stage: "cleanup", MayBeVisible: visibleWrite, Err: injected}
				}
				s.persistence = &persistence{write: func(home string, tx *packagestate.Transaction) error {
					if operation == "cleanup-intent" && tx.Intent == "cleanup" && !fired {
						if visibleWrite {
							must(t, packagestate.WriteTransaction(home, tx))
						}
						return fault()
					}
					return packagestate.WriteTransaction(home, tx)
				}, clear: func(home, recipe string) error {
					if operation == "clear-journal" && !fired {
						if visibleWrite {
							must(t, packagestate.ClearTransaction(home, recipe))
						}
						return fault()
					}
					return packagestate.ClearTransaction(home, recipe)
				}}
				result, err := s.Replace(context.Background(), req, false)
				if !errors.Is(err, injected) || !result.Mutated || result.Receipt == nil {
					t.Fatalf("lost committed outcome: %+v %v", result, err)
				}
				assertVersion(t, s.Home, "2.0.0")
				fresh := Service{Home: s.Home}
				must(t, fresh.Recover(context.Background(), "kit"))
				assertVersion(t, s.Home, "2.0.0")
			})
		}
	}
}
func TestRollbackReceiptPersistenceRetry(t *testing.T) {
	for _, visibleWrite := range []bool{false, true} {
		t.Run(map[bool]string{false: "before", true: "after"}[visibleWrite], func(t *testing.T) {
			home := t.TempDir()
			setupCrashBaseline(t, home, false)
			runCrash(t, home, "after-receipt-save")
			locked(t, home)
			s := Service{Home: home}
			s.persistence = &persistence{save: func(home string, r *packagestate.Receipt) error {
				if visibleWrite {
					must(t, packagestate.Save(home, r))
				}
				return &packagestate.DurabilityError{Stage: "rollback-save", MayBeVisible: true, Err: errors.New("rollback receipt ambiguity")}
			}}
			err := s.Recover(context.Background(), "kit")
			if !errors.Is(err, ErrRecoveryRequired) {
				t.Fatal(err)
			}
			tx, err := packagestate.ReadTransaction(home, "kit")
			must(t, err)
			if tx == nil {
				t.Fatal("rollback ambiguity lost journal")
			}
			data, err := os.ReadFile(filepath.Join(tx.Root, "dir/tool"))
			must(t, err)
			if string(data) != "1.0.0" {
				t.Fatal("preimage not restored")
			}
			fresh := Service{Home: home}
			must(t, fresh.Recover(context.Background(), "kit"))
			assertVersion(t, home, "1.0.0")
		})
	}
}
func TestInterveningReceiptPreservedBeforeTreeMutation(t *testing.T) {
	home := t.TempDir()
	setupCrashBaseline(t, home, false)
	runCrash(t, home, "after-stage-rename")
	locked(t, home)
	r, err := packagestate.Load(home, "kit")
	must(t, err)
	r.RecipeVersion = "9.0.0"
	must(t, packagestate.Save(home, r))
	s := Service{Home: home}
	err = s.Recover(context.Background(), "kit")
	if !errors.Is(err, ErrRecoveryRequired) {
		t.Fatal(err)
	}
	after, err := packagestate.Load(home, "kit")
	must(t, err)
	if !reflect.DeepEqual(after, r) {
		t.Fatal("intervening receipt overwritten")
	}
	data, err := os.ReadFile(filepath.Join(r.Root, "dir/tool"))
	must(t, err)
	if string(data) != "2.0.0" {
		t.Fatal("tree mutated before receipt conflict")
	}
}

func TestMatchingCollisionRootIsPreserved(t *testing.T) {
	s, req := installed(t)
	next, data := fixture(t, s.Home, "2.0.0")
	s.Fetcher = data
	s.Fault = func(point string) error {
		if point != "after-intent-promote-new" {
			return nil
		}
		tx, err := packagestate.ReadTransaction(s.Home, "kit")
		must(t, err)
		must(t, os.Mkdir(req.Root, 0755))
		for _, entry := range tx.Candidate.Files {
			source := filepath.Join(tx.Stage, entry.Path)
			target := filepath.Join(req.Root, entry.Path)
			must(t, os.MkdirAll(filepath.Dir(target), 0755))
			bytes, err := os.ReadFile(source)
			must(t, err)
			must(t, os.WriteFile(target, bytes, os.FileMode(entry.Mode)))
		}
		return nil
	}
	_, err := s.Replace(context.Background(), next, false)
	if !errors.Is(err, ErrRecoveryRequired) {
		t.Fatalf("matching collision not retained: %v", err)
	}
	bytes, err := os.ReadFile(filepath.Join(req.Root, "dir/tool"))
	must(t, err)
	if string(bytes) != "2.0.0" {
		t.Fatal("matching unknown root overwritten by rollback")
	}
}
