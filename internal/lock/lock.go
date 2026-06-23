// Package lock writes patronus.lock — the L11 reproducibility artifact (DESIGN
// §5d, §5e, §6d). A lock pins everything a profile resolved to: each item's name,
// its `source` PROVENANCE (so a teammate or fresh machine refetches from the same
// origin), version, and a sha256 integrity anchor.
//
// The `source` field is present from v1 even though full git:/url: FETCHING lands
// in Phase 6 — retrofitting provenance into an already-shipped lock would be a
// breaking format change, so it is designed in now (in-tree items carry
// source: "registry").
//
// Format mirrors internal/state exactly: plain JSON via stdlib (deterministic,
// git-diffable, zero deps), atomic write, and a CLOCKLESS package — the caller
// supplies the timestamp so output is byte-stable in tests.
package lock

import (
	"encoding/json"
	"os"

	"github.com/darkquasar/patronus/internal/install"
)

// Version is the lock schema version, bumped when the on-disk shape changes.
// v2 (Phase 7): dropped the registry-wide RegistryVersion (the registry is no
// longer versioned as a whole — npm/pip model) and added per-entry TarballSha256.
const Version = 2

// Lock is the full patronus.lock document. Reproducibility is PER-ITEM: each
// entry pins its own version + sha, independent of the tool version and of any
// registry-wide version (there is none).
type Lock struct {
	Version   int     `json:"version"`
	Profile   string  `json:"profile,omitempty"`   // the profile this lock was generated from
	Generated string  `json:"generated,omitempty"` // RFC3339, caller-supplied (pkg stays clockless)
	Entries   []Entry `json:"entries"`
}

// Entry pins one resolved item with full provenance. Two shas serve distinct
// purposes: SHA256 is the manifest+content fold (drift detection — "did this
// item change?"); TarballSha256 is the digest of the item's published tarball
// BYTES, used to verify the exact artifact fetched from the registry's immutable
// name/version key (per-item reality-follows-lock).
type Entry struct {
	Name          string `json:"name"`
	Source        string `json:"source"`                  // "registry" for in-tree; canonical ref otherwise
	ResolvedRef   string `json:"resolvedRef,omitempty"`   // concrete commit a mutable ref resolved to
	Version       string `json:"version,omitempty"`       // the item's own version
	SHA256        string `json:"sha256"`                  // "sha256:" + hex over the manifest + content
	TarballSha256 string `json:"tarballSha256,omitempty"` // "sha256:" + hex over the published tarball bytes
	Slot          string `json:"slot,omitempty"`          // §1A layer it filled (informational)
	Kind          string `json:"kind,omitempty"`          // "artifact" | "recipe" | "plugin"
}

// Load reads a lock file, returning an empty lock if the file is absent.
func Load(path string) (*Lock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Lock{Version: Version}, nil
		}
		return nil, err
	}
	var l Lock
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, err
	}
	if l.Version == 0 {
		l.Version = Version
	}
	return &l, nil
}

// Save writes l atomically as indented, deterministic JSON.
func Save(path string, l *Lock) error {
	out, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return install.WriteFileAtomic(path, out, 0o644)
}
