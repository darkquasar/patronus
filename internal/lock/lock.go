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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/nativepi"
)

// Version is the legacy generation/default schema, intentionally unchanged for
// non-Pi locks. Explicit Pi profile generation uses TargetVersion.
// v2 (Phase 7): dropped the registry-wide RegistryVersion (the registry is no
// longer versioned as a whole — npm/pip model) and added per-entry TarballSha256.
const Version = 2

// TargetVersion adds a required explicit target; old v2 readers reject it.
const TargetVersion = 3
const NativeVersion = 4

// Plugin reconciliation statuses (Entry.Status, Kind=="plugin" only).
const (
	StatusVerified   = "verified"
	StatusUnverified = "unverified"
	StatusMissing    = "missing"
)

// Lock is the full patronus.lock document. Reproducibility is PER-ITEM: each
// entry pins its own version + sha, independent of the tool version and of any
// registry-wide version (there is none).
type Lock struct {
	Version   int     `json:"version"`
	Target    string  `json:"target,omitempty"`
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
	NativeSource  string `json:"nativeSource,omitempty"`
	Name          string `json:"name"`
	Source        string `json:"source"`                  // "registry" for in-tree; canonical ref otherwise
	ResolvedRef   string `json:"resolvedRef,omitempty"`   // concrete commit a mutable ref resolved to
	Version       string `json:"version,omitempty"`       // the item's own version
	SHA256        string `json:"sha256"`                  // "sha256:" + hex over the manifest + content
	TarballSha256 string `json:"tarballSha256,omitempty"` // "sha256:" + hex over the published tarball bytes
	Slot          string `json:"slot,omitempty"`          // §1A layer it filled (informational)
	Kind          string `json:"kind,omitempty"`          // "artifact" | "recipe" | "plugin"

	// Delivery pins directory payloads; legacy entries omit it.
	Delivery *manifest.Delivery `json:"delivery,omitempty"`

	// Status applies to plugin entries only (Kind=="plugin"): the reconciliation
	// state between declared intent and installed reality. verified = found
	// installed at last scan; unverified = declared but not yet/cannot confirm;
	// missing = was tracked and a scan found it absent (out-of-band removal).
	Status string `json:"status,omitempty"`
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
	if l.Version < 1 || l.Version > NativeVersion {
		return nil, fmt.Errorf("unsupported lock schema v%d; upgrade patronus", l.Version)
	}
	if err := validateTarget(&l); err != nil {
		return nil, err
	}
	for _, e := range l.Entries {
		if e.NativeSource != "" {
			if l.Version != NativeVersion || l.Target != "pi" || e.Kind != "recipe" || e.Delivery != nil {
				return nil, fmt.Errorf("invalid native lock entry %q", e.Name)
			}
			if _, _, err := nativepi.ParseSource(e.NativeSource); err != nil {
				return nil, err
			}
		}
		if e.Delivery == nil {
			continue
		}
		if e.Kind != "recipe" || e.Delivery.Unpack != "directory" {
			return nil, fmt.Errorf("lock entry %q: pinned delivery requires a directory recipe", e.Name)
		}
		if err := manifest.ValidateDirectoryPin(e.Name, e.Version, e.Delivery); err != nil {
			return nil, fmt.Errorf("lock entry %q: %w", e.Name, err)
		}
		digest := strings.TrimPrefix(e.SHA256, "sha256:")
		if len(digest) != 64 {
			return nil, fmt.Errorf("lock entry %q: invalid manifest SHA-256", e.Name)
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return nil, fmt.Errorf("lock entry %q: invalid manifest SHA-256: %w", e.Name, err)
		}
	}
	return &l, nil
}

// Save writes l atomically as indented, deterministic JSON.
func Save(path string, l *Lock) error {
	if err := validateTarget(l); err != nil {
		return err
	}
	out, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return install.WriteFileAtomic(path, out, 0o644)
}

func validateTarget(l *Lock) error {
	if l.Version != TargetVersion && l.Version != NativeVersion {
		return nil
	}
	switch l.Target {
	case "pi", "claude", "codex", "opencode", "all":
		return nil
	default:
		return fmt.Errorf("lock schema v3 requires a known nonempty target, got %q", l.Target)
	}
}
