package packagedelivery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagestate"
)

func receiptsEqual(a, b *packagestate.Receipt) bool { return reflect.DeepEqual(a, b) }
func exists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
func scanConflict(recipe, root string, scan treeScan, requireAll bool) error {
	paths := append(slices.Clone(scan.changed), scan.types...)
	paths = append(paths, scan.unknown...)
	if requireAll {
		paths = append(paths, scan.missing...)
	}
	if len(paths) == 0 {
		return nil
	}
	for i := range paths {
		paths[i] = filepath.Join(root, paths[i])
	}
	slices.Sort(paths)
	return &ConflictError{Recipe: recipe, Kind: "pending-recovery", Paths: slices.Compact(paths)}
}

// removeTree removes only matching recorded files, then empty owned directories.
// Validate the entire tree before unlinking so an initial conflict preserves it.
func removeTree(ctx context.Context, home, recipe, root string, files []packagebundle.Entry, dirs []string) error {
	if root == "" {
		return nil
	}
	if err := checkAncestry(home, root); err != nil {
		return err
	}
	scan, err := scanTree(ctx, root, files, dirs)
	if err != nil {
		return err
	}
	if err := scanConflict(recipe, root, scan, false); err != nil {
		return err
	}
	if !scan.exists {
		return nil
	}
	for _, entry := range scan.entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := filepath.Join(root, filepath.FromSlash(entry.Path))
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || unixMode(info.Mode()) != entry.Mode {
			return &ConflictError{Recipe: recipe, Kind: "pending-recovery", Paths: []string{path}}
		}
		hash, err := digestFile(ctx, path)
		if err != nil {
			return err
		}
		if hash != entry.SHA256 {
			return &ConflictError{Recipe: recipe, Kind: "pending-recovery", Paths: []string{path}}
		}
		if err := checkAncestry(home, filepath.Dir(path)); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		if err := syncPath(filepath.Dir(path)); err != nil {
			return &packagestate.DurabilityError{Path: path, Stage: "unlink-parent-sync", MayBeVisible: true, Err: err}
		}
	}
	directories := ownedDirs(files, dirs)
	for i := len(directories) - 1; i >= 0; i-- {
		path := filepath.Join(root, filepath.FromSlash(directories[i]))
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return &ConflictError{Recipe: recipe, Kind: "pending-recovery", Paths: []string{path}}
		}
		if err := checkAncestry(home, filepath.Dir(path)); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return errors.Join(err, &ConflictError{Recipe: recipe, Kind: "pending-recovery", Paths: []string{path}})
		}
		if err := syncPath(filepath.Dir(path)); err != nil {
			return &packagestate.DurabilityError{Path: path, Stage: "rmdir-parent-sync", MayBeVisible: true, Err: err}
		}
	}
	return nil
}
func pendingError(recipe string, paths []string, cause error) error {
	var conflict *ConflictError
	if errors.As(cause, &conflict) {
		return errors.Join(ErrRecoveryRequired, cause)
	}
	return errors.Join(ErrRecoveryRequired, &ConflictError{Recipe: recipe, Kind: "pending-recovery", Paths: paths}, cause)
}
func (s *Service) recoveryError(tx *packagestate.Transaction, cause error) error {
	if visible(cause) {
		return pendingError(tx.Recipe, []string{tx.Root}, cause)
	}
	if tx.Phase != packagestate.RecoveryRequired {
		tx.ResumePhase = tx.Phase
		tx.Phase = packagestate.RecoveryRequired
	}
	tx.Detail = cause.Error()
	return pendingError(tx.Recipe, []string{tx.Root}, errors.Join(cause, s.write(tx)))
}
func (s *Service) stabilize(path string) error {
	if s.persistence != nil && s.persistence.stabilize != nil {
		return s.persistence.stabilize(path)
	}
	if err := syncPath(path); err != nil {
		return err
	}
	return syncPath(filepath.Dir(path))
}

