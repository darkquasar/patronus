package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
)

// _blockingBinaries are the bare commands a hook may invoke that BLOCK the agent
// (exit non-zero to veto the tool call). A hook whose command runs one of these is
// a gate whether or not it ships a script, so it must declare intent: gate. The
// list is deliberately small and explicit: a binary NOT here is unclassifiable —
// the guard cannot know if it blocks — so it passes with an advisory rather than a
// false failure. Keep this in sync with the gate hooks that invoke a binary
// directly (today only tdd-guard).
var _blockingBinaries = map[string]bool{
	"tdd-guard": true,
}

// gateHook is the reconciled view of one hook artifact the gate-intent check needs:
// its declared intent plus the two signals that a hook BLOCKS — a bundled script
// that `exit 2`s, and a command that runs an allowlisted blocking binary.
type gateHook struct {
	Name string
	// Intent is the manifest's declared hook intent ("gate" | "nudge" | "").
	Intent manifest.HookIntent
	// ScriptExits2 is true when the hook bundles a script whose body contains an
	// `exit 2` — the shell convention for a PreToolUse veto.
	ScriptExits2 bool
	// Command is the hook's command line (first token is the invoked program).
	Command string
	// HasScript is true when the hook bundles a helper script (Command is then
	// typically "{script}", so binary-classification does not apply to it).
	HasScript bool
}

// gateViolation is a blocking hook that failed to declare intent: gate.
type gateViolation struct {
	Name   string
	Reason string // why it is a gate (script exit 2 / blocking binary)
}

func (v gateViolation) String() string {
	return fmt.Sprintf("%s: %s but does not declare intent: gate — add it, or the gate is a silent no-op on tools that key on intent (opencode)", v.Name, v.Reason)
}

// gateAdvisory is a hook the guard could NOT classify: it runs a bare binary that
// is not on the blocking allowlist, so the guard cannot know whether it vetoes.
// Reported (never silently skipped) so the blind spot is visible, but it does not
// fail the build.
type gateAdvisory struct {
	Name    string
	Command string
}

func (a gateAdvisory) String() string {
	return fmt.Sprintf("%s: command %q is not a recognised blocking binary — the gate-intent guard cannot classify it (add it to the allowlist if it blocks)", a.Name, a.Command)
}

// checkGateIntent is the pure heart of the guard: given the reconciled hooks, it
// returns the ones that BLOCK but lack intent: gate (violations, a hard failure)
// and the unclassifiable binary-command hooks (advisories, reported but passing).
//
// A hook is a gate when EITHER signal fires: its bundled script `exit 2`s, or its
// command runs an allowlisted blocking binary. Such a hook MUST carry intent: gate
// — on a tool that keys wiring on intent (opencode → permission deny), a gate
// mislabelled as a nudge wires nothing and the block silently never fires. A
// command-based hook that is neither a script nor an allowlisted binary is
// unclassifiable: the guard emits an advisory (visible blind spot) but passes.
func checkGateIntent(hooks []gateHook) (violations []gateViolation, advisories []gateAdvisory) {
	for _, h := range hooks {
		if h.Intent == manifest.HookGate {
			continue // already declared; nothing to enforce
		}
		switch {
		case h.ScriptExits2:
			violations = append(violations, gateViolation{Name: h.Name, Reason: "bundles a script that exits 2 (a blocking veto)"})
		case !h.HasScript && _blockingBinaries[firstToken(h.Command)]:
			violations = append(violations, gateViolation{Name: h.Name, Reason: fmt.Sprintf("runs the blocking binary %q", firstToken(h.Command))})
		case h.Intent == manifest.HookNudge:
			// The author explicitly asserted this hook does not block; take that at
			// face value — no advisory. The advisory exists only for a hook that made
			// NO intent claim and runs a binary we cannot classify.
		case !h.HasScript && h.Command != "":
			// A bare, unclassified command with no declared intent: pass, but surface
			// the blind spot so it is never a silent gap.
			advisories = append(advisories, gateAdvisory{Name: h.Name, Command: h.Command})
		}
	}
	return violations, advisories
}

