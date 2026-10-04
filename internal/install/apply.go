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
	"github.com/darkquasar/patronus/internal/packagebundle"
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
	// BeforeWrite revalidates command-level state/consent immediately before an
	// affected write. The file snapshot check always runs afterwards, even with Force.
	BeforeWrite func(diff.FileDiff) error
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
	// readBack reads a file back after writing it, so state records the bytes that
	// actually reached disk rather than the bytes we intended to write. nil means
	// os.ReadFile. A real filesystem will not return different bytes than it was
	// given, so this exists to make the mismatch branch testable.
	readBack func(string) ([]byte, error)
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
// stops and returns the partial Result alongside the error. Successful writes
// remain real; a verification/state failure needs a fresh preview and explicit
// repair, not adoption of equal bytes on retry.
func (a *Applier) Apply(cs *diff.ChangeSet) (*Result, error) {
	res := &Result{}
	verified := map[string][]byte{}
	for i := range cs.Diffs {
		d := cs.Diffs[i]
		if d.Native != nil {
			res.Failed = &d
			return res, fmt.Errorf("native package %q requires the native lifecycle service", d.Artifact)
		}
		if d.Directory != nil {
			res.Failed = &d
			return res, fmt.Errorf("directory package %q requires the package delivery service", d.Artifact)
		}
		if d.IsDir {
			continue // display-only summary row
		}

		switch d.Action {
		case diff.Skip:
			expected := d.Before
			if own, ok := verified[d.Path]; ok {
				expected = own
			}
			if err := CheckUnchanged(d.Path, expected); err != nil {
				res.Failed = &d
				return res, err
			}
			res.Skipped = append(res.Skipped, d)
			continue

		case diff.Exec:
			// Self-wiring post-install commands are NOT run by the file writer —
			// they have no atomicity and no revert inverse. The cmd layer runs
			// them post-apply on --deploy. Here they are display-only.
			res.Skipped = append(res.Skipped, d)
			continue

		case diff.Fetch:
			if err := a.checkBeforeWrite(d); err != nil {
				res.Failed = &d
				return res, err
			}
			observed, err := a.applyFetch(d)
			if err != nil {
				res.Failed = &d
				return res, err
			}
			verified[d.Path] = observed
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

		case diff.Delete:
			// Inverse of CREATE/FETCH (Phase 8 remove): drop the file. A missing
			// target is success — re-running a remove is idempotent.
			if err := a.checkBeforeWrite(d); err != nil {
				res.Failed = &d
				return res, err
			}
			if err := os.Remove(d.Path); err != nil && !os.IsNotExist(err) {
				res.Failed = &d
				return res, fmt.Errorf("install: remove %s: %w", d.Path, err)
			}
			if _, err := os.Lstat(d.Path); !os.IsNotExist(err) {
				res.Failed = &d
				if err != nil {
					return res, fmt.Errorf("install: delete committed, verify %s: %w; ownership uncertain", d.Path, err)
				}
				return res, fmt.Errorf("install: delete committed, verify %s: still present; ownership uncertain", d.Path)
			}
			verified[d.Path] = nil
			a.note("DELETE %s", d.Path)
			res.Applied = append(res.Applied, d)
			continue

		case diff.Create, diff.Append, diff.Merge, diff.Unappend, diff.Restore:
			// write d.After below. Unappend (the file minus our section) and Restore
			// (the recorded pre-install bytes) use the same atomic write as a forward
			// edit — the inverse is just different bytes, not new machinery.

		default:
			res.Failed = &d
			return res, fmt.Errorf("install: unknown action %q for %s", d.Action, d.Path)
		}

		perm := fs.FileMode(0o644)
		if d.Mode != 0 {
			perm = d.Mode // e.g. 0o755 for an executable hook script
		}
		if err := a.checkBeforeWrite(d); err != nil {
			res.Failed = &d
			return res, err
		}
		if err := WriteFileAtomic(d.Path, d.After, perm); err != nil {
			res.Failed = &d
			return res, fmt.Errorf("install: write %s: %w", d.Path, err)
		}
		// Read back and verify. A mismatch means the bytes on disk are not the bytes
		// we wrote: a filesystem lie or a concurrent writer, not a planning error. It
		// is unresolved ownership, even though the write already committed.
		if err := a.verifyWritten(d.Path, d.After); err != nil {
			res.Failed = &d
			return res, err
		}
		verified[d.Path] = d.After
		a.note("%s %s", d.Action, d.Path)
		res.Applied = append(res.Applied, d)
	}
	return res, nil
}

