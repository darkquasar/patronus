//go:build linux || darwin

package packagestate

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLockContention(t *testing.T) {
	home := t.TempDir()
	release, err := Acquire(home)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	other, err := Acquire(home)
	if err == nil {
		other()
		t.Fatal("second lock succeeded")
	}
	if !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
}

func TestLockChild(t *testing.T) {
	home := os.Getenv("PATRONUS_TEST_LOCK_HOME")
	if home == "" {
		return
	}
	release, err := Acquire(home)
	if errors.Is(err, ErrBusy) {
		os.Exit(23)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if os.Getenv("PATRONUS_TEST_LOCK_HOLD") == "1" {
		fmt.Println("locked")
		var b [1]byte
		if _, err := os.Stdin.Read(b[:]); err != nil {
			t.Fatal(err)
		}
	}
}
func TestLockProcessesAndDeath(t *testing.T) {
	home := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := func(hold bool) *exec.Cmd {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		t.Cleanup(cancel)
		cmd := exec.CommandContext(ctx, exe, "-test.run=^TestLockChild$")
		cmd.Env = append(os.Environ(), "PATRONUS_TEST_LOCK_HOME="+home)
		if hold {
			cmd.Env = append(cmd.Env, "PATRONUS_TEST_LOCK_HOLD=1")
		}
		return cmd
	}
	holder := command(true)
	stdin, err := holder.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	stdout, err := holder.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	defer holder.Process.Kill()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "locked\n" {
		t.Fatalf("holder %q %v", line, err)
	}
	contender := command(false)
	err = contender.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 23 {
		t.Fatalf("contender: %v", err)
	}
	if err := holder.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = holder.Wait()
	if out, err := command(false).CombinedOutput(); err != nil {
		t.Fatalf("lock survived death: %s %v", out, err)
	}
}
func TestLockSymlinkRejected(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".patronus/package-state")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "lock"), filepath.Join(dir, "mutation.lock")); err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(home); err == nil {
		t.Fatal("accepted lock symlink")
	}
}