// firstToken returns the first whitespace-separated token of a command line — the
// invoked program — so a command like "tdd-guard --check" classifies on "tdd-guard".
func firstToken(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// newCheckGateIntentCmd is the build/PR guard for pat-8jow: it fails when a hook
// that BLOCKS the agent (a script that exits 2, or a command running an
// allowlisted blocking binary) does not declare intent: gate. Without the
// declaration, a tool that keys wiring on intent — OpenCode, which maps a gate to
// a permission deny and a nudge to nothing — silently wires no block, turning a
// security gate into a no-op.
//
// It mirrors check-versions / check-placeholders: a pure decision (checkGateIntent)
// plus a thin wrapper that loads the hook manifests and their scripts.
//
// Binary-based detection covers ONLY the binaries on _blockingBinaries; a hook
// running any other bare binary passes with an advisory (the guard cannot know if
// it blocks), so the coverage is honest rather than silently partial.
func newCheckGateIntentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-gate-intent",
		Short: "Fail when a blocking hook does not declare intent: gate (build guard)",
		Long: "Loads every hook under artifacts/hooks/ and fails when one that BLOCKS the agent\n" +
			"(a bundled script that exits 2, or a command running an allowlisted blocking\n" +
			"binary) does not declare intent: gate. Such a hook is a silent no-op on tools\n" +
			"that key wiring on intent (opencode → permission deny). A hook running an\n" +
			"unrecognised bare binary passes with an advisory — the guard cannot classify it.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := registry.DiscoverRoot(wd)
			if err != nil {
				return err
			}
			hooks, err := loadGateHooks(root)
			if err != nil {
				return err
			}
			violations, advisories := checkGateIntent(hooks)
			for _, a := range advisories {
				fmt.Fprintln(cmd.OutOrStdout(), "advisory: "+a.String())
			}
			if len(violations) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "gate-intent check: ok")
				return nil
			}
			for _, v := range violations {
				fmt.Fprintln(cmd.ErrOrStderr(), v.String())
			}
			return fmt.Errorf("%d blocking hook(s) missing intent: gate", len(violations))
		},
	}
}

// loadGateHooks reads every hook manifest under artifacts/hooks/ and reconciles the
// signals checkGateIntent needs, reading each bundled script to detect an exit 2.
func loadGateHooks(root string) ([]gateHook, error) {
	hooksDir := filepath.Join(root, _artifactsDir, "hooks")
	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no hooks tree — nothing to check
		}
		return nil, fmt.Errorf("read hooks dir: %w", err)
	}
	var out []gateHook
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(hooksDir, e.Name())
		data, err := os.ReadFile(filepath.Join(dir, _manifestFile))
		if err != nil {
			if os.IsNotExist(err) {
				continue // not a hook artifact dir
			}
			return nil, fmt.Errorf("read %s manifest: %w", e.Name(), err)
		}
		var art manifest.Artifact
		if err := yaml.Unmarshal(data, &art); err != nil {
			return nil, fmt.Errorf("parse %s manifest: %w", e.Name(), err)
		}
		if art.Hook == nil {
			continue
		}
		h := gateHook{
			Name:      art.Name,
			Intent:    art.Hook.Intent,
			Command:   art.Hook.Command,
			HasScript: art.Hook.Script != "",
		}
		if art.Hook.Script != "" {
			body, err := os.ReadFile(filepath.Join(dir, art.Hook.Script))
			if err != nil {
				return nil, fmt.Errorf("read %s script %q: %w", e.Name(), art.Hook.Script, err)
			}
			h.ScriptExits2 = scriptExits2(body)
		}
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// scriptExits2 reports whether a shell script body contains an `exit 2` — the
// PreToolUse convention for vetoing a tool call. It matches "exit 2" as a token so
// "exit 20" or a substring in a comment word does not false-positive.
func scriptExits2(body []byte) bool {
	for line := range strings.Lines(string(body)) {
		fields := strings.Fields(line)
		for i := 0; i+1 < len(fields); i++ {
			if fields[i] == "exit" && fields[i+1] == "2" {
				return true
			}
		}
	}
	return false
}
