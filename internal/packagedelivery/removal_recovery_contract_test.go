//go:build linux || darwin

package packagedelivery_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/darkquasar/patronus/internal/packagedelivery"
	"github.com/darkquasar/patronus/internal/packagestate"
)

func removalContent(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	removalCheck(t, err)
	if !bytes.Equal(got, want) {
		t.Fatalf("content changed at %s: %q", path, got)
	}
}

func TestRemovalContractEditsUnknownAndForce(t *testing.T) {
	home, root := removalFixture(t, 3)
	release := removalLock(t, home)
	defer release()
	edited := filepath.Join(root, "d000/f000000")
	unknown := filepath.Join(root, "d000/unknown")
	removalCheck(t, os.WriteFile(edited, []byte("user edit"), 0644))
	removalCheck(t, os.WriteFile(unknown, []byte("not owned"), 0644))
	s := packagedelivery.Service{Home: home}
	result, err := s.Remove(context.Background(), removalRecipe, false)
	removalCheck(t, err)
	if !slices.Equal(result.Retained, []string{"d000/f000000"}) || !slices.Equal(result.Leftovers, []string{"d000/unknown"}) {
		t.Fatalf("preservation result: %+v", result)
	}
	r, err := packagestate.Load(home, removalRecipe)
	removalCheck(t, err)
	if r == nil || len(r.Files) != 1 || r.Files[0].Path != "d000/f000000" {
		t.Fatalf("remaining ownership: %+v", r)
	}
	removalContent(t, edited, []byte("user edit"))
	removalContent(t, unknown, []byte("not owned"))
	removalCheck(t, packagestate.ClearTransaction(home, removalRecipe))
	result, err = s.Remove(context.Background(), removalRecipe, true)
	removalCheck(t, err)
	if result.Receipt != nil || len(result.Retained) != 0 || !slices.Equal(result.Leftovers, []string{"d000/unknown"}) {
		t.Fatalf("force result: %+v", result)
	}
	if _, err := os.Stat(edited); !os.IsNotExist(err) {
		t.Fatalf("force retained edited owned path: %v", err)
	}
	removalContent(t, unknown, []byte("not owned"))
}

// The child announces a reached Service.Fault seam, then blocks. The parent sends
// actual SIGKILL and waits/reaps it. This is not os.Exit and not a power-cut model.
func TestRemovalContractSIGKILLChild(t *testing.T) {
	home := os.Getenv("PATRONUS_REMOVAL_CONTRACT_HOME")
	if home == "" {
		t.Skip("subprocess helper")
	}
	release := removalLock(t, home)
	defer release()
	occurrence, err := strconv.Atoi(os.Getenv("PATRONUS_REMOVAL_CONTRACT_OCCURRENCE"))
	removalCheck(t, err)
	s := packagedelivery.Service{Home: home, Fault: func(point string) error {
		if point == os.Getenv("PATRONUS_REMOVAL_CONTRACT_POINT") {
			occurrence--
			if occurrence == 0 {
				removalCheck(t, os.WriteFile(filepath.Join(home, "child-ready"), []byte(point), 0600))
				for {
					time.Sleep(time.Second)
				}
			}
		}
		return nil
	}}
	_, err = s.Remove(context.Background(), removalRecipe, true)
	t.Fatalf("child did not block at requested seam: %v", err)
}

