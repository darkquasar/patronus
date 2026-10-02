package packagestate

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/darkquasar/patronus/internal/packagebundle"
)

// removalManifest is written before transaction.json, once per removal. Only the
// latter authorizes replay: an orphan manifest cannot authorize an unlink.
type removalManifest struct {
	SchemaVersion int                   `json:"schemaVersion"`
	ID            string                `json:"id"`
	Recipe        string                `json:"recipe"`
	Root          string                `json:"root"`
	Previous      *Receipt              `json:"previous"`
	Missing       []string              `json:"missing"`
	Selected      []packagebundle.Entry `json:"selected"`
}

type removalCheckpoint struct {
	SchemaVersion int    `json:"schemaVersion"`
	Recipe        string `json:"recipe"`
	Root          string `json:"root"`
	Operation     string `json:"operation"`
	RemovalID     string `json:"removalID"`
	ManifestHash  string `json:"manifestHash"`
	Completed     int    `json:"completed"`
	Phase         Phase  `json:"phase"`
	ResumePhase   Phase  `json:"resumePhase"`
	Intent        string `json:"intent"`
	Detail        string `json:"detail"`
}

// Immutable inventories are private and detached from the exported reader view.
// Checkpoint writes never validate, marshal, or traverse those inventories.
type removalState struct {
	manifest  removalManifest
	hash      string
	completed int
}

func cloneReceipt(r *Receipt) *Receipt {
	if r == nil {
		return nil
	}
	out := *r
	out.Files = slices.Clone(r.Files)
	out.Directories = slices.Clone(r.Directories)
	return &out
}

func manifestHash(m *removalManifest) (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data)), nil
}

func validateRemovalManifest(m *removalManifest, recipe, root string) error {
	if m == nil || m.SchemaVersion != 2 || m.Recipe != recipe || m.Root != root {
		return errors.New("invalid removal manifest identity")
	}
	id, err := hex.DecodeString(m.ID)
	if err != nil || len(id) != 32 || hex.EncodeToString(id) != m.ID {
		return errors.New("invalid removal transaction ID")
	}
	if err := validateReceipt(m.Previous, recipe, root); err != nil {
		return err
	}
	if err := validateEntries(m.Selected, true); err != nil {
		return err
	}
	owned := make(map[string]bool, len(m.Previous.Files))
	for _, e := range m.Previous.Files {
		owned[e.Path] = true
	}
	seen := make(map[string]bool, len(m.Missing)+len(m.Selected))
	for _, p := range m.Missing {
		if !owned[p] || seen[p] {
			return fmt.Errorf("invalid initially missing removal path: %s", p)
		}
		seen[p] = true
	}
	for i, e := range m.Selected {
		if !owned[e.Path] || seen[e.Path] || i > 0 && m.Selected[i-1].Path >= e.Path {
			return fmt.Errorf("invalid selected removal path: %s", e.Path)
		}
		seen[e.Path] = true
	}
	return nil
}

// PrepareRemoval freezes validated ownership and observed preimages. The caller
// must next WriteTransaction before deleting anything, and holds Acquire throughout.
// A manifest without a journal is inert (pre-publication or post-acknowledgement).
func PrepareRemoval(home string, tx *Transaction) error {
	return (storage{}).prepareRemoval(home, tx)
}

func (s storage) prepareRemoval(home string, tx *Transaction) error {
	if tx == nil {
		return errors.New("nil removal transaction")
	}
	h, root, path, err := transactionPath(home, tx.Recipe)
	if err != nil {
		return err
	}
	if err := validateTransaction(h, tx.Recipe, root, tx); err != nil {
		return err
	}
	if tx.Operation != "remove" || tx.Phase != Prepared || tx.Intent != "" || tx.Candidate != nil || tx.Stage != "" || tx.Backup != "" || tx.RootExisted {
		return errors.New("invalid initial removal transaction")
	}
	pending, err := ReadTransaction(home, tx.Recipe)
	if err != nil {
		return err
	}
	if pending != nil {
		return errors.New("transaction already exists")
	}
	m := removalManifest{SchemaVersion: 2, Recipe: tx.Recipe, Root: root, Previous: cloneReceipt(tx.Previous), Missing: slices.Clone(tx.Removed), Selected: slices.Clone(tx.Observed)}
	if len(tx.PendingRemove) != len(m.Selected) {
		return errors.New("removal selection/observation mismatch")
	}
	for i, e := range m.Selected {
		if tx.PendingRemove[i] != e.Path {
			return errors.New("removal selection order mismatch")
		}
	}
	var id [32]byte
	if _, err := rand.Read(id[:]); err != nil {
		return err
	}
	m.ID = hex.EncodeToString(id[:])
	if err := validateRemovalManifest(&m, tx.Recipe, root); err != nil {
		return err
	}
	hash, err := manifestHash(&m)
	if err != nil {
		return err
	}
	if _, err := s.directory(h, filepath.Join(".patronus", "package-state", "transactions", tx.Recipe), true); err != nil {
		return err
	}
	if err := s.writeJSON(filepath.Join(filepath.Dir(path), "removal.json"), &m); err != nil {
		return err
	}
	tx.SchemaVersion = 2
	tx.RemovalID, tx.ManifestHash = m.ID, hash
	tx.removal = &removalState{manifest: m, hash: hash}
	return nil
}

