package main

import "github.com/spf13/cobra"

// jsonOutput is the persistent --json flag shared by output commands.
var jsonOutput bool

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "patronus",
		Short:         "Meta-scaffolder for AI coding environments",
		Long:          "Patronus installs artifacts, recipes, and profiles onto Claude Code, Codex, and OpenCode — at the global or local-repo scope.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "emit machine-readable JSON")

	root.AddCommand(newListCmd())
	root.AddCommand(newScanCmd())
	root.AddCommand(newInstallCmd())
	root.AddCommand(newLockCmd())
	root.AddCommand(newBuildCmd())
	root.AddCommand(newUpdateCmd())
	addStubCommands(root)
	return root
}
