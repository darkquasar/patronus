package packagestate

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/packagebundle"
)

// Phase identifies a durable transaction checkpoint.
type Phase string

// Transaction phases retain the committed receipt independently of the candidate.
const (
	Prepared         Phase = "prepared"
	PackagePlaced    Phase = "package-placed"
	Committed        Phase = "committed"
	RecoveryRequired Phase = "recovery-required"
)

// Transaction is the write-ahead record for replacement, metadata, or removal.
type Transaction struct {
	SchemaVersion int      `json:"schemaVersion"`
	Recipe        string   `json:"recipe"`
	Root          string   `json:"root"`
	Stage         string   `json:"stage"`
	Backup        string   `json:"backup"`
	Phase         Phase    `json:"phase"`
	ResumePhase   Phase    `json:"resumePhase"`
	Operation     string   `json:"operation"`
	Intent        string   `json:"intent"`
	Previous      *Receipt `json:"previous"`
	Candidate     *Receipt `json:"candidate"`
	// Observed modes encode Unix permissions plus setuid/setgid/sticky (0..07777),
	// never Go FileMode type bits. These preimage modes need not be normalized.
	Observed      []packagebundle.Entry `json:"observed"`
	Removed       []string              `json:"removed"`
	PendingRemove []string              `json:"pendingRemove"`
	Detail        string                `json:"detail"`
}

func transactionPath(home, recipe string, create bool) (string, string, string, error) {
	h, root, _, err := paths(home, recipe, create)
	if err != nil {
		return "", "", "", err
	}
	dir, err := directory(h, filepath.Join(".patronus", "package-state", "transactions", recipe), create)
	if err != nil {
		return "", "", "", err
	}
	return h, root, filepath.Join(dir, "transaction.json"), nil
}
func validPhase(p Phase) bool { return p == Prepared || p == PackagePlaced || p == Committed }
func validateTransaction(home, recipe, root string, tx *Transaction) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	if tx.SchemaVersion != 1 {
		return fmt.Errorf("unsupported transaction schema %d", tx.SchemaVersion)
	}
	if tx.Recipe != recipe || tx.Root != root {
		return errors.New("transaction recipe/root does not match derived path")
	}
	if tx.Phase == RecoveryRequired {
		if !validPhase(tx.ResumePhase) {
			return errors.New("invalid recovery resume phase")
		}
	} else if !validPhase(tx.Phase) || tx.ResumePhase != "" {
		return errors.New("invalid transaction phase")
	}
	intents := map[string]string{"replace": " move-old promote-new save-receipt commit cleanup ", "metadata": " save-receipt commit ", "remove": " unlink save-receipt delete-receipt drop-reference commit cleanup "}
	allowed, ok := intents[tx.Operation]
	if !ok || tx.Intent != "" && !strings.Contains(allowed, " "+tx.Intent+" ") {
		return errors.New("invalid transaction operation/intent")
	}
	if tx.Operation == "replace" && (tx.Stage == "" || tx.Backup == "" || tx.Stage == tx.Backup) {
		return errors.New("replacement requires distinct stage and backup paths")
	}
	for _, p := range []string{tx.Stage, tx.Backup} {
		if p == "" {
			continue
		}
		base := filepath.Join(home, ".patronus", "packages", ".txn", recipe)
		if filepath.Dir(p) != base || filepath.Clean(p) != p || !namePattern.MatchString(filepath.Base(p)) {
			return fmt.Errorf("transaction stage/backup escapes reserved recipe directory: %s", p)
		}
		rel, err := filepath.Rel(home, p)
		if err != nil {
			return err
		}
		if _, err := directory(home, rel, false); err != nil {
			return err
		}
		if err := sameFilesystem(root, p); err != nil {
			return err
		}
	}
	for _, r := range []*Receipt{tx.Previous, tx.Candidate} {
		if r != nil {
			if err := validateReceipt(r, recipe, root); err != nil {
				return err
			}
		}
	}
	if tx.Operation != "remove" && tx.Candidate == nil {
		return errors.New("transaction requires candidate receipt")
	}
	if err := validateEntries(tx.Observed, true); err != nil {
		return err
	}
	for _, paths := range [][]string{tx.Removed, tx.PendingRemove} {
		seen := make(map[string]bool)
		for _, p := range paths {
			if _, err := packagebundle.ValidatePath(p, false); err != nil {
				return err
			}
			if seen[p] {
				return fmt.Errorf("duplicate removal path %s", p)
			}
			seen[p] = true
		}
	}
	return nil
}

// ReadTransaction reads without recovering or changing filesystem state.
func ReadTransaction(home, recipe string) (*Transaction, error) {
	h, root, path, err := transactionPath(home, recipe, false)
	if err != nil {
		return nil, err
	}
	var tx *Transaction
	found, err := readJSON(path, &tx)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil //nolint:nilnil // Absence is the documented contract.
	}
	if err := validateTransaction(h, recipe, root, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

// WriteTransaction durably records validated recovery intent.
func WriteTransaction(home string, tx *Transaction) error {
	return (storage{}).writeTransaction(home, tx)
}
func (s storage) writeTransaction(home string, tx *Transaction) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	h, root, path, err := transactionPath(home, tx.Recipe, false)
	if err != nil {
		return err
	}
	if err := validateTransaction(h, tx.Recipe, root, tx); err != nil {
		return err
	}
	if _, _, _, err := transactionPath(h, tx.Recipe, true); err != nil {
		return err
	}
	return s.writeJSON(path, tx)
}

// ClearTransaction durably deletes the journal, leaving its private directory.
func ClearTransaction(home, recipe string) error { return (storage{}).clearTransaction(home, recipe) }
func (s storage) clearTransaction(home, recipe string) error {
	_, _, path, err := transactionPath(home, recipe, false)
	if err != nil {
		return err
	}
	return s.remove(path)
}
