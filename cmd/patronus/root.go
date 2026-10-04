package main

import "github.com/spf13/cobra"

// jsonOutput is the persistent --json flag shared by output commands.
var jsonOutput bool

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "patronus",
		Short: "Meta-scaffolder for AI coding environments",
		Long: `Patronus installs artifacts, recipes, and profiles onto Claude Code, Codex,
OpenCode, and Pi at global or local-repo scope.

Governed Pi lifecycle (cooperative procedure, not a CLI security boundary):
Capture the normal preview and target/root/scope/item inputs; hash those bytes and
record a matching grant. Missing/stale grants stop the coordinator. Re-preview and
recheck changed inputs before --deploy. CLI does not read grants or bind an atomic
preview/apply transaction. Inventory old/new prototype sources and descriptions;
record functional-duplicate decisions. CLI checks static identities, not semantics.
Before update/remove --deploy, settle dependent work and acknowledge reload/restart;
global changes require all-consumer review. Deregister exact native sources before
payload removal. Loaded sessions may retain old bytes.
Removal selects explicit items, never inverse profile lifetimes. Desired lock pins
survive; profile update does not reinstall absent members. Optional removal is an
item decision. Static placement is runtime-unverified; runtime trust/qualification
is separate. Local global-prerequisite verification supports raw pinned scripts
and receipt-backed directory payloads. Legacy archive member-only ownership cannot
prove the archive pin; use supported delivery or separately qualify a migration.`,
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
	root.AddCommand(newCheckVersionsCmd())
	root.AddCommand(newCheckPlaceholdersCmd())
	root.AddCommand(newCheckGateIntentCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newRemoveCmd([]string{"revert"}))
	addStubCommands(root)
	return root
}
