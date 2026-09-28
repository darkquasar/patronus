package packagedelivery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagestate"
)

// persistence is service-local fault injection around the real storage contract.
// A failed visible write is never followed by another journal phase or rollback.
type persistence struct {
	write         func(string, *packagestate.Transaction) error
	save          func(string, *packagestate.Receipt) error
	deleteReceipt func(string, string) error
	clear         func(string, string) error
	stabilize     func(string) error
}

func (s *Service) write(tx *packagestate.Transaction) error {
	if s.persistence != nil && s.persistence.write != nil {
		return s.persistence.write(s.Home, tx)
	}
	return packagestate.WriteTransaction(s.Home, tx)
}
func (s *Service) save(r *packagestate.Receipt) error {
	if s.persistence != nil && s.persistence.save != nil {
		return s.persistence.save(s.Home, r)
	}
	return packagestate.Save(s.Home, r)
}
func (s *Service) deleteReceipt(recipe string) error {
	if s.persistence != nil && s.persistence.deleteReceipt != nil {
		return s.persistence.deleteReceipt(s.Home, recipe)
	}
	return packagestate.DeleteReceipt(s.Home, recipe)
}
func (s *Service) clear(recipe string) error {
	if s.persistence != nil && s.persistence.clear != nil {
		return s.persistence.clear(s.Home, recipe)
	}
	return packagestate.ClearTransaction(s.Home, recipe)
}
func (s *Service) fault(point string) error {
	if s.Fault != nil {
		return s.Fault(point)
	}
	return nil
}
func visible(err error) bool {
	var d *packagestate.DurabilityError
	return errors.As(err, &d) && d.MayBeVisible
}
func syncPath(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// durableRename rechecks both directory paths after scans and fault boundaries.
// As with other path-based operations, an external actor can still race the final
// check and syscall; this does not provide descriptor-relative atomic isolation.
func durableRename(home, old, new string) error {
	if err := checkAncestry(home, old); err != nil {
		return err
	}
	if err := checkAncestry(home, new); err != nil {
		return err
	}
	if err := os.Rename(old, new); err != nil {
		return err
	}
	if err := syncPath(filepath.Dir(old)); err != nil {
		return &packagestate.DurabilityError{Path: old, Stage: "rename-parent-sync", MayBeVisible: true, Err: err}
	}
	if err := syncPath(filepath.Dir(new)); err != nil {
		return &packagestate.DurabilityError{Path: new, Stage: "rename-parent-sync", MayBeVisible: true, Err: err}
	}
	return nil
}

// prepareDirectories repairs parent-link durability even for already visible dirs.
func prepareDirectories(home, path string) error {
	if err := checkAncestry(home, path); err != nil {
		return err
	}
	rel, err := filepath.Rel(home, path)
	if err != nil {
		return err
	}
	current := home
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		parent := current
		current = filepath.Join(current, part)
		if err := os.Mkdir(current, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("not a real directory: %s", current)
		}
		if err := syncPath(parent); err != nil {
			return &packagestate.DurabilityError{Path: current, Stage: "create-parent-sync", MayBeVisible: true, Err: err}
		}
	}
	return nil
}
func receiptFor(req Request, files []packagebundle.File) *packagestate.Receipt {
	entries := make([]packagebundle.Entry, 0, len(files))
	for _, f := range files {
		entries = append(entries, packagebundle.Entry{Path: f.Path, Mode: f.Mode, SHA256: bytesDigest(f.Data)})
	}
	slices.SortFunc(entries, func(a, b packagebundle.Entry) int { return strings.Compare(a.Path, b.Path) })
	return &packagestate.Receipt{SchemaVersion: 1, Recipe: req.Recipe, RecipeVersion: req.RecipeVersion, Root: req.Root, URL: req.URL, ArchiveSHA256: req.SHA256, Identity: req.Identity, Files: entries, Directories: ownedDirs(entries, nil)}
}
func (s *Service) stage(ctx context.Context, req Request, bundle *packagebundle.Bundle) (string, string, *packagestate.Receipt, error) {
	home := filepath.Dir(filepath.Dir(filepath.Dir(req.Root)))
	base := filepath.Join(filepath.Dir(req.Root), ".txn", req.Recipe)
	if err := prepareDirectories(home, base); err != nil {
		return "", "", nil, err
	}
	stage, err := os.MkdirTemp(base, "stage-")
	if err != nil {
		return "", "", nil, err
	}
	receipt := receiptFor(req, bundle.Files)
	// The unique stage is not user-visible; even here cleanup checks recorded members.
	fail := func(err error) (string, string, *packagestate.Receipt, error) {
		return "", "", nil, errors.Join(err, cleanupTree(home, req.Recipe, stage, receipt.Files, receipt.Directories))
	}
	if err := os.Chmod(stage, 0755); err != nil {
		return fail(err)
	}
	if err := syncPath(base); err != nil {
		return fail(err)
	}
	for _, file := range bundle.Files {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		path := filepath.Join(stage, filepath.FromSlash(file.Path))
		if err := prepareDirectories(stage, filepath.Dir(path)); err != nil {
			return fail(err)
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, os.FileMode(file.Mode))
		if err != nil {
			return fail(err)
		}
		_, writeErr := f.Write(file.Data)
		chmodErr := f.Chmod(os.FileMode(file.Mode))
		syncErr := f.Sync()
		closeErr := f.Close()
		if err := errors.Join(writeErr, chmodErr, syncErr, closeErr); err != nil {
			return fail(err)
		}
	}
	for i := len(receipt.Directories) - 1; i >= 0; i-- {
		if err := os.Chmod(filepath.Join(stage, receipt.Directories[i]), 0755); err != nil {
			return fail(err)
		}
		if err := syncPath(filepath.Join(stage, receipt.Directories[i])); err != nil {
			return fail(err)
		}
	}
	// MkdirTemp reserves a unique name; remove the empty backup placeholder durably.
	backup, err := os.MkdirTemp(base, "backup-")
	if err != nil {
		return fail(err)
	}
	if err := os.Remove(backup); err != nil {
		return fail(err)
	}
	if err := syncPath(base); err != nil {
		return fail(err)
	}
	return stage, backup, receipt, nil
}