// applyFetch downloads a recipe binary, verifies its sha256, extracts it from an
// archive when needed, and places it at the destination with the executable bit.
// A verify failure stops the apply Terraform-style — an unverified binary is
// never placed.
func (a *Applier) applyFetch(d diff.FileDiff) ([]byte, error) {
	spec := d.Fetch
	if spec == nil {
		return nil, fmt.Errorf("install: FETCH %s has no fetch spec", d.Path)
	}
	if a.Fetcher == nil {
		return nil, fmt.Errorf("install: FETCH %s requires a fetcher (none configured)", d.Path)
	}

	ctx := a.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	data, err := AcquireFetch(ctx, spec, a.Fetcher)
	if err != nil {
		return nil, err
	}
	data, err = decodeFetch(data, spec)
	if err != nil {
		return nil, err
	}

	if spec.Dest != d.Path {
		return nil, fmt.Errorf("install: FETCH destination differs from planned path %s", d.Path)
	}
	if err := a.checkBeforeWrite(d); err != nil {
		return nil, err
	}
	if err := WriteFileAtomic(spec.Dest, data, 0o755); err != nil {
		return nil, fmt.Errorf("install: place %s: %w", spec.Dest, err)
	}
	if err := a.verifyWritten(spec.Dest, data); err != nil {
		return nil, err
	}
	// Stamp the digest of the binary actually placed (the extracted member for an
	// archive), so state records the on-disk binary's sha, not the archive's.
	sum := sha256.Sum256(data)
	spec.PlacedSHA256 = hex.EncodeToString(sum[:])
	return data, nil
}

func (a *Applier) verifyWritten(path string, expected []byte) error {
	read := a.readBack
	if read == nil {
		read = os.ReadFile
	}
	observed, err := read(path)
	if err != nil {
		return fmt.Errorf("install: write committed, verify %s: %w; ownership uncertain", path, err)
	}
	if !bytes.Equal(observed, expected) {
		return fmt.Errorf("install: write committed, verify %s: expected %s, observed %s; ownership uncertain", path, shortSHA(expected), shortSHA(observed))
	}
	return nil
}

// verifySHA256 reads all of r and confirms its sha256 matches wantHex (optionally
// "sha256:"-prefixed). Returns the verified bytes. A mismatch is an error.
func verifySHA256(r io.Reader, wantHex string) ([]byte, error) {
	data, err := archive.ReadBounded(r, packagebundle.DefaultLimits.CompressedBytes)
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

// shortSHA renders a content digest for an error message: enough to compare two
// by eye, not so much that the message wraps.
func shortSHA(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])[:12]
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

// CheckUnchanged compares existence and bytes with a planning snapshot (nil
// means absent, a non-nil empty slice means present-empty). It is a last-moment
// advisory check, not atomic exclusion against noncooperating editors.
func CheckUnchanged(path string, before []byte) error {
	for p := filepath.Clean(path); ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("check %s: %w; fresh preview required", p, err)
		}
		if err == nil && (info.Mode()&os.ModeSymlink != 0 || (p == path && !info.Mode().IsRegular()) || (p != path && !info.IsDir())) {
			return fmt.Errorf("check %s: unsafe path/redirection; fresh preview required", p)
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	observed, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("check %s: %w; fresh preview required", path, err)
	}
	if exists != (before != nil) || !bytes.Equal(observed, before) {
		return fmt.Errorf("external change at %s (existence or hash differs); fresh preview required", path)
	}
	return nil
}

func (a *Applier) checkBeforeWrite(d diff.FileDiff) error {
	if a.BeforeWrite != nil {
		if err := a.BeforeWrite(d); err != nil {
			return err
		}
	}
	return CheckUnchanged(d.Path, d.Before)
}

// AcquireFetch verifies bounded acquisition and the selected archive member
// without writing destinations. The returned original bytes can be replayed by
// the operation's temporary fetcher; apply repeats verification before writing.
func AcquireFetch(ctx context.Context, spec *diff.FetchSpec, fetcher Fetcher) ([]byte, error) {
	if spec == nil || fetcher == nil {
		return nil, fmt.Errorf("FETCH requires a spec and fetcher")
	}
	body, err := fetcher.Fetch(ctx, spec.URL)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := verifySHA256(body, spec.SHA256)
	if err != nil {
		return nil, err
	}
	if _, err := decodeFetch(data, spec); err != nil {
		return nil, err
	}
	return data, nil
}

func decodeFetch(data []byte, spec *diff.FetchSpec) ([]byte, error) {
	l := packagebundle.DefaultLimits
	if spec.Archive == "" {
		if int64(len(data)) > l.FileBytes {
			return nil, fmt.Errorf("FETCH file limit exceeded")
		}
		return data, nil
	}
	return archive.ExtractFileBounded(bytes.NewReader(data), spec.Archive, spec.BinaryPath, archive.Limits{CompressedBytes: l.CompressedBytes, DecodedBytes: l.DecodedBytes, ExpandedBytes: l.ExpandedBytes, FileBytes: l.FileBytes, Entries: l.Entries})
}
