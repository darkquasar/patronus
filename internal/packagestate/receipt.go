// Package packagestate stores authoritative package ownership and recovery records.
// Mutating callers must hold the lock returned by Acquire.
package packagestate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/packagebundle"
)

// Receipt records the last committed ownership of a directory package.
type Receipt struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Recipe        string                 `json:"recipe"`
	RecipeVersion string                 `json:"recipeVersion"`
	Root          string                 `json:"root"`
	URL           string                 `json:"url"`
	ArchiveSHA256 string                 `json:"archiveSHA256"`
	Identity      packagebundle.Identity `json:"identity"`
	Files         []packagebundle.Entry  `json:"files"`
	Directories   []string               `json:"directories"`
}

// ErrBusy means another process holds the package mutation lock.
var ErrBusy = errors.New("package mutation is busy; retry after the other command finishes")

// ErrUnsupported means directory mutation is unavailable on this operating system.
var ErrUnsupported = errors.New("unsupported directory mutation on this operating system")

// DurabilityError reports a persistence failure. MayBeVisible is true after a
// rename or unlink: callers must inspect/recover, never assume the old state.
type DurabilityError struct {
	Path         string
	Stage        string
	MayBeVisible bool
	Err          error
}

func (e *DurabilityError) Error() string {
	return fmt.Sprintf("persist %s at %s (may be visible: %t): %v", e.Path, e.Stage, e.MayBeVisible, e.Err)
}
func (e *DurabilityError) Unwrap() error { return e.Err }

// storage provides instance-local fault injection for persistence boundary tests.
// Hooks surround temp Sync, rename/unlink, and parent Sync; production has none.
type storage struct {
	fault func(point, path string) error
}