// Replace verifies and replaces an owned tree. The caller must hold Acquire.
// A non-nil error with a committed Result never authorizes rollback by the caller.
func (s *Service) Replace(ctx context.Context, req Request, force bool) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	req, err := s.normalize(req)
	if err != nil {
		return Result{}, err
	}
	if err := s.Recover(ctx, req.Recipe); err != nil {
		return Result{}, err
	}
	in, err := s.Inspect(req)
	if err != nil {
		return Result{}, err
	}
	if err := inspectionConflict(req.Recipe, in, force); err != nil {
		return Result{}, err
	}
	intact := in.Receipt != nil && len(in.Changed) == 0 && len(in.Missing) == 0
	samePayload := intact && completeInventory(in.Receipt) && in.Receipt.ArchiveSHA256 == req.SHA256 && in.Receipt.Identity == req.Identity
	if samePayload && in.Receipt.RecipeVersion == req.RecipeVersion && in.Receipt.URL == req.URL {
		return Result{Receipt: in.Receipt}, nil
	}
	tx := &packagestate.Transaction{SchemaVersion: 1, Recipe: req.Recipe, Root: req.Root, Previous: in.Receipt, Phase: packagestate.Prepared, Operation: "replace"}
	if samePayload {
		copyReceipt := *in.Receipt
		copyReceipt.RecipeVersion = req.RecipeVersion
		copyReceipt.URL = req.URL
		tx.Candidate = &copyReceipt
		tx.Operation = "metadata"
	} else {
		bundle, err := s.fetch(ctx, req)
		if err != nil {
			return Result{}, err
		}
		tx.Stage, tx.Backup, tx.Candidate, err = s.stage(ctx, req, bundle)
		if err != nil {
			return Result{}, err
		}
	}
	// Repeat inspection after potentially slow network/staging work. Explicit force
	// authorizes only the snapshot recorded immediately before switching.
	current, err := s.Inspect(req)
	if err == nil {
		err = inspectionConflict(req.Recipe, current, force)
	}
	if err != nil {
		return Result{}, errors.Join(err, s.discardStage(tx))
	}
	if !receiptsEqual(current.Receipt, in.Receipt) {
		return Result{}, errors.Join(errors.New("receipt changed during preparation"), s.discardStage(tx))
	}
	if in.Receipt != nil {
		if err := s.fault("before-final-scan"); err != nil {
			return Result{}, errors.Join(err, s.discardStage(tx))
		}
		scan, scanErr := scanTree(ctx, req.Root, in.Receipt.Files, in.Receipt.Directories)
		if scanErr != nil {
			return Result{}, errors.Join(scanErr, s.discardStage(tx))
		}
		if len(scan.types) > 0 {
			return Result{}, errors.Join(&ConflictError{Recipe: req.Recipe, Kind: "path-type", Paths: scan.types}, s.discardStage(tx))
		}
		final := Inspection{Receipt: in.Receipt, Changed: scan.changed, Missing: scan.missing, Unknown: scan.unknown}
		if err := inspectionConflict(req.Recipe, final, force); err != nil {
			return Result{}, errors.Join(err, s.discardStage(tx))
		}
		tx.Observed = scan.entries
		tx.RootExisted = scan.exists
		if tx.Operation == "metadata" && (len(scan.changed)+len(scan.missing)+len(scan.types)+len(scan.unknown) > 0) {
			return Result{}, &ConflictError{Recipe: req.Recipe, Kind: "owned-drift", Paths: append(scan.changed, scan.missing...)}
		}
	}
	if err := ctx.Err(); err != nil {
		return Result{}, errors.Join(err, s.discardStage(tx))
	}
	if err := s.write(tx); err != nil {
		if visible(err) {
			return Result{}, pendingError(req.Recipe, []string{req.Root}, err)
		}
		return Result{}, errors.Join(err, s.discardStage(tx))
	}
	err = s.transition(ctx, tx)
	if err != nil {
		if visible(err) {
			return Result{}, pendingError(req.Recipe, []string{req.Root}, err)
		}
		// Read the durable phase, never the in-memory next phase after a failed write.
		recorded, readErr := packagestate.ReadTransaction(s.Home, req.Recipe)
		if readErr != nil {
			return Result{}, errors.Join(err, ErrRecoveryRequired, readErr)
		}
		if recorded != nil && recorded.Phase == packagestate.Committed {
			return Result{Receipt: tx.Candidate, Mutated: true}, pendingError(req.Recipe, []string{req.Root}, err)
		}
		return Result{}, errors.Join(err, s.Recover(context.Background(), req.Recipe))
	}
	result := Result{Receipt: tx.Candidate, Mutated: true}
	cleanupCtx, cancel := newCleanupContext()
	defer cancel()
	if err := s.finishCommitted(cleanupCtx, tx); err != nil {
		return result, err
	}
	return result, nil
}
func (s *Service) discardStage(tx *packagestate.Transaction) error {
	if tx.Stage == "" {
		return nil
	}
	return cleanupTree(filepath.Dir(filepath.Dir(filepath.Dir(tx.Root))), tx.Recipe, tx.Stage, tx.Candidate.Files, tx.Candidate.Directories)
}
func (s *Service) transition(ctx context.Context, tx *packagestate.Transaction) error {
	home := filepath.Dir(filepath.Dir(filepath.Dir(tx.Root)))
	if err := s.fault("after-prepared"); err != nil {
		return err
	}
	if tx.Operation == "replace" {
		if err := ctx.Err(); err != nil {
			return err
		}
		tx.Intent = "move-old"
		if err := s.write(tx); err != nil {
			return err
		}
		if err := s.fault("after-intent-move-old"); err != nil {
			return err
		}
		if tx.RootExisted {
			scan, err := scanTree(ctx, tx.Root, tx.Observed, tx.Previous.Directories)
			if err != nil {
				return err
			}
			if !scan.exists {
				return &ConflictError{Recipe: tx.Recipe, Kind: "pending-recovery", Paths: []string{tx.Root}}
			}
			if err := scanConflict(tx.Recipe, tx.Root, scan, true); err != nil {
				return err
			}
			backup, err := exists(tx.Backup)
			if err != nil {
				return err
			}
			if backup {
				return &ConflictError{Recipe: tx.Recipe, Kind: "pending-recovery", Paths: []string{tx.Backup}}
			}
			if err := durableRename(home, tx.Root, tx.Backup); err != nil {
				return err
			}
		}
		if err := s.fault("after-backup-rename"); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		tx.Intent = "promote-new"
		if err := s.write(tx); err != nil {
			return err
		}
		if err := s.fault("after-intent-promote-new"); err != nil {
			return err
		}
		present, err := exists(tx.Root)
		if err != nil {
			return err
		}
		if present {
			return &ConflictError{Recipe: tx.Recipe, Kind: "pending-recovery", Paths: []string{tx.Root}}
		}
		if err := verifyCandidate(ctx, tx, tx.Stage); err != nil {
			return err
		}
		if err := durableRename(home, tx.Stage, tx.Root); err != nil {
			return err
		}
		if err := s.fault("after-stage-rename"); err != nil {
			return err
		}
		tx.Phase = packagestate.PackagePlaced
		tx.Intent = "save-receipt"
		if err := s.write(tx); err != nil {
			return err
		}
		if err := s.fault("after-package-placed"); err != nil {
			return err
		}
	} else {
		tx.Intent = "save-receipt"
		if err := s.write(tx); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.fault("before-receipt-save"); err != nil {
		return err
	}
	if err := verifyCandidate(ctx, tx, tx.Root); err != nil {
		return err
	}
	if err := s.save(tx.Candidate); err != nil {
		return err
	}
	if err := s.fault("after-receipt-save"); err != nil {
		return err
	}
	tx.Intent = "commit"
	if err := s.write(tx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.fault("before-commit-write"); err != nil {
		return err
	}
	if err := verifyCandidate(ctx, tx, tx.Root); err != nil {
		return err
	}
	tx.Phase = packagestate.Committed
	if err := s.write(tx); err != nil {
		return err
	}
	return s.fault("after-committed")
}

func verifyCandidate(ctx context.Context, tx *packagestate.Transaction, path string) error {
	scan, err := scanTree(ctx, path, tx.Candidate.Files, tx.Candidate.Directories)
	if err != nil {
		return err
	}
	if !scan.exists {
		return &ConflictError{Recipe: tx.Recipe, Kind: "pending-recovery", Paths: []string{path}}
	}
	return scanConflict(tx.Recipe, path, scan, true)
}

// A reduced removal receipt proves ownership, not completeness of the archive.
// Only hash-verified package.json can establish the complete no-download path.
func completeInventory(r *packagestate.Receipt) bool {
	var metadataEntry *packagebundle.Entry
	payload := make([]packagebundle.Entry, 0, len(r.Files))
	for i, e := range r.Files {
		if e.Path == "package.json" {
			metadataEntry = &r.Files[i]
		} else {
			payload = append(payload, e)
		}
	}
	if metadataEntry == nil {
		return false
	}
	f, err := os.Open(filepath.Join(r.Root, "package.json"))
	if err != nil {
		return false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, packagebundle.DefaultLimits.FileBytes+1))
	if err != nil || int64(len(data)) > packagebundle.DefaultLimits.FileBytes || bytesDigest(data) != metadataEntry.SHA256 {
		return false
	}
	var metadata packagebundle.Metadata
	if json.Unmarshal(data, &metadata) != nil || metadata.SchemaVersion != 1 || metadata.Identity != r.Identity {
		return false
	}
	return slices.Equal(payload, metadata.Files)
}
