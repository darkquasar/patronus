//go:build !artifactbehavior

package main

// Application tests deliver these bytes but never execute a hook program.
const fixClaudeHookBytes = "Inert fixture hook bytes, only placed and removed.\n"
