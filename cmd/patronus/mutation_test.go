package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func TestMutationCommandsFailFastWhileHomeBusy(t *testing.T) {
	f := newDirectoryFixture(t)
	release, err := packagestate.Acquire(f.home)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	tests := []struct {
		name string
		run  func() error
	}{
		{"install", func() error {
			_, _, err := runInstall(t, "invented-item", "--target", "pi", "--global", "--deploy")
			return err
		}},
		{"update", func() error { _, _, err := runUpdate(t, "invented-item", "--deploy"); return err }},
		{"remove", func() error { _, _, err := execRemove(t, "invented-item", "--deploy"); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			err := tt.run()
			if !errors.Is(err, packagestate.ErrBusy) {
				t.Fatalf("want busy before planning, got %v", err)
			}
			if time.Since(start) > time.Second {
				t.Fatal("busy writer queued instead of failing immediately")
			}
		})
	}
}

func dp05AssertCoordinationLock(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() != 0 {
		t.Fatalf("unexpected coordination inode %s: %v size=%d", path, info.Mode(), info.Size())
	}
}

// The holder is a separate process running a real command, not a test mutex.
// The child pauses at preview output: its state/config reads and persistence
// must all lie within the same outer advisory-lock lifetime.
func TestMutationCooperatingCommandsAcrossProcesses(t *testing.T) {
	f := dp02Setup(t)
	dp02Artifact(t, f.root, "fixture-prompt", "command", "Invented prompt\n")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMutationHolderProcess$")
	child.Env = append(os.Environ(), "PATRONUS_DP05_HOLDER=1")
	output, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	input, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	child.Stderr = &stderr
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = input.Close(); _ = child.Wait() }()
	line, err := bufio.NewReader(output).ReadString('\n')
	if err != nil || line != "holding\n" {
		t.Fatalf("child failed to hold: %q %v %s", line, err, stderr.String())
	}
	tests := []struct {
		name string
		run  func() error
	}{
		{"install", func() error {
			_, _, e := runInstall(t, "fixture-prompt", "--target", "pi", "--global", "--deploy")
			return e
		}},
		{"update", func() error { _, _, e := runUpdate(t, "fixture-prompt", "--deploy"); return e }},
		{"remove", func() error { _, _, e := execRemove(t, "fixture-prompt", "--deploy"); return e }},
		{"lock", func() error { _, _, e := runLock(t, "--profile", "invented-profile"); return e }},
		{"scan", func() error { _, _, e := runScan(t); return e }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			err := tt.run()
			if !errors.Is(err, packagestate.ErrBusy) {
				t.Fatalf("not immediate busy: %v", err)
			}
			if time.Since(start) > time.Second {
				t.Fatal("queued busy writer")
			}
		})
	}
	path := filepath.Join(f.home, ".pi/agent/prompts/fixture-prompt.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("contender wrote while held: %v", err)
	}
	if err := input.Close(); err != nil {
		t.Fatal(err)
	}
	if err := child.Wait(); err != nil {
		t.Fatalf("holder: %v %s", err, stderr.String())
	}
	if string(mustRead(t, path)) != "Invented prompt\n" {
		t.Fatal("holder did not finish")
	}
	release, err := packagestate.Acquire(f.home)
	if err != nil {
		t.Fatalf("holder leaked lock: %v", err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
}

type dp05HoldingWriter struct{ held bool }

func (w *dp05HoldingWriter) Write(p []byte) (int, error) {
	if !w.held {
		w.held = true
		fmt.Fprintln(os.Stdout, "holding")
		var b [1]byte
		_, _ = os.Stdin.Read(b[:])
	}
	return len(p), nil
}

func TestMutationHolderProcess(t *testing.T) {
	if os.Getenv("PATRONUS_DP05_HOLDER") != "1" {
		return
	}
	cmd := newInstallCmd()
	cmd.SetOut(&dp05HoldingWriter{})
	cmd.SetErr(os.Stderr)
	cmd.SetArgs([]string{"fixture-prompt", "--target", "pi", "--global", "--deploy", "--yes"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestMutationReleaseJoinsOperationError(t *testing.T) {
	operation := errors.New("operation failed")
	release := errors.New("release failed")
	err := operation
	m := &mutation{release: func() error { return release }}
	m.close(&err)
	if !errors.Is(err, operation) || !errors.Is(err, release) {
		t.Fatalf("lost error: %v", err)
	}
}

func TestPiRaceConfirmedPreparedContextRequiresFreshInteractivePreview(t *testing.T) {
	f := dp02Setup(t)
	dp02Artifact(t, f.root, "context-fixture", "instruction", "Pi-specific contribution\n")
	path := filepath.Join(f.root, "CLAUDE.md")
	dp02File(t, path, "Operator-prepared combined instructions\n")
	m, err := beginMutation(f.home, f.root)
	if err != nil {
		t.Fatal(err)
	}
	defer m.release()
	cmd := newInstallCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	p, err := planInstall(cmd, installPlanRequest{Names: []string{"context-fixture"}, Tool: "pi", Scope: "local", Home: f.home, ProjectDir: f.root})
	if err != nil {
		t.Fatal(err)
	}
	reviews, err := piPreflightPlan(p.Changes, p.Resolver, f.home, f.root)
	if err != nil {
		t.Fatal(err)
	}
	consents, err := confirmPiContexts(reviews, true, false, false, strings.NewReader("prepared combined file\n"), io.Discard)
	if err != nil || len(consents) != 1 {
		t.Fatalf("confirmation: %v %#v", err, consents)
	}
	if err := m.checkPlan(p.Changes, consents); err != nil {
		t.Fatal(err)
	}
	dp02File(t, path, "External edit after confirmed consent\n")
	before := dp02Snapshot(t, f.home, f.root)
	err = m.checkPlan(p.Changes, consents)
	if err == nil || !strings.Contains(err.Error(), "new interactive preview required") {
		t.Fatalf("stale consent not invalidated: %v", err)
	}
	// This is also the exact callback run immediately before each real write.
	app := install.Applier{BeforeWrite: func(d diff.FileDiff) error { return m.checkFile(d, consents) }}
	result, err := app.Apply(p.Changes)
	if err == nil || len(result.Applied) != 0 {
		t.Fatalf("stale consent reached write: %+v %v", result, err)
	}
	if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
		t.Fatal("consent rejection changed bytes/state")
	}
}

// Output is synchronous, so this injects an editor after planning but before
// admission without a timing-sensitive goroutine or a production global hook.
type dp05WriterFunc func([]byte) (int, error)

func (f dp05WriterFunc) Write(p []byte) (int, error) { return f(p) }

func TestPiRaceCommandRejectsEditAfterPlan(t *testing.T) {
	f := dp02Setup(t)
	dp02Artifact(t, f.root, "fixture-prompt", "command", "Desired prompt\n")
	path := filepath.Join(f.home, ".pi/agent/prompts/fixture-prompt.md")
	cmd := newInstallCmd()
	edited := false
	cmd.SetOut(dp05WriterFunc(func(p []byte) (int, error) {
		if !edited {
			edited = true
			dp02File(t, path, "External edit\n")
		}
		return len(p), nil
	}))
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"fixture-prompt", "--target", "pi", "--global", "--deploy", "--yes"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "fresh preview") {
		t.Fatalf("stale command plan admitted: %v", err)
	}
	if string(mustRead(t, path)) != "External edit\n" {
		t.Fatal("overwrote external edit")
	}
	if _, err := os.Stat(filepath.Join(f.home, ".patronus/state.json")); !os.IsNotExist(err) {
		t.Fatalf("state advanced: %v", err)
	}
}

func TestMutationStateDriftBeforePersistencePreservesExternalBytes(t *testing.T) {
	f := dp02Setup(t)
	path := filepath.Join(f.home, ".patronus/state.json")
	original := &state.State{Version: state.Version}
	if err := state.Save(path, original); err != nil {
		t.Fatal(err)
	}
	m, err := beginMutation(f.home, f.root)
	if err != nil {
		t.Fatal(err)
	}
	defer m.release()
	external := []byte(`{"version":1,"items":[],"external":"retain"}`)
	if err := os.WriteFile(path, external, 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	err = m.saveState(path, original, func(string, *state.State) error { called = true; return nil })
	if err == nil || called || !bytes.Equal(mustRead(t, path), external) {
		t.Fatalf("state overwritten/adopted: %v called=%t", err, called)
	}
}

func TestMutationStatePersistenceReadbackFailure(t *testing.T) {
	f := dp02Setup(t)
	path := filepath.Join(f.home, ".patronus/state.json")
	m, err := beginMutation(f.home, f.root)
	if err != nil {
		t.Fatal(err)
	}
	defer m.release()
	err = m.saveState(path, &state.State{Version: state.Version}, func(path string, _ *state.State) error { return os.WriteFile(path, []byte("external bytes"), 0600) })
	if err == nil || !strings.Contains(err.Error(), "ownership uncertain") || !strings.Contains(err.Error(), path) {
		t.Fatalf("missing readback uncertainty: %v", err)
	}
	if string(mustRead(t, path)) != "external bytes" {
		t.Fatal("readback failure retried write")
	}
	if err := m.check(path); err == nil {
		t.Fatal("unverified state bytes advanced the snapshot")
	}
}

func TestMutationPiSelectionRejectsProvisioningBeforeWrites(t *testing.T) {
	f := dp02Setup(t)
	path := filepath.Join(f.home, ".pi/agent/prompts/fixture.md")
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{
		{Artifact: "fixture", Tool: "pi", Scope: "global", Type: "command", Action: diff.Create, Path: path, After: []byte("inert prompt")},
		{Artifact: "invented-provisioner", Tool: "agnostic", Scope: "global", Action: diff.Exec, Exec: &diff.ExecSpec{Command: []string{"never-run"}}},
	}}
	runner := &fakeRunner{}
	cmd := newInstallCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := runDeployWith(cmd, cs, toolpath.New(os.LookupEnv, f.home, f.root), deployOptions{home: f.home, projectDir: f.root, allowPkgInstalls: true}, runner)
	if err == nil || !strings.Contains(err.Error(), "EXEC/provisioning") {
		t.Fatalf("unsafe static selection: %v", err)
	}
	if len(runner.ran) != 0 {
		t.Fatal("executed runtime code")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("wrote before static admission")
	}
}

func TestMutationExplicitPiRejectsAgnosticProvisioning(t *testing.T) {
	for _, target := range []string{"pi", "claude", ""} {
		t.Run("target="+target, func(t *testing.T) {
			f := dp02Setup(t)
			dp02File(t, filepath.Join(f.root, "recipes/fixture-provisioner.yaml"), `apiVersion: patronus/v2
family: recipe
name: fixture-provisioner
version: 1.0.0
role: tools
summary: Invented install-only fixture
scope:
  default: global
deliver:
  via: package-manager
  install:
    - manager: uv
      ref: invented-package
`)
			bin := t.TempDir()
			manager := filepath.Join(bin, "uv")
			if err := os.WriteFile(manager, []byte("# inert manager fixture; must never execute\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin)
			runner := &fakeRunner{}
			oldRunner := runnerForCommands
			runnerForCommands = runner
			t.Cleanup(func() { runnerForCommands = oldRunner })
			before := dp02Snapshot(t, f.home, f.root)
			args := []string{"fixture-provisioner", "--global", "--deploy", "--allow-package-installs"}
			if target != "" {
				args = append(args, "--target", target)
			}
			_, _, err := runInstall(t, args...)
			if target == "pi" {
				if err == nil || !strings.Contains(err.Error(), "EXEC/provisioning") || len(runner.ran) != 0 {
					t.Fatalf("Pi provisioning not refused: err=%v executions=%v", err, runner.ran)
				}
				if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
					t.Fatal("Pi refusal changed destination/state/profile lock/receipt bytes")
				}
				return
			}
			if err != nil || len(runner.ran) != 1 || runner.ran[0][0] != "uv" {
				t.Fatalf("legacy provisioning changed: err=%v executions=%v", err, runner.ran)
			}
		})
	}
}

func TestMutationPreviewDoesNotAcquireBusyLock(t *testing.T) {
	f := dp02Setup(t)
	dp02Artifact(t, f.root, "fixture-prompt", "command", "Inert preview\n")
	release, err := packagestate.Acquire(f.home)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, _, err := runInstall(t, "fixture-prompt", "--target", "pi", "--global", "--dry-run"); err != nil {
		t.Fatalf("read-only preview required mutation lock: %v", err)
	}
	oldJSON := jsonOutput
	jsonOutput = true
	defer func() { jsonOutput = oldJSON }()
	oldCatalog := scanCatalogFn
	scanCatalogFn = func(context.Context, string, func(string, ...any)) *registry.Catalog { return nil }
	defer func() { scanCatalogFn = oldCatalog }()
	if _, _, err := runScan(t); err != nil {
		t.Fatalf("read-only scan required mutation lock: %v", err)
	}
}

type dp05RunnerFunc func([]string) error

func (f dp05RunnerFunc) Run(argv []string) error { return f(argv) }

func TestMutationExternalStateEditAfterWriteReportsPartialUncertainty(t *testing.T) {
	f := dp02Setup(t)
	path := filepath.Join(f.home, ".claude/prompts/fixture.md")
	sp := filepath.Join(f.home, ".patronus/state.json")
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{
		{Artifact: "fixture", Tool: "claude", Scope: "global", Action: diff.Create, Path: path, After: []byte("committed inert bytes")},
		{Artifact: "fixture-command", Tool: "claude", Scope: "global", Action: diff.Exec, Exec: &diff.ExecSpec{Command: []string{"invented-command"}}},
	}}
	external := []byte(`{"version":1,"items":[],"editor":"preserve"}`)
	runner := dp05RunnerFunc(func([]string) error {
		release, err := packagestate.Acquire(f.home)
		if release != nil {
			_ = release()
		}
		if !errors.Is(err, packagestate.ErrBusy) {
			t.Fatalf("outer lock released before state persistence: %v", err)
		}
		return os.WriteFile(sp, external, 0600)
	})
	cmd := newInstallCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := runDeployWith(cmd, cs, toolpath.New(os.LookupEnv, f.home, f.root), deployOptions{home: f.home, projectDir: f.root}, runner)
	if err == nil || !strings.Contains(err.Error(), "ownership uncertain") || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "re-preview") {
		t.Fatalf("missing partial effect diagnostic: %v", err)
	}
	if !bytes.Equal(mustRead(t, sp), external) || string(mustRead(t, path)) != "committed inert bytes" {
		t.Fatal("lost external state or committed path")
	}
}

func TestMutationLockSaveRejectsExternalEdit(t *testing.T) {
	f := dp02Setup(t)
	path := filepath.Join(f.root, "patronus.lock")
	m, err := beginMutation(f.home, f.root)
	if err != nil {
		t.Fatal(err)
	}
	defer m.release()
	external := []byte(`{"version":2,"profile":"editor","entries":[]}`)
	dp02File(t, path, string(external))
	err = m.saveLock(path, &lock.Lock{Version: lock.Version})
	if err == nil || !strings.Contains(err.Error(), "fresh preview") || !bytes.Equal(mustRead(t, path), external) {
		t.Fatalf("lock drift overwritten: %v", err)
	}
}
