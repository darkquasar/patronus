// Package source parses and resolves sourced references — the §5e grammar that
// lets a profile slot or an install argument name an item that lives outside the
// official registry (a local path, a git repo, a manifest URL) instead of just a
// bare catalog name.
//
// Every installable is already a self-describing folder (an artifact is a
// directory with a patronus.yaml; a recipe is a *.yaml manifest), so an
// out-of-tree item needs only a *location*: point Patronus at where the manifest
// lives, fetch+parse it, then fetch what it declares. This package is the front
// half — turning a reference string into a typed Ref and (for the schemes Phase 5
// supports) loading it into a catalog entry.
//
// Phase 5 scope: bare names (registry) and file: resolve end-to-end; git: and
// https: are fully PARSED so the grammar is accepted now, but resolution rejects
// them with a clear "Phase 6" error (full fetching lands with the remote
// registry, DESIGN §8 row 6). The lock's source provenance field (§5e) is
// populated for every scheme regardless, so the lock format is forward-compatible.
package source

import (
	"fmt"
	"strings"
)

// Scheme classifies a parsed reference.
type Scheme string

const (
	// Registry is a bare name resolved against the official catalog (the default,
	// unchanged from before §5e). Its canonical lock provenance is the literal
	// string "registry".
	Registry Scheme = "registry"
	// File is a local path (file:<path>) — developing a skill before publishing,
	// or an airgapped install. Resolved end-to-end in Phase 5.
	File Scheme = "file"
	// Git is a git repo reference (git:<host>/<owner>/<repo>[@<ref>][#<item>]).
	// Parsed now, fetched in Phase 6.
	Git Scheme = "git"
	// HTTPS is a direct manifest URL (https://….yaml). Parsed now, fetched in
	// Phase 6.
	HTTPS Scheme = "https"
)

// Ref is a parsed sourced reference. Raw is always the exact input string and is
// what the lock records in its `source` field (except Registry, whose canonical
// provenance is "registry"). The git fields are populated for Git refs so Phase 6
// can fetch without re-parsing.
type Ref struct {
	Scheme Scheme
	Raw    string // the exact reference string (canonical for the lock)
	Name   string // bare name (Registry) — the catalog lookup key
	Path   string // local path (File)

	// git: fields (populated for Scheme == Git; unused until Phase 6).
	Host   string
	Owner  string
	Repo   string
	GitRef string // tag/branch/commit after @ (empty => default branch)
	Item   string // artifact dir / recipe file after # (empty => repo is one item)
}

// Parse classifies a reference string. It fully validates every scheme's grammar
// so an invalid reference is caught at parse time, even for schemes whose
// resolution is deferred to Phase 6.
func Parse(ref string) (*Ref, error) {
	s := strings.TrimSpace(ref)
	if s == "" {
		return nil, fmt.Errorf("empty reference")
	}

	switch {
	case strings.HasPrefix(s, "file:"):
		path := strings.TrimPrefix(s, "file:")
		if path == "" {
			return nil, fmt.Errorf("file: reference has no path: %q", ref)
		}
		return &Ref{Scheme: File, Raw: s, Path: path}, nil

	case strings.HasPrefix(s, "git:"):
		return parseGit(s)

	case strings.HasPrefix(s, "https://"):
		if !strings.HasSuffix(s, ".yaml") && !strings.HasSuffix(s, ".yml") {
			return nil, fmt.Errorf("https source must point at a .yaml manifest: %q", ref)
		}
		return &Ref{Scheme: HTTPS, Raw: s}, nil

	case strings.HasPrefix(s, "http://"):
		return nil, fmt.Errorf("insecure http:// sources are not allowed; use https://: %q", ref)

	default:
		// A bare name. Reject anything that looks like a mistyped scheme so a typo
		// (e.g. "gti:...") doesn't silently become a registry lookup.
		if i := strings.IndexByte(s, ':'); i >= 0 {
			return nil, fmt.Errorf("unknown source scheme %q (want file:/git:/https: or a bare name)", s[:i+1])
		}
		return &Ref{Scheme: Registry, Raw: s, Name: s}, nil
	}
}

// parseGit decodes git:<host>/<owner>/<repo>[@<ref>][#<item>].
func parseGit(s string) (*Ref, error) {
	body := strings.TrimPrefix(s, "git:")
	r := &Ref{Scheme: Git, Raw: s}

	// Split off the optional #item selector first.
	if i := strings.IndexByte(body, '#'); i >= 0 {
		r.Item = body[i+1:]
		body = body[:i]
	}
	// Then the optional @ref.
	if i := strings.IndexByte(body, '@'); i >= 0 {
		r.GitRef = body[i+1:]
		body = body[:i]
	}

	parts := strings.Split(body, "/")
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, fmt.Errorf("git source must be git:<host>/<owner>/<repo>[@ref][#item]: %q", s)
	}
	r.Host = parts[0]
	r.Owner = parts[1]
	// repo may itself contain slashes? GitHub-style is owner/repo, so the rest is
	// the repo path (allows nested hosts like a self-hosted gitlab group).
	r.Repo = strings.Join(parts[2:], "/")
	return r, nil
}

// LockSource returns the canonical provenance string for the lock's `source`
// field: "registry" for bare names, otherwise the exact reference.
func (r *Ref) LockSource() string {
	if r.Scheme == Registry {
		return string(Registry)
	}
	return r.Raw
}
