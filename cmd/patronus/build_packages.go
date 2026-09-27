package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/registry"
)

type builtPackage struct {
	Identity packagebundle.Identity `json:"identity"`
	Key      string                 `json:"key"`
	SHA256   string                 `json:"sha256"`
}
type packagePlatform struct {
	OS   string `yaml:"os" json:"os"`
	Arch string `yaml:"arch" json:"arch"`
}
type packagePayload struct {
	Path       string `yaml:"path" json:"path"`
	Executable bool   `yaml:"executable" json:"executable"`
}
type packageDescriptor struct {
	SchemaVersion int               `yaml:"schemaVersion" json:"schemaVersion"`
	Name          string            `yaml:"name" json:"name"`
	Version       string            `yaml:"version" json:"version"`
	Platforms     []packagePlatform `yaml:"platforms" json:"platforms"`
	Payload       []packagePayload  `yaml:"payload" json:"payload"`
}

func decodePackageDescriptor(data []byte, name string) (packageDescriptor, error) {
	var d packageDescriptor
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&d); err != nil {
		return d, fmt.Errorf("package descriptor: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return d, errors.New("package descriptor contains trailing document")
	}
	if d.SchemaVersion != 1 || d.Name != name || !regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*$`).MatchString(name) || !manifest.ValidPackageVersion(d.Version) {
		return d, errors.New("invalid package descriptor schema or identity")
	}
	if len(d.Platforms) == 0 || len(d.Payload) == 0 || len(d.Payload) >= packagebundle.DefaultLimits.Entries {
		return d, errors.New("package requires bounded payload and platforms")
	}
	seen := make(map[packagePlatform]bool)
	for _, p := range d.Platforms {
		if seen[p] || (p.OS != "darwin" && p.OS != "linux" && p.OS != "windows") || (p.Arch != "arm64" && p.Arch != "amd64") {
			return d, errors.New("invalid or duplicate package platform")
		}
		seen[p] = true
	}
	paths := make(map[string]bool)
	for _, p := range d.Payload {
		if _, err := packagebundle.ValidatePath(p.Path, false); err != nil {
			return d, err
		}
		if p.Path == "package.json" || paths[strings.ToLower(p.Path)] {
			return d, fmt.Errorf("reserved or duplicate payload %q", p.Path)
		}
		paths[strings.ToLower(p.Path)] = true
	}
	sort.Slice(d.Platforms, func(i, j int) bool {
		a, b := d.Platforms[i], d.Platforms[j]
		return a.OS < b.OS || a.OS == b.OS && a.Arch < b.Arch
	})
	sort.Slice(d.Payload, func(i, j int) bool { return d.Payload[i].Path < d.Payload[j].Path })
	return d, nil
}

// readPackageFile rejects links at every source component and bounds allocation.
func readPackageFile(root, relative string, limit int64) ([]byte, error) {
	if _, err := packagebundle.ValidatePath(relative, false); err != nil {
		return nil, err
	}
	current := root
	parts := strings.Split(relative, "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("package source link %q", relative)
		}
		if i < len(parts)-1 {
			if !info.IsDir() {
				return nil, fmt.Errorf("package source ancestor is not directory: %q", relative)
			}
			continue
		}
		if !info.Mode().IsRegular() || info.Size() > limit {
			return nil, fmt.Errorf("package source is not a bounded regular file: %q", relative)
		}
	}
	sourceRoot, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer sourceRoot.Close()
	file, err := sourceRoot.Open(relative)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("package source exceeds limit: %q", relative)
	}
	return data, nil
}
func loadPackageSource(root, name string) (packageDescriptor, []packagebundle.File, error) {
	var empty packageDescriptor
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*$`).MatchString(name) {
		return empty, nil, errors.New("invalid package name")
	}
	prefix := "packages/" + name + "/"
	data, err := readPackageFile(root, prefix+"package.yaml", 1<<20)
	if err != nil {
		return empty, nil, err
	}
	d, err := decodePackageDescriptor(data, name)
	if err != nil {
		return d, nil, err
	}
	var files []packagebundle.File
	var total int64
	for _, p := range d.Payload {
		data, err := readPackageFile(root, prefix+p.Path, packagebundle.DefaultLimits.FileBytes)
		if err != nil {
			return d, nil, err
		}
		total += int64(len(data))
		if total > packagebundle.DefaultLimits.ExpandedBytes {
			return d, nil, errors.New("package expanded size exceeded")
		}
		mode := uint32(0644)
		if p.Executable {
			mode = 0755
		}
		files = append(files, packagebundle.File{Path: p.Path, Mode: mode, Data: data})
	}
	return d, files, nil
}

