//go:build linux || darwin

package packagedelivery_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagedelivery"
	"github.com/darkquasar/patronus/internal/packagestate"
)

const removalRecipe = "removal-fixture"

// removalFixture invents an ownership inventory, not an installed web package.
// Every file is 128 bytes, mode 0644; there are <=100 files per one-level directory.
// Save, Remove and Recover use real filesystem persistence without overrides.
func removalFixture(tb testing.TB, n int) (home, root string) {
	tb.Helper()
	home, err := filepath.EvalSymlinks(tb.TempDir())
	removalCheck(tb, err)
	root = filepath.Join(home, ".patronus", "packages", removalRecipe)
	release := removalLock(tb, home)
	defer release()
	data := bytes.Repeat([]byte("synthetic-data!\n"), 9)[:128]
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	r := &packagestate.Receipt{
		SchemaVersion: 1, Recipe: removalRecipe, RecipeVersion: "1.0.0", Root: root,
		URL: "https://example.invalid/synthetic.tar.gz", ArchiveSHA256: digest,
		Identity:    packagebundle.Identity{Name: removalRecipe, Version: "1.0.0", OS: "linux", Arch: "amd64"},
		Directories: []string{"."},
	}
	for i := 0; i < n; i++ {
		dir := fmt.Sprintf("d%03d", i/100)
		if i%100 == 0 {
			removalCheck(tb, os.MkdirAll(filepath.Join(root, dir), 0755))
			r.Directories = append(r.Directories, dir)
		}
		name := fmt.Sprintf("%s/f%06d", dir, i)
		path := filepath.Join(root, name)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		removalCheck(tb, err)
		_, err = f.Write(data)
		removalCheck(tb, err)
		removalCheck(tb, f.Sync())
		removalCheck(tb, f.Close())
		r.Files = append(r.Files, packagebundle.Entry{Path: name, Mode: 0644, SHA256: digest})
	}
	for _, dir := range append(r.Directories, "..", "../..", "../../..") {
		f, err := os.Open(filepath.Join(root, dir))
		removalCheck(tb, err)
		removalCheck(tb, f.Sync())
		removalCheck(tb, f.Close())
	}
	removalCheck(tb, packagestate.Save(home, r))
	return home, root
}

func removalCheck(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Fatal(err)
	}
}

func removalLock(tb testing.TB, home string) func() {
	tb.Helper()
	release, err := packagestate.Acquire(home)
	removalCheck(tb, err)
	return func() { removalCheck(tb, release()) }
}

func removalUsage(tb testing.TB) (user, system time.Duration) {
	tb.Helper()
	var usage syscall.Rusage
	removalCheck(tb, syscall.Getrusage(syscall.RUSAGE_SELF, &usage))
	return time.Duration(usage.Utime.Nano()), time.Duration(usage.Stime.Nano())
}

func BenchmarkRemovalRealPersistence(b *testing.B) {
	benchmarkRemoval(b, false, false)
}

// Recovery starts from the first durable unlink intent. Preparing the interrupted
// transaction is not timed. This is restart work, not total Remove+retry latency.
func BenchmarkRemovalRecoveryRealPersistence(b *testing.B) {
	benchmarkRemoval(b, true, false)
}

// This includes the interrupted Remove AND a fresh Service.Recover. It excludes
// process launch and operator delay; the interruption is a returned fault error.
func BenchmarkRemovalInterruptedTotalRealPersistence(b *testing.B) {
	benchmarkRemoval(b, false, true)
}

func removalPrepareRestart(tb testing.TB, s *packagedelivery.Service) {
	tb.Helper()
	stop := errors.New("prepare restart measurement")
	s.Fault = func(point string) error {
		if point == "after-unlink-intent" {
			return stop
		}
		return nil
	}
	_, err := s.Remove(context.Background(), removalRecipe, false)
	if !errors.Is(err, stop) {
		tb.Fatalf("did not reach restart seam: %v", err)
	}
	*s = packagedelivery.Service{Home: s.Home}
}

func benchmarkRemoval(b *testing.B, recoverOnly, totalRetry bool) {
	for _, n := range []int{100, 1000, 4014} {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			b.StopTimer()
			var user, system time.Duration
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				home, root := removalFixture(b, n)
				release := removalLock(b, home)
				s := packagedelivery.Service{Home: home}
				if recoverOnly {
					removalPrepareRestart(b, &s)
				}
				u0, s0 := removalUsage(b)
				b.StartTimer()
				if totalRetry {
					removalPrepareRestart(b, &s)
				}
				if recoverOnly || totalRetry {
					removalCheck(b, s.Recover(context.Background(), removalRecipe))
				} else {
					result, err := s.Remove(context.Background(), removalRecipe, false)
					removalCheck(b, err)
					if !result.Mutated || result.Receipt != nil || len(result.Retained)+len(result.Leftovers) != 0 {
						b.Fatalf("unexpected removal result: %+v", result)
					}
				}
				b.StopTimer()
				u1, s1 := removalUsage(b)
				user += u1 - u0
				system += s1 - s0
				if _, err := os.Stat(root); !os.IsNotExist(err) {
					b.Fatalf("root survives: %v", err)
				}
				r, err := packagestate.Load(home, removalRecipe)
				removalCheck(b, err)
				if r != nil {
					b.Fatal("receipt survives")
				}
				// Discovery acknowledgement is deliberately outside removal timing.
				removalCheck(b, packagestate.ClearTransaction(home, removalRecipe))
				release()
				removalCheck(b, os.RemoveAll(home))
			}
			b.ReportMetric(float64(user.Nanoseconds())/float64(b.N), "user-ns/op")
			b.ReportMetric(float64(system.Nanoseconds())/float64(b.N), "sys-ns/op")
		})
	}
}

// Optional instrumentation, never a headline timing. Linux process I/O counters
// count logical bytes/write syscalls, not physical device traffic or fsync calls.
func TestRemovalPerformanceLogicalIO(t *testing.T) {
	if runtime.GOOS != "linux" || os.Getenv("PATRONUS_REMOVAL_IO") != "1" {
		t.Skip("opt in with PATRONUS_REMOVAL_IO=1 on Linux")
	}
	readIO := func() map[string]int64 {
		data, err := os.ReadFile("/proc/self/io")
		removalCheck(t, err)
		counters := make(map[string]int64)
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			fields := strings.Fields(line)
			value, err := strconv.ParseInt(fields[1], 10, 64)
			removalCheck(t, err)
			counters[strings.TrimSuffix(fields[0], ":")] = value
		}
		return counters
	}
	for _, n := range []int{100, 1000, 4014} {
		t.Run(fmt.Sprintf("N%d", n), func(t *testing.T) {
			home, _ := removalFixture(t, n)
			release := removalLock(t, home)
			defer release()
			s := packagedelivery.Service{Home: home}
			before := readIO()
			_, err := s.Remove(context.Background(), removalRecipe, false)
			after := readIO()
			removalCheck(t, err)
			t.Logf("logical_io n=%d wchar=%d syscw=%d", n, after["wchar"]-before["wchar"], after["syscw"]-before["syscw"])
		})
	}
}
