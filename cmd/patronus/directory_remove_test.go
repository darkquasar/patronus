package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/packagedelivery"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/state"
)

func TestDirectoryRemoveReceiptOnly(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "kit", "1.0.0", "payload")
	if _, _, err := runInstall(t, "kit", "--deploy"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(f.home, ".patronus", "state.json")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := execRemove(t, "kit", "--deploy"); err != nil {
		t.Fatal(err)
	}
	r, err := packagestate.Load(f.home, "kit")
	if err != nil || r != nil {
		t.Fatalf("receipt: %+v %v", r, err)
	}
	tx, err := packagestate.ReadTransaction(f.home, "kit")
	if err != nil || tx != nil {
		t.Fatalf("transaction: %+v %v", tx, err)
	}
	s, err := state.Load(filepath.Join(f.home, ".patronus", "state.json"))
	if err != nil || len(s.Items) != 0 {
		t.Fatalf("state: %+v %v", s, err)
	}
	runner := runnerForCommands.(*fakeRunner)
	if len(runner.ran) != 0 {
		t.Fatalf("runtime calls: %+v", runner.ran)
	}
}

func TestDirectoryRemoveDryRunAndRetainedRetry(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "kit", "1.0.0", "payload")
	f.install(t, "kit")
	path := f.readme("kit")
	if err := os.WriteFile(path, []byte("edited"), 0644); err != nil {
		t.Fatal(err)
	}
	unknown := filepath.Join(filepath.Dir(path), "mine")
	if err := os.WriteFile(unknown, []byte("unknown"), 0644); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(f.home, ".patronus", "package-state", "kit.json")
	before := mustRead(t, receiptPath)
	out, _, err := execRemove(t, "kit")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "retain [README.md]") || !strings.Contains(out, "mine") {
		t.Fatal(out)
	}
	if !bytes.Equal(before, mustRead(t, receiptPath)) {
		t.Fatal("dry-run wrote receipt")
	}
	if _, _, err := execRemove(t, "kit", "--deploy"); err != nil {
		t.Fatal(err)
	}
	r, err := packagestate.Load(f.home, "kit")
	if err != nil || r == nil || len(r.Files) != 1 {
		t.Fatalf("reduced receipt %+v %v", r, err)
	}
	if _, _, err := execRemove(t, "kit", "--deploy", "--force"); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, unknown)) != "unknown" {
		t.Fatal("unknown changed")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("owned file retained", err)
	}
	if len(runnerForCommands.(*fakeRunner).ran) != 0 {
		t.Fatal("runtime call")
	}
}

func TestDirectoryRemoveDanglingReference(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "kit", "1.0.0", "payload")
	f.install(t, "kit")
	if err := packagestate.DeleteReceipt(f.home, "kit"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := execRemove(t, "kit", "--deploy", "--force"); err == nil {
		t.Fatal("dangling reference authorized removal")
	}
	if string(mustRead(t, f.readme("kit"))) != "payload" {
		t.Fatal("payload changed")
	}
}

func TestDirectoryRemovalReferenceFailureRetainsJournal(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "kit", "1.0.0", "payload")
	f.install(t, "kit")
	old := directoryServiceForDeploy
	t.Cleanup(func() { directoryServiceForDeploy = old })
	statePath := filepath.Join(f.home, ".patronus", "state.json")
	saved := mustRead(t, statePath)
	directoryServiceForDeploy = func(home string) *packagedelivery.Service {
		service := old(home)
		service.Fault = func(point string) error {
			if point == "after-reduced-receipt-write" {
				if err := os.WriteFile(statePath, []byte("broken json"), 0644); err != nil {
					return err
				}
			}
			return nil
		}
		return service
	}
	if _, _, err := execRemove(t, "kit", "--deploy"); err == nil {
		t.Fatal("reference failure swallowed")
	}
	tx, err := packagestate.ReadTransaction(f.home, "kit")
	if err != nil || tx == nil || tx.Phase != packagestate.Committed {
		t.Fatalf("journal: %+v %v", tx, err)
	}
	if r, err := packagestate.Load(f.home, "kit"); err != nil || r != nil {
		t.Fatalf("receipt resurrected: %+v %v", r, err)
	}
	if err := os.WriteFile(statePath, saved, 0644); err != nil {
		t.Fatal(err)
	}
	directoryServiceForDeploy = old
	if _, _, err := execRemove(t, "kit", "--deploy"); err != nil {
		t.Fatal(err)
	}
	tx, err = packagestate.ReadTransaction(f.home, "kit")
	if err != nil || tx != nil {
		t.Fatalf("acknowledgement failed: %+v %v", tx, err)
	}
}