func validateRemovalCheckpoint(c removalCheckpoint, state *removalState) error {
	m := &state.manifest
	if c.SchemaVersion != 2 || c.Operation != "remove" || c.Recipe != m.Recipe || c.Root != m.Root || c.RemovalID != m.ID || c.ManifestHash != state.hash {
		return errors.New("removal checkpoint/manifest identity mismatch")
	}
	if c.Completed < 0 || c.Completed > len(m.Selected) || len(c.Detail) > 1024 {
		return errors.New("invalid removal progress bounds")
	}
	phase := c.Phase
	if phase == RecoveryRequired {
		phase = c.ResumePhase
	} else if c.ResumePhase != "" {
		return errors.New("invalid removal resume phase")
	}
	if phase != Prepared && phase != Committed {
		return errors.New("invalid removal phase")
	}
	switch c.Intent {
	case "":
		if c.Completed != 0 || phase != Prepared {
			return errors.New("invalid initial removal progress")
		}
	case "unlink":
		if c.Completed == len(m.Selected) || phase != Prepared {
			return errors.New("invalid unlink progress")
		}
	case "progress":
		if c.Completed == 0 || phase != Prepared {
			return errors.New("invalid completed removal progress")
		}
	case "save-receipt", "delete-receipt":
		if c.Completed != len(m.Selected) || phase != Prepared {
			return errors.New("premature removal receipt")
		}
	case "drop-reference":
		if c.Completed != len(m.Selected) || phase != Committed {
			return errors.New("premature removal commit")
		}
	default:
		return errors.New("invalid removal intent")
	}
	return nil
}

func (tx *Transaction) removalCheckpoint() removalCheckpoint {
	return removalCheckpoint{SchemaVersion: tx.SchemaVersion, Recipe: tx.Recipe, Root: tx.Root, Operation: tx.Operation, RemovalID: tx.RemovalID, ManifestHash: tx.ManifestHash, Completed: tx.Completed, Phase: tx.Phase, ResumePhase: tx.ResumePhase, Intent: tx.Intent, Detail: tx.Detail}
}

func readRemoval(path string, tx *Transaction) (*Transaction, error) {
	if tx.Previous != nil || tx.Candidate != nil || tx.RootExisted || tx.Stage != "" || tx.Backup != "" || len(tx.Observed)+len(tx.Removed)+len(tx.PendingRemove) != 0 {
		return nil, errors.New("inventories in removal checkpoint")
	}
	var m *removalManifest
	found, err := readJSON(filepath.Join(filepath.Dir(path), "removal.json"), &m)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("removal manifest missing")
	}
	if err := validateRemovalManifest(m, tx.Recipe, tx.Root); err != nil {
		return nil, err
	}
	hash, err := manifestHash(m)
	if err != nil {
		return nil, err
	}
	state := &removalState{manifest: *m, hash: hash, completed: tx.Completed}
	if err := validateRemovalCheckpoint(tx.removalCheckpoint(), state); err != nil {
		return nil, err
	}
	tx.removal = state
	// Materialize only for full readers. These slices never drive checkpoint writes.
	tx.Previous = cloneReceipt(m.Previous)
	tx.Observed = slices.Clone(m.Selected)
	tx.Removed = slices.Clone(m.Missing)
	for i, e := range m.Selected {
		if i < state.completed {
			tx.Removed = append(tx.Removed, e.Path)
		} else {
			tx.PendingRemove = append(tx.PendingRemove, e.Path)
		}
	}
	return tx, nil
}

func (s storage) writeRemoval(path string, tx *Transaction) error {
	if tx.removal == nil || tx.Completed != tx.removal.completed {
		return errors.New("removal checkpoint requires validated private state")
	}
	c := tx.removalCheckpoint()
	// Error text is diagnostic only; bound the on-disk checkpoint independently of N.
	c.Detail = strings.ToValidUTF8(c.Detail, "?")
	if len(c.Detail) > 1024 {
		c.Detail = c.Detail[:1024]
		for !utf8.ValidString(c.Detail) {
			c.Detail = c.Detail[:len(c.Detail)-1]
		}
	}
	if err := validateRemovalCheckpoint(c, tx.removal); err != nil {
		return err
	}
	return s.writeJSON(path, c)
}

// NextRemoval returns a value from the privately validated immutable selection.
func (tx *Transaction) NextRemoval() (packagebundle.Entry, bool) {
	if tx.removal == nil || tx.removal.completed == len(tx.removal.manifest.Selected) {
		return packagebundle.Entry{}, false
	}
	return tx.removal.manifest.Selected[tx.removal.completed], true
}

// CompleteRemoval advances only the next selected path. WriteTransaction must
// durably record this before attempting another unlink. On failure stop and reopen.
func (tx *Transaction) CompleteRemoval(path string) error {
	e, ok := tx.NextRemoval()
	if !ok || e.Path != path || tx.Intent != "unlink" {
		return errors.New("out-of-order removal completion")
	}
	tx.removal.completed++
	tx.Completed = tx.removal.completed
	tx.Intent = "progress"
	return nil
}

// RemovalPrevious returns a detached ownership snapshot, not the reader's mutable view.
func (tx *Transaction) RemovalPrevious() *Receipt {
	if tx.removal == nil {
		return nil
	}
	return cloneReceipt(tx.removal.manifest.Previous)
}

// RemovalReceipt materializes the completed-prefix ownership view in linear time.
// Call only at open/finalization boundaries, never once per file.
func (tx *Transaction) RemovalReceipt() *Receipt {
	if tx.removal == nil {
		return nil
	}
	m := &tx.removal.manifest
	removed := make(map[string]bool, len(m.Missing)+tx.removal.completed)
	for _, p := range m.Missing {
		removed[p] = true
	}
	for _, e := range m.Selected[:tx.removal.completed] {
		removed[e.Path] = true
	}
	r := cloneReceipt(m.Previous)
	r.Files = nil
	for _, e := range m.Previous.Files {
		if !removed[e.Path] {
			r.Files = append(r.Files, e)
		}
	}
	if len(r.Files) == 0 {
		return nil
	}
	return r
}
