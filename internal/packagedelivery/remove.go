package packagedelivery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagestate"
)

// Remove deletes only recorded owned files. The caller holds Acquire and must
// repair discovery, then acknowledge the committed removal with ClearTransaction.
// Retained edits remain owned; Leftovers are unowned and never authorize deletion.
func (s *Service) Remove(ctx context.Context, recipe string, force bool) (Result, error) {
	if err := s.Recover(ctx, recipe); err != nil {
		return Result{}, err
	}
	pending, err := packagestate.ReadTransaction(s.Home, recipe)
	if err != nil {
		return Result{}, err
	}
	if pending != nil {
		return s.removalResult(ctx, pending)
	}
	root, err := s.root(recipe)
	if err != nil {
		return Result{}, err
	}
	receipt, err := packagestate.Load(s.Home, recipe)
	if err != nil {
		return Result{}, err
	}
	if receipt == nil {
		return Result{}, fmt.Errorf("package %s: missing receipt; no deletion authorized", recipe)
	}
	scan, err := scanTree(ctx, root, receipt.Files, receipt.Directories)
	if err != nil {
		return Result{}, err
	}
	if len(scan.types) > 0 {
		return Result{}, &ConflictError{Recipe: recipe, Kind: "path-type", Paths: scan.types}
	}
	tx := &packagestate.Transaction{SchemaVersion: 1, Recipe: recipe, Root: root, Operation: "remove", Phase: packagestate.Prepared, Previous: receipt, Removed: slices.Clone(scan.missing)}
	changed := make(map[string]bool, len(scan.changed))
	for _, path := range scan.changed {
		changed[path] = true
	}
	for _, entry := range scan.entries {
		if !force && changed[entry.Path] {
			continue
		}
		tx.Observed = append(tx.Observed, entry)
		tx.PendingRemove = append(tx.PendingRemove, entry.Path)
	}
	if err := s.fault("before-removal-manifest"); err != nil {
		return Result{}, err
	}
	if err := packagestate.PrepareRemoval(s.Home, tx); err != nil {
		return Result{}, pendingError(recipe, []string{root}, err)
	}
	if err := s.fault("after-removal-manifest"); err != nil {
		return Result{}, pendingError(recipe, []string{root}, err)
	}
	if err := s.write(tx); err != nil {
		return Result{}, pendingError(recipe, []string{root}, err)
	}
	if err := s.fault("after-removal-publication"); err != nil {
		return Result{}, pendingError(recipe, []string{root}, err)
	}
	if err := s.finishRemoval(ctx, tx); err != nil {
		return Result{}, err
	}
	return s.removalResult(ctx, tx)
}

func reducedRemovalReceipt(tx *packagestate.Transaction) *packagestate.Receipt {
	if tx.SchemaVersion == 2 {
		return tx.RemovalReceipt()
	}
	removed := make(map[string]bool, len(tx.Removed))
	for _, path := range tx.Removed {
		removed[path] = true
	}
	r := *tx.Previous
	r.Files = nil
	r.Directories = slices.Clone(tx.Previous.Directories)
	for _, entry := range tx.Previous.Files {
		if !removed[entry.Path] {
			r.Files = append(r.Files, entry)
		}
	}
	if len(r.Files) == 0 {
		return nil
	}
	return &r
}

