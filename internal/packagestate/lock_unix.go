//go:build darwin || linux

package packagestate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// Acquire takes the host-wide advisory lock without waiting. Call release on exit.
func Acquire(home string) (func() error, error) {
	h, err := canonicalHome(home)
	if err != nil {
		return nil, err
	}
	dir, err := directory(h, filepath.Join(".patronus", "package-state"), true)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "mutation.lock")
	if err := regular(path); err != nil {
		return nil, err
	}
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	if err := f.Chmod(0600); err != nil {
		_ = f.Close()
		return nil, err
	}
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("%w: %s", ErrBusy, path)
		}
		return nil, err
	}
	var once sync.Once
	var releaseErr error
	return func() error {
		once.Do(func() { releaseErr = errors.Join(syscall.Flock(fd, syscall.LOCK_UN), f.Close()) })
		return releaseErr
	}, nil
}

// nearestDevice handles stage and active roots that do not exist yet.
func nearestDevice(path string) (uint64, error) {
	for {
		info, err := os.Lstat(path)
		if err == nil {
			stat, ok := info.Sys().(*syscall.Stat_t)
			if !ok {
				return 0, errors.New("filesystem identity unavailable")
			}
			return uint64(stat.Dev), nil //nolint:unconvert // Darwin uses a signed device identifier.
		}
		if !os.IsNotExist(err) {
			return 0, err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return 0, err
		}
		path = parent
	}
}
func sameFilesystem(root, other string) error {
	a, err := nearestDevice(root)
	if err != nil {
		return err
	}
	b, err := nearestDevice(other)
	if err != nil {
		return err
	}
	if a != b {
		return fmt.Errorf("transaction path is on a different filesystem: %s", other)
	}
	return nil
}
