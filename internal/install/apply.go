// Package install realizes a computed change set on disk. It consumes the same
// diff.ChangeSet the planner produces and the dry-run renderer displays, so
// there is one change model from compute to apply. Writes are atomic per file
// and Terraform-style on failure: stop at the first error, keep what already
// succeeded, and surface the error (no whole-set rollback).
package install

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/darkquasar/patronus/internal/archive"
	"github.com/darkquasar/patronus/internal/diff"
)

// Fetcher downloads the bytes at a URL for a FETCH apply. It is consumer-defined
// here (the cmd layer injects recipe.HTTPFetcher; tests inject a fake) so the
// install package depends only on diff + archive — no cycle with the recipe
// engine that produces FETCH diffs.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (io.ReadCloser, error)
}

// Resolution is the user's answer to a CONFLICT prompt.
type Resolution int

const (
	Skip      Resolution = iota // leave the existing file untouched
	Overwrite                   // replace it with the computed content
)

// ConflictFunc is asked how to resolve a CONFLICT (target exists & differs from
// a CREATE). The renderer can show d.Unified() before prompting. nil means
// non-interactive: every conflict is skipped (never silently overwritten).
type ConflictFunc func(d diff.FileDiff) (Resolution, error)

// Applier writes change sets to disk.
type Applier struct {
	// Force overwrites conflicting files without prompting.
	Force bool
	// Conflict resolves CONFLICT actions when Force is false. nil => skip.
	Conflict ConflictFunc
	// Progress, if set, receives a one-line note per applied op.
	Progress io.Writer
	// Fetcher downloads recipe binaries for FETCH ops. nil for pure-artifact
	// installs; a FETCH diff reaching a nil Fetcher fails loudly (never panics).
	Fetcher Fetcher
	// Ctx, if set, scopes downloads (cancellation/timeout). Defaults to Background.
	Ctx context.Context
}

// Result reports the outcome of an Apply. Applied lists the ops actually written
// (the input for state recording); Failed, if non-nil, is the op whose write
// errored (everything after it was not attempted).
type Result struct {
	Applied []diff.FileDiff
	Skipped []diff.FileDiff
	Failed  *diff.FileDiff
}

// Apply writes cs to disk and returns what happened. On the first write error it
// stops and returns the partial Result alongside the error, mirroring
// Terraform: state reflects reality, re-running is safe (done files SKIP).
func (a *Applier) Apply(cs *diff.ChangeSet) (*Result, error) {
	res := &Result{}
	for i := range cs.Diffs {
		d := cs.Diffs[i]
		if d.IsDir {
			continue // display-only summary row
		}

		switch d.Action {
		case diff.Skip:
			res.Skipped = append(res.Skipped, d)
			continue

		case diff.Exec:
			// Self-wiring post-install commands are NOT run by the file writer —
			// they have no atomicity and no revert inverse. The cmd layer runs
			// them post-apply on --deploy. Here they are display-only.
			res.Skipped = append(res.Skipped, d)
			continue

		case diff.Fetch:
			if err := a.applyFetch(d); err != nil {
				res.Failed = &d
				return res, err
			}
			a.note("FETCH %s", d.Path)
			res.Applied = append(res.Applied, d)
			continue

		case diff.Conflict:
			how, err := a.resolveConflict(d)
			if err != nil {
				res.Failed = &d
				return res, err
			}
			if how == Skip {
				res.Skipped = append(res.Skipped, d)
				continue
			}
			// Overwrite falls through to the write below.

		case diff.Create, diff.Append, diff.Merge:
			// write below

		default:
			res.Failed = &d
			return res, fmt.Errorf("install: unknown action %q for %s", d.Action, d.Path)
		}

		if err := WriteFileAtomic(d.Path, d.After, 0o644); err != nil {
			res.Failed = &d
			return res, fmt.Errorf("install: write %s: %w", d.Path, err)
		}
		a.note("%s %s", d.Action, d.Path)
		res.Applied = append(res.Applied, d)
	}
	return res, nil
}

// applyFetch downloads a recipe binary, verifies its sha256, extracts it from an
// archive when needed, and places it at the destination with the executable bit.
// A verify failure stops the apply Terraform-style — an unverified binary is
// never placed.
func (a *Applier) applyFetch(d diff.FileDiff) error {
	spec := d.Fetch
	if spec == nil {
		return fmt.Errorf("install: FETCH %s has no fetch spec", d.Path)
	}
	if a.Fetcher == nil {
		return fmt.Errorf("install: FETCH %s requires a fetcher (none configured)", d.Path)
	}

	ctx := a.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	body, err := a.Fetcher.Fetch(ctx, spec.URL)
	if err != nil {
		return fmt.Errorf("install: fetch %s: %w", spec.URL, err)
	}
	defer body.Close()

	data, err := verifySHA256(body, spec.SHA256)
	if err != nil {
		return fmt.Errorf("install: %s: %w", spec.Label, err)
	}

	// If the asset is an archive, extract the named binary member; otherwise the
	// downloaded bytes are the binary.
	if spec.Archive != "" {
		bin, err := archive.ExtractFile(bytes.NewReader(data), spec.Archive, spec.BinaryPath)
		if err != nil {
			return fmt.Errorf("install: extract %s: %w", spec.Label, err)
		}
		data = bin
	}

	if err := WriteFileAtomic(spec.Dest, data, 0o755); err != nil {
		return fmt.Errorf("install: place %s: %w", spec.Dest, err)
	}
	// Stamp the digest of the binary actually placed (the extracted member for an
	// archive), so state records the on-disk binary's sha, not the archive's.
	sum := sha256.Sum256(data)
	spec.PlacedSHA256 = hex.EncodeToString(sum[:])
	return nil
}

// verifySHA256 reads all of r and confirms its sha256 matches wantHex (optionally
// "sha256:"-prefixed). Returns the verified bytes. A mismatch is an error.
func verifySHA256(r io.Reader, wantHex string) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read download: %w", err)
	}
	want := wantHex
	if len(want) > 7 && want[:7] == "sha256:" {
		want = want[7:]
	}
	if want == "" {
		return nil, fmt.Errorf("verify: no expected sha256 pinned")
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != lowerHex(want) {
		return nil, fmt.Errorf("verify: sha256 mismatch (got %s, want %s)", got, lowerHex(want))
	}
	return data, nil
}

func lowerHex(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

// resolveConflict decides whether a CONFLICT op should be written.
func (a *Applier) resolveConflict(d diff.FileDiff) (Resolution, error) {
	if a.Force {
		return Overwrite, nil
	}
	if a.Conflict == nil {
		return Skip, nil // non-interactive: never silently overwrite
	}
	return a.Conflict(d)
}

func (a *Applier) note(format string, args ...any) {
	if a.Progress != nil {
		fmt.Fprintf(a.Progress, format+"\n", args...)
	}
}

// WriteFileAtomic writes data to path via a temp file in the same directory then
// renames it over the target. Rename within a directory is atomic on POSIX and
// Windows, so a crash can never leave a half-written file. Parent directories
// are created as needed. Exported because the state writer reuses it.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".patronus-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we bail before the rename succeeds.
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	tmpName = "" // rename succeeded; don't remove the now-real file
	return nil
}
