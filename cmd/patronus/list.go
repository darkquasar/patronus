package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
)

func newListCmd() *cobra.Command {
	var (
		artifacts bool
		recipes   bool
		profiles  bool
		layers    bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the catalog of artifacts, recipes, and profiles",
		Long:  "Reads the local registry (on-disk manifests in the current Patronus repo) and lists installable items. With no type flag, all three sections are shown.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := registry.DiscoverRoot(wd)
			if err != nil {
				return err
			}
			cat, err := registry.NewLocalRegistry(root).Catalog(context.Background())
			if err != nil {
				return err
			}

			// No type flag => show everything.
			view := render.CatalogView{Artifacts: artifacts, Recipes: recipes, Profiles: profiles, Layers: layers}
			if !artifacts && !recipes && !profiles {
				view.Artifacts, view.Recipes, view.Profiles = true, true, true
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
	cmd.Flags().BoolVar(&layers, "layers", false, "expand profile layers (with --profiles)")
	return cmd
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
	return out
}