func removalKillAt(t *testing.T, home, point string, occurrence int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRemovalContractSIGKILLChild$", "-test.count=1", "-test.timeout=25s")
	cmd.Env = append(os.Environ(),
		"PATRONUS_REMOVAL_CONTRACT_HOME="+home,
		"PATRONUS_REMOVAL_CONTRACT_POINT="+point,
		fmt.Sprintf("PATRONUS_REMOVAL_CONTRACT_OCCURRENCE=%d", occurrence))
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	removalCheck(t, cmd.Start())
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	settled := false
	defer func() {
		if !settled {
			_ = cmd.Process.Kill()
			<-done
		}
	}()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			settled = true
			t.Fatalf("child exited before SIGKILL: %v\n%s", err, output.String())
		case <-ctx.Done():
			t.Fatal("child did not reach crash seam before deadline")
		case <-ticker.C:
			if _, err := os.Stat(filepath.Join(home, "child-ready")); os.IsNotExist(err) {
				continue
			} else {
				removalCheck(t, err)
			}
			removalCheck(t, cmd.Process.Kill())
			err := <-done
			settled = true
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("expected killed child, got %v", err)
			}
			status, ok := exit.Sys().(syscall.WaitStatus)
			if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
				t.Fatalf("not SIGKILL: %v", err)
			}
			return
		}
	}
}

func TestRemovalContractSIGKILLRestart(t *testing.T) {
	for _, point := range []string{"after-unlink-intent", "after-unlink", "after-removal-commit"} {
		t.Run(point, func(t *testing.T) {
			home, root := removalFixture(t, 3)
			removalKillAt(t, home, point, 1)
			release := removalLock(t, home) // Also proves the killed child's lock was released.
			defer release()
			s := packagedelivery.Service{Home: home}
			removalCheck(t, s.Recover(context.Background(), removalRecipe))
			removalCheck(t, s.Recover(context.Background(), removalRecipe))
			if _, err := os.Stat(root); !os.IsNotExist(err) {
				t.Fatalf("root survives restart: %v", err)
			}
			r, err := packagestate.Load(home, removalRecipe)
			removalCheck(t, err)
			if r != nil {
				t.Fatal("completed removal retained ownership")
			}
			tx, err := packagestate.ReadTransaction(home, removalRecipe)
			removalCheck(t, err)
			if tx == nil {
				t.Fatal("discovery acknowledgement cleared prematurely")
			}
			removalCheck(t, packagestate.ClearTransaction(home, removalRecipe))
			removalCheck(t, s.Recover(context.Background(), removalRecipe))
		})
	}
}

func TestRemovalContractRecreatedCompletedPath(t *testing.T) {
	for _, identical := range []bool{true, false} {
		t.Run(fmt.Sprintf("identical=%t", identical), func(t *testing.T) {
			home, root := removalFixture(t, 3)
			first := filepath.Join(root, "d000/f000000")
			data, err := os.ReadFile(first)
			removalCheck(t, err)
			// Reaching the next file's intent means the first completed durably,
			// without inspecting any journal representation or cursor field.
			removalKillAt(t, home, "after-unlink-intent", 2)
			release := removalLock(t, home)
			defer release()
			if _, err := os.Stat(first); !os.IsNotExist(err) {
				t.Fatalf("first path not removed: %v", err)
			}
			if !identical {
				data = []byte("new user data")
			}
			removalCheck(t, os.WriteFile(first, data, 0644))
			s := packagedelivery.Service{Home: home}
			removalCheck(t, s.Recover(context.Background(), removalRecipe))
			result, err := s.Remove(context.Background(), removalRecipe, true)
			removalCheck(t, err)
			if result.Receipt != nil || !slices.Contains(result.Leftovers, "d000/f000000") {
				t.Fatalf("recreated path acquired ownership: %+v", result)
			}
			removalContent(t, first, data)
			removalCheck(t, packagestate.ClearTransaction(home, removalRecipe))
			if _, err := s.Remove(context.Background(), removalRecipe, true); err == nil {
				t.Fatal("force without receipt authorized removal")
			}
			removalContent(t, first, data)
		})
	}
}

