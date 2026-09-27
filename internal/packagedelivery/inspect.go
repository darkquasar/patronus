// Package packagedelivery installs verified static trees without executing them.
package packagedelivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagestate"
)

// Request pins a package and its fixed installation destination.
type Request struct {
	Recipe        string                 `json:"recipe"`
	RecipeVersion string                 `json:"recipeVersion"`
	URL           string                 `json:"url"`
	SHA256        string                 `json:"sha256"`
	Root          string                 `json:"root"`
	Identity      packagebundle.Identity `json:"identity"`
}

// Inspection describes committed ownership and current drift without writing.
type Inspection struct {
	Receipt *packagestate.Receipt     `json:"receipt"`
	Changed []string                  `json:"changed"`
	Missing []string                  `json:"missing"`
	Unknown []string                  `json:"unknown"`
	Pending *packagestate.Transaction `json:"pending"`
}

// Result describes the committed outcome, including retryable removal leftovers.
type Result struct {
	Receipt   *packagestate.Receipt `json:"receipt"`
	Mutated   bool                  `json:"mutated"`
	Retained  []string              `json:"retained"`
	Leftovers []string              `json:"leftovers"`
}

// Fetcher opens a package stream with cancellation.
type Fetcher interface {
	Open(context.Context, string) (io.ReadCloser, error)
}

// Service performs package operations while its caller holds packagestate.Acquire.
type Service struct {
	Home    string
	Fetcher Fetcher
	// Fault is nil in production; tests return errors or exit a subprocess.
	Fault       func(point string) error
	persistence *persistence
}

// ConflictError identifies paths requiring user reconciliation.
type ConflictError struct {
	Recipe string   `json:"recipe"`
	Kind   string   `json:"kind"`
	Paths  []string `json:"paths"`
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("package %s: %s: %s", e.Recipe, e.Kind, strings.Join(e.Paths, ", "))
}

// ErrRecoveryRequired means transaction evidence remains for a later recovery.
var ErrRecoveryRequired = errors.New("package recovery required")

func (s *Service) root(recipe string) (string, error) {
	if len(recipe) > 240 || !regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*$`).MatchString(recipe) {
		return "", errors.New("invalid recipe name")
	}
	home, err := filepath.Abs(s.Home)
	if err != nil {
		return "", err
	}
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, ".patronus", "packages", recipe)
	if err = checkAncestry(home, root); err != nil {
		return "", &ConflictError{Recipe: recipe, Kind: "path-type", Paths: []string{root}}
	}
	return root, nil
}
func checkAncestry(base, path string) error {
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("path escapes home")
	}
	current := base
	for _, p := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, p)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("not a real directory: %s", current)
		}
	}
	return nil
}
func (s *Service) normalize(req Request) (Request, error) {
	root, err := s.root(req.Recipe)
	if err != nil {
		return req, err
	}
	if req.Root != root {
		return req, errors.New("request root does not match fixed package root")
	}
	digest := strings.ToLower(strings.TrimPrefix(req.SHA256, "sha256:"))
	raw, err := hex.DecodeString(digest)
	if err != nil || len(raw) != sha256.Size {
		return req, errors.New("invalid archive digest")
	}
	req.SHA256 = "sha256:" + digest
	u, err := url.Parse(req.URL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return req, errors.New("package URL must be HTTPS without credentials")
	}
	if !manifest.ValidPackageVersion(req.RecipeVersion) || !manifest.ValidPackageVersion(req.Identity.Version) || !regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*$`).MatchString(req.Identity.Name) {
		return req, errors.New("invalid package identity/version")
	}
	if (req.Identity.OS != "linux" && req.Identity.OS != "darwin" && req.Identity.OS != "windows") || (req.Identity.Arch != "amd64" && req.Identity.Arch != "arm64") {
		return req, errors.New("invalid package platform")
	}
	return req, nil
}

