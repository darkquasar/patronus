package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
	"github.com/darkquasar/patronus/internal/scan"
)

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Detect installed AI coding tools and their configs",
		Long:  "Detects Claude Code, Codex, and OpenCode at global and local scope using the repo's adapter detect: markers (honoring CODEX_HOME, OPENCODE_CONFIG_DIR, XDG_CONFIG_HOME).",
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
			adapters, err := loadAdapters(filepath.Join(root, "adapters"))
			if err != nil {
				return err
			}
			inv, err := scan.Scan(scan.Options{ProjectDir: wd, Adapters: adapters})
			if err != nil {
				return err
			}
			if jsonOutput {
				return render.JSON(cmd.OutOrStdout(), inv)
			}
			render.PrintInventory(cmd.OutOrStdout(), inv)
			return nil
		},
	}
}

func loadAdapters(dir string) ([]*manifest.Adapter, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	var out []*manifest.Adapter
	for _, path := range matches {
		ad, err := manifest.LoadAdapter(path)
		if err != nil {
			return nil, err
		}
		out = append(out, ad)
	}
	return out, nil
}