// validateRemoval establishes ownership as well as syntax. A valid journal may
// only reduce its previous receipt, and observed baselines never grant ownership.
func validateRemoval(tx *packagestate.Transaction, current *packagestate.Receipt) error {
	if tx.Previous == nil || tx.Candidate != nil || tx.Stage != "" || tx.Backup != "" || (tx.Phase != packagestate.Prepared && tx.Phase != packagestate.Committed) {
		return errors.New("invalid removal transaction")
	}
	owned := make(map[string]packagebundle.Entry, len(tx.Previous.Files))
	for _, e := range tx.Previous.Files {
		owned[e.Path] = e
	}
	observed := make(map[string]bool, len(tx.Observed))
	for _, e := range tx.Observed {
		if _, ok := owned[e.Path]; !ok {
			return fmt.Errorf("removal observation is not owned: %s", e.Path)
		}
		observed[e.Path] = true
	}
	for _, p := range tx.PendingRemove {
		if !observed[p] || slices.Contains(tx.Removed, p) {
			return fmt.Errorf("invalid removal intent: %s", p)
		}
	}
	for _, p := range tx.Removed {
		if _, ok := owned[p]; !ok {
			return fmt.Errorf("removed path is not owned: %s", p)
		}
	}
	for p := range observed {
		if !slices.Contains(tx.PendingRemove, p) && !slices.Contains(tx.Removed, p) {
			return fmt.Errorf("unaccounted removal observation: %s", p)
		}
	}
	if tx.Phase == packagestate.Committed {
		if len(tx.PendingRemove) > 0 || !receiptsEqual(current, reducedRemovalReceipt(tx)) {
			return errors.New("committed removal receipt mismatch")
		}
		return nil
	}
	if current == nil {
		if len(tx.Removed) != len(owned) {
			return errors.New("removal receipt missing before all owned paths completed")
		}
		return nil
	}
	// Storage may have stopped after writing Removed but before saving its reduced receipt.
	expected := *tx.Previous
	expected.Files = slices.Clone(current.Files)
	if !receiptsEqual(current, &expected) {
		return errors.New("removal receipt metadata changed")
	}
	present := make(map[string]bool, len(current.Files))
	for _, e := range current.Files {
		if original, ok := owned[e.Path]; !ok || original != e {
			return fmt.Errorf("removal receipt changed: %s", e.Path)
		}
		present[e.Path] = true
	}
	for p := range owned {
		if !present[p] && !slices.Contains(tx.Removed, p) {
			return fmt.Errorf("unrecorded receipt reduction: %s", p)
		}
	}
	return nil
}

func (s *Service) finishRemoval(ctx context.Context, tx *packagestate.Transaction) error {
	if tx.SchemaVersion == 2 {
		return s.finishCheckpointRemoval(ctx, tx)
	}
	current, err := packagestate.Load(s.Home, tx.Recipe)
	if err != nil {
		return pendingError(tx.Recipe, []string{tx.Root}, err)
	}
	if err := validateRemoval(tx, current); err != nil {
		return s.recoveryError(tx, err)
	}
	if tx.Phase == packagestate.Committed {
		return nil
	}
	for len(tx.PendingRemove) > 0 {
		if err := ctx.Err(); err != nil {
			return s.recoveryError(tx, err)
		}
		name := tx.PendingRemove[0]
		var baseline packagebundle.Entry
		for _, e := range tx.Observed {
			if e.Path == name {
				baseline = e
				break
			}
		}
		tx.Intent = "unlink"
		if err := s.write(tx); err != nil {
			return pendingError(tx.Recipe, []string{name}, err)
		}
		if err := s.fault("after-unlink-intent"); err != nil {
			return s.recoveryError(tx, err)
		}
		if err := s.unlinkOwned(ctx, tx, baseline); err != nil {
			return s.recoveryError(tx, err)
		}
		if err := s.fault("after-unlink"); err != nil {
			return s.recoveryError(tx, err)
		}
		tx.Removed = append(tx.Removed, name)
		tx.PendingRemove = tx.PendingRemove[1:]
		tx.Intent = "save-receipt"
		if err := s.write(tx); err != nil {
			return pendingError(tx.Recipe, []string{name}, err)
		}
		if err := s.saveRemovalReceipt(tx); err != nil {
			return err
		}
	}
	if err := s.saveRemovalReceipt(tx); err != nil {
		return err
	}
	if err := s.pruneRemoval(ctx, tx); err != nil {
		return s.recoveryError(tx, err)
	}
	tx.Phase = packagestate.Committed
	tx.Intent = "drop-reference"
	tx.Detail = ""
	if err := s.write(tx); err != nil {
		return pendingError(tx.Recipe, []string{tx.Root}, err)
	}
	if err := s.fault("after-removal-commit"); err != nil {
		return pendingError(tx.Recipe, []string{tx.Root}, err)
	}
	return nil
}

