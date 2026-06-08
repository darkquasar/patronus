package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// fetcherForCommands is the network seam every command uses to reach the remote
// registry and out-of-tree sources. It defaults to the real HTTPS fetcher;
// integration tests swap it for a fakeFetcher serving a canned in-memory registry
// so the commands run end-to-end with zero network. Kept as a package var (not a
// flag) because it is a test hook, not a user-facing knob.
var fetcherForCommands registry.Fetcher = recipe.HTTPFetcher{}

// registrySel carries the registry-selection flags shared by list/install/lock.
type registrySel struct {
	version   string // --registry-version <tag>; "" => default policy
	localOnly bool   // --local-registry: force the on-disk checkout
	url       string // --registry-url: fork/mirror base (advanced)
}

// addRegistryFlags registers the shared registry-selection flags on a command.
func addRegistryFlags(cmd *cobra.Command, sel *registrySel) {
	cmd.Flags().StringVar(&sel.version, "registry-version", "", "install against a specific registry release tag (default: latest, or the local checkout)")
	cmd.Flags().BoolVar(&sel.localOnly, "local-registry", false, "force the local checkout registry (dev)")
	cmd.Flags().StringVar(&sel.url, "registry-url", "", "registry base URL override (fork/mirror)")
}

// resolveRegistry picks the registry implementation:
//   - --registry-version <tag>  → Remote pinned at that tag (works in a checkout too)
//   - --local-registry          → Local from the discovered checkout
//   - else, inside a checkout   → Local (dev mode; today's behavior, unchanged)
//   - else (installed binary)   → Remote at latest (cold cache bootstraps once)
//
// It returns the registry plus the checkout root ("" for remote), which callers
// use to find adapters/ (loadAdapters falls back to embedded adapters when root
// is "").
func resolveRegistry(_ context.Context, wd string, sel registrySel, home string, warnf func(string, ...any)) (registry.Registry, string, error) {
	cacheDir := filepath.Join(home, ".patronus", "cache")
	newRemote := func(version string) *registry.RemoteRegistry {
		r := registry.NewRemoteRegistry(fetcherForCommands, cacheDir, version)
		r.BaseURL = sel.url
		r.Warnf = warnf
		return r
	}

	if sel.version != "" {
		return newRemote(sel.version), "", nil
	}
	if sel.localOnly {
		root, err := registry.DiscoverRoot(wd)
		if err != nil {
			return nil, "", fmt.Errorf("--local-registry: %w", err)
		}
		return registry.NewLocalRegistry(root), root, nil
	}
	if root, err := registry.DiscoverRoot(wd); err == nil {
		return registry.NewLocalRegistry(root), root, nil
	}
	// Installed binary with no checkout: use the remote registry at latest.
	return newRemote(""), "", nil
}

// registryVersionOf reports the concrete release tag a registry resolved against,
// for the lock's provenance field ("" for a local checkout / dev). For an
// unpinned remote registry this is the tag the loaded index reported, so the lock
// pins the exact snapshot even when the user said "latest".
func registryVersionOf(reg registry.Registry) string {
	if rr, ok := reg.(*registry.RemoteRegistry); ok {
		return rr.ResolvedVersion()
	}
	return ""
}

// materializeSelected ensures every selected remote artifact's portable source is
// on disk (Source.LocalDir set), so the unchanged plan.Compute/adapter path runs
// against it. It is a no-op for a LocalRegistry and for recipes. Only the named
// items are materialized — never the whole catalog — preserving "don't download
// content just to browse".
func materializeSelected(ctx context.Context, reg registry.Registry, cat *registry.Catalog, names []string) error {
	rr, ok := reg.(*registry.RemoteRegistry)
	if !ok {
		return nil
	}
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	for i := range cat.Artifacts {
		e := &cat.Artifacts[i]
		if !want[e.Manifest.Name] || e.Source.LocalDir != "" {
			continue
		}
		if _, err := rr.Materialize(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

// homeDir returns the user's home for cache/state path construction.
func homeDir() string {
	return toolpath.HomeDir(os.LookupEnv)
}