func TestRemovalContractPendingEditAfterForce(t *testing.T) {
	home, root := removalFixture(t, 3)
	path := filepath.Join(root, "d000/f000000")
	removalCheck(t, os.WriteFile(path, []byte("force-authorized edit"), 0644))
	removalKillAt(t, home, "after-unlink-intent", 1)
	release := removalLock(t, home)
	defer release()
	removalCheck(t, os.WriteFile(path, []byte("later edit"), 0644))
	s := packagedelivery.Service{Home: home}
	if err := s.Recover(context.Background(), removalRecipe); !errors.Is(err, packagedelivery.ErrRecoveryRequired) {
		t.Fatalf("pending edit did not block restart: %v", err)
	}
	removalContent(t, path, []byte("later edit"))
	if _, err := os.Stat(filepath.Join(root, "d000/f000001")); err != nil {
		t.Fatalf("recovery continued after conflict: %v", err)
	}
	removalCheck(t, os.WriteFile(path, []byte("force-authorized edit"), 0644))
	removalCheck(t, s.Recover(context.Background(), removalRecipe))
}

// Opt in with an already-built pre-checkpoint binary. This exercises the actual
// old CLI's mutation/discovery barrier, rather than today's legacy validator.
func TestRemovalContractLegacyBinaryRefusesPending(t *testing.T) {
	binary := os.Getenv("PATRONUS_REMOVAL_LEGACY_BINARY")
	if binary == "" {
		t.Skip("requires an existing pre-checkpoint Patronus binary")
	}
	if !filepath.IsAbs(binary) {
		t.Fatal("legacy binary path must be absolute")
	}
	home, root := removalFixture(t, 3)
	s := packagedelivery.Service{Home: home}
	func() {
		release := removalLock(t, home)
		defer release()
		removalPrepareRestart(t, &s)
	}() // The old CLI must acquire its own lock, not block behind this test.
	journal := filepath.Join(home, ".patronus", "package-state", "transactions", removalRecipe, "transaction.json")
	before, err := os.ReadFile(journal)
	removalCheck(t, err)
	contents := make([][]byte, 3)
	for i := range contents {
		contents[i], err = os.ReadFile(filepath.Join(root, fmt.Sprintf("d000/f%06d", i)))
		removalCheck(t, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "remove", removalRecipe, "--global", "--force", "--deploy")
	cmd.Dir = home
	// No inherited tool/home overrides or live installation inputs.
	cmd.Env = []string{"HOME=" + home, "TMPDIR=" + home, "PATH=/usr/bin:/bin"}
	output, err := cmd.CombinedOutput() // Wait and reap even on timeout.
	if ctx.Err() != nil || err == nil || !strings.Contains(string(output), `unknown field "removalID"`) {
		t.Fatalf("legacy CLI did not reject the new marker: %v: %s", err, output)
	}
	t.Logf("legacy refusal: %s", output)
	after, err := os.ReadFile(journal)
	removalCheck(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("legacy CLI rewrote pending removal evidence")
	}
	for i, content := range contents {
		removalContent(t, filepath.Join(root, fmt.Sprintf("d000/f%06d", i)), content)
	}
	release := removalLock(t, home)
	defer release()
	removalCheck(t, s.Recover(context.Background(), removalRecipe))
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("supporting binary did not finish removal: %v", err)
	}
}

func TestRemovalContractCancellationAndRestart(t *testing.T) {
	home, root := removalFixture(t, 3)
	release := removalLock(t, home)
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancel()
	s := packagedelivery.Service{Home: home}
	if _, err := s.Remove(ctx, removalRecipe, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancelled remove: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := os.Stat(filepath.Join(root, fmt.Sprintf("d000/f%06d", i))); err != nil {
			t.Fatalf("pre-cancelled remove mutated files: %v", err)
		}
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	s.Fault = func(point string) error {
		if point == "after-unlink-intent" {
			cancel()
		}
		return nil
	}
	if _, err := s.Remove(ctx, removalRecipe, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("in-flight cancellation lost: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "d000/f000001")); err != nil {
		t.Fatalf("continued to next pending path after cancellation: %v", err)
	}
	s = packagedelivery.Service{Home: home}
	removalCheck(t, s.Recover(context.Background(), removalRecipe))
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("restart did not finish cancelled removal: %v", err)
	}
}