func TestDirectoryInstallRecoversInterruptedRemoval(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "kit", "1.0.0", "payload")
	f.install(t, "kit")
	old := directoryServiceForDeploy
	t.Cleanup(func() { directoryServiceForDeploy = old })
	directoryServiceForDeploy = func(home string) *packagedelivery.Service {
		service := old(home)
		service.Fault = func(point string) error {
			if point == "after-unlink" {
				return errors.New("interrupted removal")
			}
			return nil
		}
		return service
	}
	if _, _, err := execRemove(t, "kit", "--deploy"); err == nil {
		t.Fatal("fault missed")
	}
	directoryServiceForDeploy = old
	f.install(t, "kit")
	if string(mustRead(t, f.readme("kit"))) != "payload" {
		t.Fatal("install did not finish")
	}
	tx, err := packagestate.ReadTransaction(f.home, "kit")
	if err != nil || tx != nil {
		t.Fatalf("journal: %+v %v", tx, err)
	}
	if len(runnerForCommands.(*fakeRunner).ran) != 0 {
		t.Fatal("runtime call")
	}
}

func TestDirectoryRemovalCrashHelper(t *testing.T) {
	if os.Getenv("PATRONUS_REMOVE_CLI_CHILD") != "1" {
		return
	}
	home := os.Getenv("PATRONUS_REMOVE_CLI_HOME")
	release, err := packagestate.Acquire(home)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := release(); err != nil {
			t.Error(err)
		}
	}()
	service := packagedelivery.Service{Home: home, Fault: func(point string) error {
		if point == "after-unlink" {
			os.Exit(97)
		}
		return nil
	}}
	_, err = service.Remove(context.Background(), "kit", false)
	t.Fatalf("hard-exit hook missed: %v", err)
}

func TestDirectoryInstallAfterRemovalProcessCrash(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "kit", "1.0.0", "payload")
	f.install(t, "kit")
	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestDirectoryRemovalCrashHelper$")
	cmd.Env = append(os.Environ(), "PATRONUS_REMOVE_CLI_CHILD=1", "PATRONUS_REMOVE_CLI_HOME="+f.home)
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 97 {
		t.Fatalf("expected exit 97: %v %s", err, output)
	}
	f.install(t, "kit")
	if string(mustRead(t, f.readme("kit"))) != "payload" {
		t.Fatal("reinstall failed")
	}
	tx, err := packagestate.ReadTransaction(f.home, "kit")
	if err != nil || tx != nil {
		t.Fatalf("journal: %+v %v", tx, err)
	}
	if len(runnerForCommands.(*fakeRunner).ran) != 0 {
		t.Fatal("runtime call")
	}
}

func TestDirectoryMixedRemovalDoesNotRestoreReference(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "kit", "1.0.0", "payload")
	f.install(t, "kit")
	path := filepath.Join(f.home, "legacy.txt")
	if err := os.WriteFile(path, []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(f.home, ".patronus", "state.json")
	st, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	st.Items = append(st.Items, state.Item{Artifact: "legacy", Tool: "claude", Scope: "global", Type: "agent", Files: []state.FileState{{Path: path, Action: "CREATE", Checksum: shaState([]byte("legacy"))}}})
	if err := state.Save(statePath, st); err != nil {
		t.Fatal(err)
	}
	if _, _, err := execRemove(t, "kit", "legacy", "--deploy"); err != nil {
		t.Fatal(err)
	}
	st, err = state.Load(statePath)
	if err != nil || len(st.Items) != 0 {
		t.Fatalf("removed reference restored: %+v %v", st, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("legacy survived: %v", err)
	}
	if len(runnerForCommands.(*fakeRunner).ran) != 0 {
		t.Fatal("runtime call")
	}
}