// Recover resolves only this recipe's transaction under the caller-held lock.
// Once begun, bounded recovery ignores cancellation to leave coherent evidence.
func (s *Service) Recover(ctx context.Context, recipe string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := s.root(recipe)
	if err != nil {
		return pendingError(recipe, []string{recipe}, err)
	}
	tx, err := packagestate.ReadTransaction(s.Home, recipe)
	if err != nil {
		return pendingError(recipe, []string{recipe}, err)
	}
	if tx == nil {
		return nil
	}
	home := filepath.Dir(filepath.Dir(filepath.Dir(root)))
	journal := filepath.Join(home, ".patronus", "package-state", "transactions", recipe, "transaction.json")
	// Reopening validated records and syncing their links resolves MayBeVisible.
	for _, path := range []string{filepath.Dir(journal), filepath.Dir(root), filepath.Dir(tx.Stage)} {
		if path == "." {
			continue
		}
		if err := syncAncestors(home, path); err != nil {
			return pendingError(recipe, []string{path}, err)
		}
	}
	if err := s.stabilize(journal); err != nil {
		return pendingError(recipe, []string{journal}, &packagestate.DurabilityError{Path: journal, Stage: "recovery-sync", MayBeVisible: true, Err: err})
	}
	receipt, err := packagestate.Load(s.Home, recipe)
	if err != nil {
		return pendingError(recipe, []string{recipe}, err)
	}
	if receipt != nil {
		if err := s.stabilize(filepath.Join(home, ".patronus", "package-state", recipe+".json")); err != nil {
			return pendingError(recipe, []string{recipe}, err)
		}
	}
	if tx.Phase == packagestate.RecoveryRequired {
		tx.Phase = tx.ResumePhase
		tx.ResumePhase = ""
	}
	cleanupCtx, cancel := newCleanupContext()
	defer cancel()
	if tx.Operation == "remove" {
		return s.finishRemoval(cleanupCtx, tx)
	}
	if tx.Phase == packagestate.Committed {
		return s.finishCommitted(cleanupCtx, tx)
	}
	if err := s.rollback(cleanupCtx, tx); err != nil {
		return s.recoveryError(tx, err)
	}
	return nil
}
func (s *Service) rollback(ctx context.Context, tx *packagestate.Transaction) error {
	home := filepath.Dir(filepath.Dir(filepath.Dir(tx.Root)))
	current, err := packagestate.Load(s.Home, tx.Recipe)
	if err != nil {
		return err
	}
	if !receiptsEqual(current, tx.Previous) && !receiptsEqual(current, tx.Candidate) {
		return &ConflictError{Recipe: tx.Recipe, Kind: "pending-recovery", Paths: []string{tx.Recipe + ".json"}}
	}

	if tx.Operation == "replace" {
		// Validate every participant before removing a promoted candidate: an
		// unsafe backup must not leave the active tree needlessly absent.
		for _, path := range []string{tx.Root, tx.Stage, tx.Backup} {
			if err := checkAncestry(home, path); err != nil {
				return err
			}
		}
		backup, err := exists(tx.Backup)
		if err != nil {
			return err
		}
		root, err := exists(tx.Root)
		if err != nil {
			return err
		}
		var oldDirs []string
		if tx.Previous != nil {
			oldDirs = tx.Previous.Directories
		}
		if backup {
			scan, err := scanTree(ctx, tx.Backup, tx.Observed, oldDirs)
			if err != nil {
				return err
			}
			if err := scanConflict(tx.Recipe, tx.Backup, scan, true); err != nil {
				return err
			}
			if !tx.RootExisted {
				return errors.New("unexpected backup for initially absent root")
			}
			if root {
				stage, err := exists(tx.Stage)
				if err != nil {
					return err
				}
				// A still-present stage proves this root was not our promotion,
				// even if an unknown root happens to have identical bytes.
				if stage {
					return &ConflictError{Recipe: tx.Recipe, Kind: "pending-recovery", Paths: []string{tx.Root}}
				}
				if err := removeTree(ctx, home, tx.Recipe, tx.Root, tx.Candidate.Files, tx.Candidate.Directories); err != nil {
					return err
				}
			}
			if err := durableRename(home, tx.Backup, tx.Root); err != nil {
				return err
			}
		} else if tx.RootExisted {
			// Either move-old never happened or rollback already restored the preimage.
			if !root {
				return &ConflictError{Recipe: tx.Recipe, Kind: "pending-recovery", Paths: []string{tx.Root, tx.Backup}}
			}
			scan, err := scanTree(ctx, tx.Root, tx.Observed, oldDirs)
			if err != nil {
				return err
			}
			if err := scanConflict(tx.Recipe, tx.Root, scan, true); err != nil {
				return err
			}
		} else if root {
			// If stage still exists, this root cannot be our promoted candidate.
			stage, err := exists(tx.Stage)
			if err != nil {
				return err
			}
			if stage {
				return &ConflictError{Recipe: tx.Recipe, Kind: "pending-recovery", Paths: []string{tx.Root}}
			}
			if err := removeTree(ctx, home, tx.Recipe, tx.Root, tx.Candidate.Files, tx.Candidate.Directories); err != nil {
				return err
			}
		}
		if err := s.discardStage(tx); err != nil {
			return err
		}
	}
	if tx.Previous != nil {
		if err := s.save(tx.Previous); err != nil {
			return err
		}
	} else {
		if err := s.deleteReceipt(tx.Recipe); err != nil {
			return err
		}
	}
	return s.clear(tx.Recipe)
}
func (s *Service) finishCommitted(ctx context.Context, tx *packagestate.Transaction) error {
	home := filepath.Dir(filepath.Dir(filepath.Dir(tx.Root)))
	// Never re-save a candidate over an unrelated receipt or modify the active tree.
	current, err := packagestate.Load(s.Home, tx.Recipe)
	if err != nil {
		return s.recoveryError(tx, err)
	}
	if !receiptsEqual(current, tx.Candidate) {
		return s.recoveryError(tx, &ConflictError{Recipe: tx.Recipe, Kind: "pending-recovery", Paths: []string{tx.Recipe + ".json"}})
	}
	if tx.Operation == "replace" {
		tx.Intent = "cleanup"
		if err := s.write(tx); err != nil {
			return pendingError(tx.Recipe, []string{tx.Root}, err)
		}
		if err := s.fault("during-backup-cleanup"); err != nil {
			return pendingError(tx.Recipe, []string{tx.Root}, err)
		}
		var dirs []string
		if tx.Previous != nil {
			dirs = tx.Previous.Directories
		}
		if err := removeTree(ctx, home, tx.Recipe, tx.Backup, tx.Observed, dirs); err != nil {
			return s.recoveryError(tx, err)
		}
		if err := s.discardStage(tx); err != nil {
			return s.recoveryError(tx, err)
		}
	}
	if err := s.clear(tx.Recipe); err != nil {
		return pendingError(tx.Recipe, []string{tx.Root}, fmt.Errorf("clear completed transaction: %w", err))
	}
	return nil
}

// Sync existing ancestor links without creating paths during recovery.
func syncAncestors(home, path string) error {
	if err := checkAncestry(home, path); err != nil {
		return err
	}
	rel, err := filepath.Rel(home, path)
	if err != nil {
		return err
	}
	current := home
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		child := filepath.Join(current, part)
		present, err := exists(child)
		if err != nil {
			return err
		}
		if !present {
			break
		}
		if err := syncPath(current); err != nil {
			return &packagestate.DurabilityError{Path: child, Stage: "ancestor-sync", MayBeVisible: true, Err: err}
		}
		current = child
	}
	return nil
}

func newCleanupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Minute)
}
func cleanupTree(home, recipe, root string, files []packagebundle.Entry, dirs []string) error {
	ctx, cancel := newCleanupContext()
	defer cancel()
	return removeTree(ctx, home, recipe, root, files, dirs)
}
