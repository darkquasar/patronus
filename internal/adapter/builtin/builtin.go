// Package builtin embeds the canonical adapter definitions into the binary so an
// installed Patronus (which has no source checkout) can still transform artifacts
// for each tool. The embedded copies are the SAME yaml authored at the repo root
// `adapters/`; an equality test (builtin_test.go) fails CI if the two ever drift,
// so the root remains the single authoring source and the embed stays in lockstep
// with the adapter engine it ships alongside.
package builtin

import (
	"embed"
	"io/fs"
	"sort"

	"github.com/darkquasar/patronus/internal/manifest"
)

//go:embed *.yaml
var files embed.FS

// Adapters parses every embedded adapter definition, keyed nowhere — returned as
// a slice in stable (tool-name) order, matching what loadAdapters returns from a
// checkout's adapters/ dir.
func Adapters() ([]*manifest.Adapter, error) {
	entries, err := fs.Glob(files, "*.yaml")
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)
	out := make([]*manifest.Adapter, 0, len(entries))
	for _, name := range entries {
		data, err := files.ReadFile(name)
		if err != nil {
			return nil, err
		}
		ad, err := manifest.DecodeAdapter(data)
		if err != nil {
			return nil, err
		}
		out = append(out, ad)
	}
	return out, nil
}