func (s *Service) saveRemovalReceipt(tx *packagestate.Transaction) error {
	receipt := reducedRemovalReceipt(tx)
	tx.Intent = "save-receipt"
	if receipt == nil {
		tx.Intent = "delete-receipt"
	}
	if err := s.write(tx); err != nil {
		return pendingError(tx.Recipe, []string{tx.Root}, err)
	}
	var err error
	if receipt == nil {
		err = s.deleteReceipt(tx.Recipe)
	} else {
		err = s.save(receipt)
	}
	if err != nil {
		return pendingError(tx.Recipe, []string{tx.Root}, err)
	}
	if err := s.fault("after-reduced-receipt-write"); err != nil {
		return s.recoveryError(tx, err)
	}
	return nil
}

func (s *Service) unlinkOwned(ctx context.Context, tx *packagestate.Transaction, baseline packagebundle.Entry) error {
	path := filepath.Join(tx.Root, filepath.FromSlash(baseline.Path))
	home := filepath.Dir(filepath.Dir(filepath.Dir(tx.Root)))
	if err := checkAncestry(home, filepath.Dir(path)); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return syncExistingParent(home, filepath.Dir(path))
	}
	if err != nil {
		return err
	}
	conflict := &ConflictError{Recipe: tx.Recipe, Kind: "pending-recovery", Paths: []string{path}}
	if !info.Mode().IsRegular() || unixMode(info.Mode()) != baseline.Mode {
		return conflict
	}
	hash, err := digestFile(ctx, path)
	if err != nil {
		return err
	}
	if hash != baseline.SHA256 {
		return conflict
	}
	if err := checkAncestry(home, filepath.Dir(path)); err != nil {
		return err
	}
	latest, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !latest.Mode().IsRegular() || !os.SameFile(info, latest) || unixMode(latest.Mode()) != baseline.Mode {
		return conflict
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if err := s.fault("after-unlink-before-parent-sync"); err != nil {
		return &packagestate.DurabilityError{Path: path, Stage: "unlink-parent-sync", MayBeVisible: true, Err: err}
	}
	if err := syncPath(filepath.Dir(path)); err != nil {
		return &packagestate.DurabilityError{Path: path, Stage: "unlink-parent-sync", MayBeVisible: true, Err: err}
	}
	return nil
}

func syncExistingParent(home, path string) error {
	for {
		if _, err := os.Lstat(path); err == nil {
			return syncPath(path)
		} else if !os.IsNotExist(err) {
			return err
		}
		if path == home {
			return errors.New("home disappeared during removal")
		}
		path = filepath.Dir(path)
	}
}

func (s *Service) pruneRemoval(ctx context.Context, tx *packagestate.Transaction) error {
	previous := tx.Previous
	if tx.SchemaVersion == 2 {
		previous = tx.RemovalPrevious()
	}
	dirs := slices.Clone(previous.Directories)
	slices.Sort(dirs)
	home := filepath.Dir(filepath.Dir(filepath.Dir(tx.Root)))
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := filepath.Join(tx.Root, filepath.FromSlash(dirs[i]))
		if err := checkAncestry(home, path); err != nil {
			return err
		}
		err := os.Remove(path)
		if os.IsNotExist(err) || errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST) {
			continue
		}
		if err != nil {
			return err
		}
		if err := syncPath(filepath.Dir(path)); err != nil {
			return &packagestate.DurabilityError{Path: path, Stage: "rmdir-parent-sync", MayBeVisible: true, Err: err}
		}
	}
	return nil
}

func (s *Service) removalResult(ctx context.Context, tx *packagestate.Transaction) (Result, error) {
	receipt := reducedRemovalReceipt(tx)
	result := Result{Receipt: receipt, Mutated: true}
	if receipt != nil {
		for _, e := range receipt.Files {
			result.Retained = append(result.Retained, e.Path)
		}
	}
	var files []packagebundle.Entry
	if receipt != nil {
		files = receipt.Files
	}
	scan, err := scanTree(ctx, tx.Root, files, tx.Previous.Directories)
	result.Leftovers = scan.unknown
	return result, err
}
