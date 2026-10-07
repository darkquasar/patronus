// ADR-0003 folder validation, authored for Patronus. No model or network calls.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func validate(folder string) error {
	data, err := os.ReadFile(filepath.Join(folder, "meta.yaml"))
	if err != nil {
		return fmt.Errorf("read meta.yaml: %w", err)
	}
	var meta struct {
		Research string `yaml:"research"`
		Streams  []struct {
			Spec string `yaml:"spec"`
			Plan string `yaml:"plan"`
		} `yaml:"streams"`
	}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("parse meta.yaml: %w", err)
	}
	refs := []string{meta.Research}
	for _, stream := range meta.Streams {
		refs = append(refs, stream.Spec, stream.Plan)
	}
	named := make(map[string]bool)
	for _, name := range refs {
		if name == "" {
			continue // Null or absent references are unfinished documents, not flags.
		}
		if filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) || !strings.HasSuffix(name, ".md") {
			return fmt.Errorf("invalid document filename: %s", name)
		}
		info, err := os.Lstat(filepath.Join(folder, name))
		if err != nil {
			return fmt.Errorf("missing referenced file: %s: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("referenced file is not regular: %s", name)
		}
		named[name] = true
	}
	entries, err := os.ReadDir(folder)
	if err != nil {
		return fmt.Errorf("read folder: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, "-spec.md") || strings.HasSuffix(name, "-plan.md") {
			if !named[name] {
				return fmt.Errorf("orphan spec/plan file: %s", name)
			}
		}
	}
	return nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: adr0003-validator <spec-folder>")
		os.Exit(2)
	}
	if err := validate(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("ADR-0003 OK")
}
