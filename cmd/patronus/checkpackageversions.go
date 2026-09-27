package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/packagebundle"
)

// checkPackageVersions compares shipped inputs at the actual merge-base and worktree.
func checkPackageVersions(ctx context.Context, root, base string) error {
	mergeBase, err := runGit(ctx, root, "merge-base", "--", base, "HEAD")
	if err != nil {
		return fmt.Errorf("package version check requires a readable merge-base: %w", err)
	}
	mergeBase = strings.TrimSpace(mergeBase)
	tree, err := runGitRaw(ctx, root, "ls-tree", "-r", "-z", mergeBase, "--", "packages/")
	if err != nil {
		return fmt.Errorf("package base tree: %w", err)
	}
	entries := make(map[string]string)
	for _, entry := range strings.Split(string(tree), "\x00") {
		if entry == "" {
			continue
		}
		header, name, ok := strings.Cut(entry, "\t")
		if !ok {
			return errors.New("invalid git tree entry")
		}
		fields := strings.Fields(header)
		if len(fields) != 3 {
			return errors.New("invalid git tree header")
		}
		entries[name] = fields[0]
	}
	dirs, err := os.ReadDir(filepath.Join(root, "packages"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		name := dir.Name()
		descriptorPath := "packages/" + name + "/package.yaml"
		if _, exists := entries[descriptorPath]; !exists {
			continue
		} // New packages have no previous release.
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(descriptorPath))); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		current, files, err := loadPackageSource(root, name)
		if err != nil {
			return fmt.Errorf("package %s: %w", name, err)
		}
		readBase := func(path string, limit int64) ([]byte, error) {
			if entries[path] != "100644" && entries[path] != "100755" {
				return nil, fmt.Errorf("base package path is missing or not regular: %s", path)
			}
			data, err := runGitRaw(ctx, root, "show", mergeBase+":"+path)
			if err != nil {
				return nil, err
			}
			if int64(len(data)) > limit {
				return nil, fmt.Errorf("base package file exceeds limit: %s", path)
			}
			return data, nil
		}
		data, err := readBase(descriptorPath, 1<<20)
		if err != nil {
			return err
		}
		previous, err := decodePackageDescriptor(data, name)
		if err != nil {
			return err
		}
		var oldFiles []packagebundle.File
		for _, p := range previous.Payload {
			data, err := readBase("packages/"+name+"/"+p.Path, packagebundle.DefaultLimits.FileBytes)
			if err != nil {
				return err
			}
			mode := uint32(0644)
			if p.Executable {
				mode = 0755
			}
			oldFiles = append(oldFiles, packagebundle.File{Path: p.Path, Mode: mode, Data: data})
		}
		oldVersion, newVersion := previous.Version, current.Version
		previous.Version = ""
		current.Version = ""
		oldDigest, err := packageSourceDigest(previous, oldFiles)
		if err != nil {
			return err
		}
		newDigest, err := packageSourceDigest(current, files)
		if err != nil {
			return err
		}
		if oldDigest != newDigest && oldVersion == newVersion {
			return fmt.Errorf("package %s changed shipped inputs without changing version %s", name, newVersion)
		}
	}
	return nil
}
