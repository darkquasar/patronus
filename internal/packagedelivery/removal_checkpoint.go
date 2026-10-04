package packagedelivery

import (
	"context"
	"errors"

	"github.com/darkquasar/patronus/internal/packagestate"
)

// The schema-2 loop keeps inventories immutable. Each unlink has its own durable
// intent and completed-prefix checkpoint; the receipt is an outcome, not a ledger.
func (s *Service) finishCheckpointRemoval(ctx context.Context, tx *packagestate.Transaction) error {
	previous := tx.RemovalPrevious()
	if previous == nil {
		return pendingError(tx.Recipe, []string{tx.Root}, errors.New("missing validated removal manifest"))
	}
	current, err := packagestate.Load(s.Home, tx.Recipe)
	if err != nil {
		return pendingError(tx.Recipe, []string{tx.Root}, err)
	}
	_, pending := tx.NextRemoval()
	remaining := tx.RemovalReceipt()
	if tx.Phase == packagestate.Committed {
		if pending || !receiptsEqual(current, remaining) {
			return s.recoveryError(tx, errors.New("committed removal receipt mismatch"))
		}
		return nil
	}
	// Until final receipt intent, only the immutable previous receipt is valid.
	// After that intent, either side of the atomic receipt replacement is valid.
	finalIntent := tx.Intent == "save-receipt" || tx.Intent == "delete-receipt"
	if !receiptsEqual(current, previous) && (pending || !finalIntent || !receiptsEqual(current, remaining)) {
		return s.recoveryError(tx, errors.New("pending removal receipt mismatch"))
	}
	for {
		baseline, ok := tx.NextRemoval()
		if !ok {
			break
		}
		if err := ctx.Err(); err != nil {
			return s.recoveryError(tx, err)
		}
		tx.Intent = "unlink"
		if err := s.write(tx); err != nil {
			return pendingError(tx.Recipe, []string{baseline.Path}, err)
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
		if err := tx.CompleteRemoval(baseline.Path); err != nil {
			return s.recoveryError(tx, err)
		}
		if err := s.write(tx); err != nil {
			return pendingError(tx.Recipe, []string{baseline.Path}, err)
		}
		if err := s.fault("after-removal-progress"); err != nil {
			return s.recoveryError(tx, err)
		}
	}
	remaining = tx.RemovalReceipt()
	tx.Intent = "save-receipt"
	if remaining == nil {
		tx.Intent = "delete-receipt"
	}
	if err := s.write(tx); err != nil {
		return pendingError(tx.Recipe, []string{tx.Root}, err)
	}
	// A reopened final receipt need only be stabilized by Recover, not rewritten.
	if !receiptsEqual(current, remaining) {
		if remaining == nil {
			err = s.deleteReceipt(tx.Recipe)
		} else {
			err = s.save(remaining)
		}
		if err != nil {
			return pendingError(tx.Recipe, []string{tx.Root}, err)
		}
	}
	if err := s.fault("after-reduced-receipt-write"); err != nil {
		return s.recoveryError(tx, err)
	}
	if err := s.pruneRemoval(ctx, tx); err != nil {
		return s.recoveryError(tx, err)
	}
	tx.Phase, tx.Intent, tx.Detail = packagestate.Committed, "drop-reference", ""
	if err := s.write(tx); err != nil {
		return pendingError(tx.Recipe, []string{tx.Root}, err)
	}
	if err := s.fault("after-removal-commit"); err != nil {
		return pendingError(tx.Recipe, []string{tx.Root}, err)
	}
	return nil
}