func packageSourceDigest(d packageDescriptor, files []packagebundle.File) (string, error) {
	canonical, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	// Every variable-length field is framed; source bytes cannot alias another layout.
	write := func(b []byte) {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(b)))
		h.Write(size[:])
		h.Write(b)
	}
	write(canonical)
	sorted := append([]packagebundle.File(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	for _, f := range sorted {
		write([]byte(f.Path))
		var mode [4]byte
		binary.BigEndian.PutUint32(mode[:], f.Mode)
		write(mode[:])
		write(f.Data)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
func buildPackage(root, name, out string) ([]builtPackage, error) {
	d, files, err := loadPackageSource(root, name)
	if err != nil {
		return nil, err
	}
	source, err := packageSourceDigest(d, files)
	if err != nil {
		return nil, err
	}
	// Provenance is intentionally outside the archive and its source digest.
	commit, err := runGit(context.Background(), root, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("package provenance commit: %w", err)
	}
	provenance, err := json.Marshal(struct {
		SchemaVersion    int    `json:"schemaVersion"`
		RepositoryCommit string `json:"repositoryCommit"`
		CIRun            string `json:"ciRun,omitempty"`
	}{1, strings.TrimSpace(commit), os.Getenv("GITHUB_RUN_ID")})
	if err != nil {
		return nil, err
	}
	var built []builtPackage
	for _, p := range d.Platforms {
		id := packagebundle.Identity{Name: d.Name, Version: d.Version, OS: p.OS, Arch: p.Arch}
		data, err := packagebundle.Build(id, source, files)
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf("packages/%s/%s/%s-%s-%s-%s.tar.gz", d.Name, d.Version, d.Name, d.Version, p.OS, p.Arch)
		sum := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
		for _, file := range []struct {
			suffix string
			data   []byte
		}{{"", data}, {".sha256", []byte(sum + "\n")}, {".provenance.json", provenance}} {
			if err := install.WriteFileAtomic(filepath.Join(out, filepath.FromSlash(key+file.suffix)), file.data, 0644); err != nil {
				return nil, err
			}
		}
		built = append(built, builtPackage{Identity: id, Key: key, SHA256: sum})
	}
	return built, nil
}
func buildReferencedPackages(root, out string, cat *registry.Catalog) ([]builtPackage, error) {
	seen := make(map[string]bool)
	var built []builtPackage
	for _, r := range cat.Recipes {
		d := r.Manifest.Delivery
		if d == nil || d.Unpack != "directory" {
			continue
		}
		if d.Package == nil {
			return nil, fmt.Errorf("recipe %s has no package identity", r.Manifest.Name)
		}
		if seen[d.Package.Name] {
			continue
		}
		seen[d.Package.Name] = true
		items, err := buildPackage(root, d.Package.Name, out)
		if err != nil {
			return nil, fmt.Errorf("package %s: %w", d.Package.Name, err)
		}
		built = append(built, items...)
	}
	return built, nil
}
func verifyPackagePins(cat *registry.Catalog, built []builtPackage, baseURL string) error {
	packages := make(map[packagebundle.Identity]builtPackage, len(built))
	for _, p := range built {
		packages[p.Identity] = p
	}
	for _, r := range cat.Recipes {
		d := r.Manifest.Delivery
		if d == nil || d.Unpack != "directory" {
			continue
		}
		if err := manifest.ValidateRecipe(r.Manifest); err != nil {
			return err
		}
		for _, a := range d.Assets {
			id := packagebundle.Identity{Name: d.Package.Name, Version: d.Package.Version, OS: a.OS, Arch: a.Arch}
			p, ok := packages[id]
			if !ok {
				return fmt.Errorf("recipe %s: package identity/platform not built: %+v", r.Manifest.Name, id)
			}
			if a.URL != strings.TrimRight(baseURL, "/")+"/"+p.Key || "sha256:"+strings.ToLower(strings.TrimPrefix(a.SHA256, "sha256:")) != p.SHA256 {
				return fmt.Errorf("recipe %s: package URL or checksum pin does not match %s", r.Manifest.Name, p.Key)
			}
		}
	}
	return nil
}