// Inspect hashes owned regular files, enumerates unknown paths and reports pending work.
func (s *Service) Inspect(req Request) (Inspection, error) {
	req, err := s.normalize(req)
	if err != nil {
		return Inspection{}, err
	}
	in := Inspection{}
	in.Pending, err = packagestate.ReadTransaction(s.Home, req.Recipe)
	if err != nil {
		return in, err
	}
	in.Receipt, err = packagestate.Load(s.Home, req.Recipe)
	if err != nil {
		return in, err
	}
	if in.Pending != nil {
		return in, nil
	}
	if in.Receipt == nil {
		if _, err := os.Lstat(req.Root); err == nil {
			return in, &ConflictError{Recipe: req.Recipe, Kind: "unowned-root", Paths: []string{req.Root}}
		} else if !os.IsNotExist(err) {
			return in, err
		}
		return in, nil
	}
	scan, err := scanTree(context.Background(), req.Root, in.Receipt.Files, in.Receipt.Directories)
	in.Changed, in.Missing, in.Unknown = scan.changed, scan.missing, scan.unknown
	if err != nil {
		return in, err
	}
	if len(scan.types) > 0 {
		return in, &ConflictError{Recipe: req.Recipe, Kind: "path-type", Paths: scan.types}
	}
	return in, nil
}

type treeScan struct {
	entries                          []packagebundle.Entry
	changed, missing, unknown, types []string
	exists                           bool
}

func unixMode(mode fs.FileMode) uint32 {
	n := uint32(mode.Perm())
	if mode&os.ModeSetuid != 0 {
		n |= 04000
	}
	if mode&os.ModeSetgid != 0 {
		n |= 02000
	}
	if mode&os.ModeSticky != 0 {
		n |= 01000
	}
	return n
}
func digestFile(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	buf := make([]byte, 64<<10)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := f.Read(buf)
		if n > 0 {
			_, _ = h.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
func ownedDirs(files []packagebundle.Entry, dirs []string) []string {
	set := map[string]bool{".": true}
	for _, d := range dirs {
		set[d] = true
	}
	for _, f := range files {
		for d := filepath.ToSlash(filepath.Dir(f.Path)); d != "."; d = filepath.ToSlash(filepath.Dir(d)) {
			set[d] = true
		}
	}
	result := make([]string, 0, len(set))
	for d := range set {
		result = append(result, d)
	}
	slices.Sort(result)
	return result
}
func scanTree(ctx context.Context, root string, files []packagebundle.Entry, dirs []string) (treeScan, error) {
	var result treeScan
	expected := make(map[string]packagebundle.Entry, len(files))
	for _, e := range files {
		expected[e.Path] = e
	}
	expectedDirs := make(map[string]bool)
	for _, d := range ownedDirs(files, dirs) {
		expectedDirs[d] = true
	}
	seen := make(map[string]bool)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if os.IsNotExist(err) && path == root {
			return nil
		}
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		seen[rel] = true
		if rel == "." {
			result.exists = true
		}
		e, isFile := expected[rel]
		isDir := expectedDirs[rel]
		if !isFile && !isDir {
			result.unknown = append(result.unknown, rel)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if isDir {
			if !info.IsDir() {
				result.types = append(result.types, rel)
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			result.types = append(result.types, rel)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		hash, err := digestFile(ctx, path)
		if err != nil {
			return err
		}
		observed := packagebundle.Entry{Path: rel, Mode: unixMode(info.Mode()), SHA256: hash}
		result.entries = append(result.entries, observed)
		if observed != e {
			result.changed = append(result.changed, rel)
		}
		return nil
	})
	for _, e := range files {
		if !seen[e.Path] {
			result.missing = append(result.missing, e.Path)
		}
	}
	slices.SortFunc(result.entries, func(a, b packagebundle.Entry) int { return strings.Compare(a.Path, b.Path) })
	return result, err
}
func inspectionConflict(recipe string, in Inspection, force bool) error {
	if in.Pending != nil {
		return errors.Join(ErrRecoveryRequired, &ConflictError{Recipe: recipe, Kind: "pending-recovery", Paths: []string{in.Pending.Root}})
	}
	if len(in.Unknown) > 0 {
		return &ConflictError{Recipe: recipe, Kind: "unknown-content", Paths: in.Unknown}
	}
	if len(in.Changed) > 0 && !force {
		return &ConflictError{Recipe: recipe, Kind: "owned-drift", Paths: in.Changed}
	}
	return nil
}
func bytesDigest(data []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(data)) }
