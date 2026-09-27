package scan

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagedelivery"
	"github.com/darkquasar/patronus/internal/packagestate"
)

// PackageStatus reports authoritative package ownership and pending recovery.
type PackageStatus struct {
	Recipe  string   `json:"recipe"`
	Version string   `json:"version"`
	Root    string   `json:"root"`
	Status  string   `json:"status"`
	Paths   []string `json:"paths"`
}

// Packages discovers receipts and pending first installs independently of tools.
// It never repairs references, resumes transactions, or creates state directories.
func Packages(home string) ([]PackageStatus, error) {
	receipts, err := packagestate.List(home)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]*packagestate.Receipt, len(receipts))
	names := make(map[string]bool, len(receipts))
	for i := range receipts {
		byName[receipts[i].Recipe] = &receipts[i]
		names[receipts[i].Recipe] = true
	}
	canonical, err := filepath.EvalSymlinks(home)
	if err != nil {
		return nil, err
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(canonical, ".patronus", "package-state", "transactions")
	// List validates state ancestry. Validate the additional transaction component
	// before enumerating it so discovery never traverses a substituted symlink.
	info, err := os.Lstat(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("package transactions is not a real directory: %s", dir)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			tx, err := packagestate.ReadTransaction(canonical, entry.Name())
			if err != nil {
				return nil, err
			}
			if tx != nil {
				names[tx.Recipe] = true
			}
		}
	}
	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}
	slices.Sort(ordered)
	var result []PackageStatus
	service := packagedelivery.Service{Home: canonical}
	for _, name := range ordered {
		tx, err := packagestate.ReadTransaction(canonical, name)
		if err != nil {
			return nil, err
		}
		receipt := byName[name]
		status := PackageStatus{Recipe: name, Root: filepath.Join(canonical, ".patronus", "packages", name), Status: "installed"}
		if receipt != nil {
			status.Version = receipt.RecipeVersion
		}
		if tx != nil {
			status.Status = "recovery-required"
			status.Paths = append(slices.Clone(tx.PendingRemove), tx.Root)
			if receipt == nil && tx.Candidate != nil {
				status.Version = tx.Candidate.RecipeVersion
			}
			if receipt == nil && tx.Previous != nil {
				status.Version = tx.Previous.RecipeVersion
			}
			result = append(result, status)
			continue
		}
		req := packagedelivery.Request{Recipe: name, RecipeVersion: receipt.RecipeVersion, Root: receipt.Root, URL: receipt.URL, SHA256: receipt.ArchiveSHA256, Identity: receipt.Identity}
		in, err := service.Inspect(req)
		var conflict *packagedelivery.ConflictError
		if err != nil && !errors.As(err, &conflict) {
			return nil, err
		}
		status.Paths = append(status.Paths, in.Changed...)
		status.Paths = append(status.Paths, in.Unknown...)
		if conflict != nil {
			status.Paths = append(status.Paths, conflict.Paths...)
		}
		if len(status.Paths) > 0 {
			status.Status = "modified"
		}
		hasMetadata := false
		for _, entry := range receipt.Files {
			if entry.Path == "package.json" {
				hasMetadata = true
			}
		}
		if len(in.Missing) > 0 || !hasMetadata {
			status.Status = "incomplete"
			status.Paths = append(status.Paths, in.Missing...)
			if !hasMetadata {
				status.Paths = append(status.Paths, "package.json")
			}
		}
		if hasMetadata && conflict == nil && !slices.Contains(in.Changed, "package.json") && !slices.Contains(in.Missing, "package.json") {
			if missing := missingPackageOwnership(receipt); len(missing) > 0 {
				status.Status = "incomplete"
				status.Paths = append(status.Paths, missing...)
			}
		}
		slices.Sort(status.Paths)
		status.Paths = slices.Compact(status.Paths)
		result = append(result, status)
	}
	return result, nil
}

// A reduced receipt can retain package.json. Compare its verified inventory so
// absent ownership does not masquerade as a complete installation.
func missingPackageOwnership(receipt *packagestate.Receipt) []string {
	f, err := os.Open(filepath.Join(receipt.Root, "package.json"))
	if err != nil {
		return []string{"package.json"}
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, packagebundle.DefaultLimits.FileBytes+1))
	if err != nil || int64(len(data)) > packagebundle.DefaultLimits.FileBytes {
		return []string{"package.json"}
	}
	owned := make(map[string]packagebundle.Entry, len(receipt.Files))
	for _, entry := range receipt.Files {
		owned[entry.Path] = entry
	}
	if fmt.Sprintf("sha256:%x", sha256.Sum256(data)) != owned["package.json"].SHA256 {
		return []string{"package.json"}
	}
	var metadata packagebundle.Metadata
	if json.Unmarshal(data, &metadata) != nil || metadata.SchemaVersion != 1 || metadata.Identity != receipt.Identity {
		return []string{"package.json"}
	}
	var missing []string
	for _, entry := range metadata.Files {
		if recorded, ok := owned[entry.Path]; !ok || recorded != entry {
			missing = append(missing, entry.Path)
		}
	}
	return missing
}
