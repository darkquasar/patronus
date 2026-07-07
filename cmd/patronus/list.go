package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
)

func newListCmd() *cobra.Command {
	var (
		artifacts   bool
		recipes     bool
		profiles    bool
		plugins     bool
		layers      bool
		description bool
		artifact    string
		regSel      registrySel
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the catalog of artifacts, recipes, profiles, and plugins",
		Long:  "Lists installable items from the registry — the local checkout when run inside a Patronus repo, otherwise the cached remote registry (a cold cache bootstrap-fetches once). With no type flag, all four sections are shown.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }
			reg, _, err := resolveRegistry(cmd.Context(), wd, regSel, homeDir(), warnf)
			if err != nil {
				return err
			}
			cat, err := reg.Catalog(cmd.Context())
			if err != nil {
				return err
			}

			// No type flag => show everything. A single-artifact lookup (--artifact)
			// or --description is artifact-scoped, so it implies the artifacts section.
			view := render.CatalogView{
				Artifacts: artifacts, Recipes: recipes, Profiles: profiles, Plugins: plugins, Layers: layers,
				Description: description, Artifact: artifact,
			}
			switch {
			case artifact != "" || description:
				// Scope to artifacts unless the user also explicitly asked for others.
				if !recipes && !profiles && !plugins {
					view.Artifacts, view.Recipes, view.Profiles, view.Plugins = true, false, false, false
				} else {
					view.Artifacts = true
				}
			case !artifacts && !recipes && !profiles && !plugins:
				view.Artifacts, view.Recipes, view.Profiles, view.Plugins = true, true, true, true
			}

			// When the plugin section is shown and a project lock exists, surface each
			// plugin's reconciliation status (verified|unverified|missing) beside its
			// name; a catalog plugin absent from the lock reads "untracked".
			if view.Plugins {
				view.PluginStatus = pluginStatusFromLock(wd)
			}

			if jsonOutput {
				return render.JSON(cmd.OutOrStdout(), filterCatalog(cat, view))
			}
			render.PrintCatalog(cmd.OutOrStdout(), cat, view)
			return nil
		},
	}
	cmd.Flags().BoolVar(&artifacts, "artifacts", false, "show artifacts")
	cmd.Flags().BoolVar(&recipes, "recipes", false, "show recipes")
	cmd.Flags().BoolVar(&profiles, "profiles", false, "show profiles")
	cmd.Flags().BoolVar(&plugins, "plugins", false, "show plugins")
	cmd.Flags().BoolVar(&layers, "layers", false, "expand profile layers (with --profiles)")
	cmd.Flags().BoolVar(&description, "description", false, "show artifacts as a block list with each item's full description (instead of the compact table)")
	cmd.Flags().StringVar(&artifact, "artifact", "", "show the full details of a single artifact by name")
	addRegistryFlags(cmd, &regSel)
	return cmd
}

// pluginStatusFromLock reads <wd>/patronus.lock and returns a name->status map of
// its plugin entries, or nil when there is no lock (or it has no plugins) — nil
// keeps list's plugin section in its original no-status-column form.
func pluginStatusFromLock(wd string) map[string]string {
	l, err := lock.Load(filepath.Join(wd, "patronus.lock"))
	if err != nil || len(l.Entries) == 0 {
		return nil
	}
	out := map[string]string{}
	for _, e := range l.Entries {
		if e.Kind == "plugin" {
			out[e.Name] = e.Status
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// filterCatalog returns a catalog containing only the sections the view selects,
// so --json output matches what the text view would show.
func filterCatalog(cat *registry.Catalog, view render.CatalogView) *registry.Catalog {
	out := &registry.Catalog{}
	if view.Artifacts {
		out.Artifacts = cat.Artifacts
	}
	if view.Recipes {
		out.Recipes = cat.Recipes
	}
	if view.Profiles {
		out.Profiles = cat.Profiles
	}
	if view.Plugins {
		out.Plugins = cat.Plugins
	}
	return out
}