func (s storage) check(point, path string) error {
	if s.fault != nil {
		return s.fault(point, path)
	}
	return nil
}

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*$`)
var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func canonicalHome(home string) (string, error) {
	if home == "" {
		return "", errors.New("home directory is required")
	}
	abs, err := filepath.Abs(home)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("home is not a directory: %s", real)
	}
	return real, nil
}
func safeRecipe(recipe string) error {
	if !namePattern.MatchString(recipe) || len(recipe) > 240 {
		return fmt.Errorf("invalid recipe name %q", recipe)
	}
	return nil
}

// directory walks from canonical home without traversing symlinks. Missing
// ancestors are allowed during reads. Newly created directory entries are synced.
func directory(home, rel string, create bool) (string, error) {
	current := home
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		parent := current
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if !create {
				continue
			}
			if err = os.Mkdir(current, 0700); err != nil && !os.IsExist(err) {
				return "", err
			}
			if err = syncDirectory(parent); err != nil {
				return "", &DurabilityError{Path: current, Stage: "create-parent-sync", MayBeVisible: true, Err: err}
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("unsupported symlink ancestor: %s", current)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("not a directory: %s", current)
		}
		if create && info.Mode().Perm() != 0700 {
			if err := os.Chmod(current, 0700); err != nil {
				return "", err
			}
			if err := syncDirectory(current); err != nil {
				return "", err
			}
		}
	}
	return current, nil
}
func paths(home, recipe string, create bool) (string, string, string, error) {
	if err := safeRecipe(recipe); err != nil {
		return "", "", "", err
	}
	h, err := canonicalHome(home)
	if err != nil {
		return "", "", "", err
	}
	root, err := directory(h, filepath.Join(".patronus", "packages", recipe), false)
	if err != nil {
		return "", "", "", err
	}
	state, err := directory(h, filepath.Join(".patronus", "package-state"), create)
	if err != nil {
		return "", "", "", err
	}
	return h, root, filepath.Join(state, recipe+".json"), nil
}
func regular(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("unsupported symlink leaf: %s", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular state file: %s", path)
	}
	return nil
}
func validateReceipt(r *Receipt, recipe, root string) error {
	if r == nil {
		return errors.New("nil receipt")
	}
	if r.SchemaVersion != 1 {
		return fmt.Errorf("unsupported receipt schema %d", r.SchemaVersion)
	}
	if r.Recipe != recipe || r.Root != root {
		return errors.New("receipt recipe/root does not match derived path")
	}
	if !manifest.ValidPackageVersion(r.RecipeVersion) || !namePattern.MatchString(r.Identity.Name) || !manifest.ValidPackageVersion(r.Identity.Version) {
		return errors.New("invalid receipt identity/version")
	}
	if (r.Identity.OS != "linux" && r.Identity.OS != "darwin" && r.Identity.OS != "windows") || (r.Identity.Arch != "amd64" && r.Identity.Arch != "arm64") {
		return errors.New("invalid receipt platform")
	}
	if !digestPattern.MatchString(r.ArchiveSHA256) {
		return errors.New("invalid archive digest")
	}
	if err := validateEntries(r.Files, false); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, dir := range r.Directories {
		if dir != "." {
			if _, err := packagebundle.ValidatePath(dir, false); err != nil {
				return err
			}
		}
		key := strings.ToLower(dir)
		if seen[key] {
			return fmt.Errorf("duplicate directory %s", dir)
		}
		seen[key] = true
	}
	names := make(map[string]string)
	files := make(map[string]bool)
	for _, f := range r.Files {
		files[strings.ToLower(f.Path)] = true
	}
	all := append([]string(nil), r.Directories...)
	for _, f := range r.Files {
		all = append(all, f.Path)
	}
	for _, p := range all {
		key := strings.ToLower(p)
		if seen[key] && files[key] {
			return fmt.Errorf("file/directory collision %s", p)
		}
		for current := p; current != "."; current = filepath.ToSlash(filepath.Dir(current)) {
			key := strings.ToLower(current)
			if old, ok := names[key]; ok && old != current {
				return fmt.Errorf("case-colliding record paths %s and %s", old, current)
			}
			names[key] = current
			if current != p && files[key] {
				return fmt.Errorf("file ancestor collision %s", current)
			}
		}
	}
	return nil
}
func validateEntries(entries []packagebundle.Entry, observed bool) error {
	seen := make(map[string]bool)
	for _, e := range entries {
		if _, err := packagebundle.ValidatePath(e.Path, false); err != nil {
			return err
		}
		validMode := e.Mode == 0644 || e.Mode == 0755
		if observed {
			validMode = e.Mode & ^uint32(07777) == 0
		}
		if !validMode || !digestPattern.MatchString(e.SHA256) {
			return fmt.Errorf("invalid entry mode/digest: %s", e.Path)
		}
		key := strings.ToLower(e.Path)
		if seen[key] {
			return fmt.Errorf("duplicate file %s", e.Path)
		}
		seen[key] = true
	}
	for key := range seen {
		for parent := filepath.ToSlash(filepath.Dir(key)); parent != "."; parent = filepath.ToSlash(filepath.Dir(parent)) {
			if seen[parent] {
				return fmt.Errorf("file ancestor collision %s", key)
			}
		}
	}
	return nil
}
func readJSON(path string, out any) (bool, error) {
	if err := regular(path); err != nil {
		return false, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return false, err
	}
	if info.Size() > 8<<20 {
		return false, errors.New("package state exceeds 8 MiB limit")
	}
	dec := json.NewDecoder(io.LimitReader(f, (8<<20)+1))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return false, err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("trailing JSON data")
		}
		return false, err
	}
	return true, nil
}

// Load returns nil, nil when no committed receipt exists. It never writes.
func Load(home, recipe string) (*Receipt, error) {
	_, root, path, err := paths(home, recipe, false)
	if err != nil {
		return nil, err
	}
	var r *Receipt
	found, err := readJSON(path, &r)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil //nolint:nilnil // Absence is the documented contract.
	}
	if err := validateReceipt(r, recipe, root); err != nil {
		return nil, err
	}
	return r, nil
}

// Save durably replaces a validated receipt; the candidate transaction is separate.
func Save(home string, r *Receipt) error { return (storage{}).save(home, r) }
func (s storage) save(home string, r *Receipt) error {
	if r == nil {
		return errors.New("nil receipt")
	}
	_, root, path, err := paths(home, r.Recipe, false)
	if err != nil {
		return err
	}
	if err := validateReceipt(r, r.Recipe, root); err != nil {
		return err
	}
	if _, _, _, err := paths(home, r.Recipe, true); err != nil {
		return err
	}
	return s.writeJSON(path, r)
}

// List returns committed receipts in recipe-name order. Transactions are excluded.
func List(home string) ([]Receipt, error) {
	h, err := canonicalHome(home)
	if err != nil {
		return nil, err
	}
	dir, err := directory(h, filepath.Join(".patronus", "package-state"), false)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Receipt
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		r, err := Load(h, strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		if r != nil {
			out = append(out, *r)
		}
	}
	return out, nil
}

// DeleteReceipt durably removes a receipt, including when it is already absent.
func DeleteReceipt(home, recipe string) error { return (storage{}).deleteReceipt(home, recipe) }
func (s storage) deleteReceipt(home, recipe string) error {
	_, _, path, err := paths(home, recipe, false)
	if err != nil {
		return err
	}
	return s.remove(path)
}
func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}
func (s storage) parentSync(path string) error {
	if err := s.check("before-parent-sync", path); err != nil {
		return err
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	return s.check("after-parent-sync", path)
}
func (s storage) writeJSON(path string, value any) (err error) {
	if err = regular(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if len(data) > 8<<20 {
		return errors.New("package state exceeds 8 MiB limit")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".state-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	visible := false
	stage := "write-temp"
	defer func() {
		if err != nil {
			err = &DurabilityError{Path: path, Stage: stage, MayBeVisible: visible, Err: err}
		}
	}()
	defer func() {
		closeErr := f.Close()
		if !errors.Is(closeErr, os.ErrClosed) {
			err = errors.Join(err, closeErr)
		}
		removeErr := os.Remove(temp)
		if !os.IsNotExist(removeErr) {
			err = errors.Join(err, removeErr)
		}
	}()
	if _, err = f.Write(data); err != nil {
		return err
	}
	stage = "before-temp-sync"
	if err = s.check(stage, path); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	stage = "after-temp-sync"
	if err = s.check(stage, path); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	stage = "before-rename"
	if err = s.check(stage, path); err != nil {
		return err
	}
	if err = regular(path); err != nil {
		return err
	}
	if err = os.Rename(temp, path); err != nil {
		return err
	}
	visible = true
	stage = "after-rename"
	if err = s.check(stage, path); err != nil {
		return err
	}
	stage = "parent-sync"
	return s.parentSync(path)
}
func (s storage) remove(path string) error {
	if err := regular(path); err != nil {
		return err
	}
	if _, err := os.Lstat(filepath.Dir(path)); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err := s.check("before-unlink", path); err != nil {
		return &DurabilityError{Path: path, Stage: "before-unlink", Err: err}
	}
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := s.check("after-unlink", path); err != nil {
		return &DurabilityError{Path: path, Stage: "after-unlink", MayBeVisible: true, Err: err}
	}
	if err := s.parentSync(path); err != nil {
		return &DurabilityError{Path: path, Stage: "parent-sync", MayBeVisible: true, Err: err}
	}
	return nil
}
