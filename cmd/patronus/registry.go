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

// fetcherForCommands is the network seam for out-of-tree SOURCE fetches (recipe
// binaries, git:/https: sourced refs) — Go-default TLS (1.2+), since those point
// at third-party upstreams we don't control. Integration tests swap it for a
// fakeFetcher so commands run end-to-end with zero network. Package var, not a
// flag: a test hook, not a user knob.
var fetcherForCommands registry.Fetcher = recipe.HTTPFetcher{}

// registryFetcher is the SEPARATE seam for catalog/registry fetches, defaulting
// to a TLS-1.3-floor client because the registry endpoint
// (registry.patronus.quasarops.com) is one we control. Tests swap it (often to
// the same fakeFetcher as fetcherForCommands) for network-free runs.
var registryFetcher registry.Fetcher = recipe.NewTLS13Fetcher()

// registrySel carries the registry-selection flags shared by list/install/lock.
type registrySel struct {
	localOnly bool   // --local-registry: force the on-disk checkout
	url       string // --registry-url: registry base URL override (fork/mirror)
}

// addRegistryFlags registers the shared registry-selection flags on a command.
func addRegistryFlags(cmd *cobra.Command, sel *registrySel) {
	cmd.Flags().BoolVar(&sel.localOnly, "local-registry", false, "force the local checkout registry (dev)")
	cmd.Flags().StringVar(&sel.url, "registry-url", "", "registry base URL override (fork/mirror)")
}

// registryBaseURL resolves the remote registry base: --registry-url flag, then the
// PATRONUS_REGISTRY_URL env var, then the built-in DefaultRegistryURL const.
func registryBaseURL(flagURL string) string {
	if flagURL != "" {
		return flagURL
	}
	if env, ok := os.LookupEnv("PATRONUS_REGISTRY_URL"); ok && env != "" {
		return env
	}
	return registry.DefaultRegistryURL
}

// resolveRegistry picks the registry implementation:
//   - --local-registry        → Local from the discovered checkout
//   - else, inside a checkout  → Local (dev mode; today's behavior, unchanged)
//   - else (installed binary)  → Remote at the resolved base URL (cold cache
//     bootstraps once)
//
// It returns the registry plus the checkout root ("" for remote), which callers
// use to find adapters/ (loadAdapters falls back to embedded adapters when root
// is "").
func resolveRegistry(_ context.Context, wd string, sel registrySel, home string, warnf func(string, ...any)) (registry.Registry, string, error) {
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
	// Installed binary with no checkout: use the remote R2 registry.
	cacheDir := filepath.Join(home, ".patronus", "cache")
	r := registry.NewRemoteRegistry(registryFetcher, cacheDir, registryBaseURL(sel.url))
	r.Warnf = warnf
	return r, "", nil
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
