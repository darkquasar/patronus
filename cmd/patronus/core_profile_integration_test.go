package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/profile"
)

// These are the §6b acceptance gate for the real `core` profile — and they are the
// CLASS-B heart of the suite: they assert what the CATALOG SHIPS, not what Patronus
// can do.
//
// ⚠️ coreSkills below is a PRODUCT GUARANTEE: "the core profile really wires
// grilling, tdd, executing-plans, ...". A fixture CANNOT express it — the real names
// ARE the assertion. Renaming them to fixture names would produce a green tautology
// that passes forever while testing nothing. DO NOT rename them, and do NOT point
// this file at fixtureRegistry.
//
// What makes a real-catalog test SAFE is that it reads the catalog's SHAPE and never
// its PINS: it must not fetch, hash an upstream digest, or place a binary. So these
// assert against profile.Resolve, the LOCK, and the PLAN — all of which resolve from
// the catalog without fetching a thing.
//
// They used to --deploy, kept offline by stubBinary pre-placing dummy bytes so core's
// gitleaks FETCH classified SKIP. That only ever worked because classifyFetch SKIPped
// an archive on MERE PRESENCE without hashing it — the security hole this work closes.
// Once Patronus hashes the binary it placed, a stub is correctly detected as tampered
// and re-fetched, so a DEPLOYED real-catalog test becomes structurally impossible: it
// would have to fetch upstream bytes in CI (forbidden) or weaken the sha check
// (forbidden). See test-surface-plan.md, Task 9.
//
// The deploy MECHANICS those tests also covered — composed APPEND into one CLAUDE.md,
// hooks folding into one settings array, per-tool flavour divergence, selective remove
// round-trips — are proven on the FIXTURE catalog, with items and binaries this suite
// invents, where the FETCH path actually RUNS (which stubBinary never did).

// coreSkills are the vendored L2+L4 skill artifacts core wires that land as a
// per-tool skills/<name>/SKILL.md file.
//
// ⚠️ These names ARE the assertion. See the file comment. Do not rename them.
var coreSkills = []string{
	"skills-dispatch", "writing-plans", "executing-plans",
	"grilling", "diagnosing-bugs", "tdd",
	"codebase-design", "domain-modeling",
	"verification-before-completion", // P7.5.2 L8 eval skill
}

// withFakeRunner swaps the self-wiring EXEC runner for a process-free fake (core
// wires memory-ai-memory, whose post-install shells out to the ai-memory binary)
// and restores it on cleanup, keeping the suite offline + process-free.
func withFakeRunner(t *testing.T) {
	t.Helper()
	prev := runnerForCommands
	runnerForCommands = &fakeRunner{}
	t.Cleanup(func() { runnerForCommands = prev })
}

// resolvedNames returns the item names a real-catalog profile resolves to for a tool.
// No install, no plan, no fetch: resolution is where "which items does this profile
// name" actually lives.
func resolvedNames(t *testing.T, profileName, tool string) string {
	t.Helper()
	r, err := profile.Resolve(realCatalog(t), profileName, tool)
	if err != nil {
		t.Fatalf("resolve %s/%s: %v", profileName, tool, err)
	}
	return strings.Join(r.Names(), " ")
}

// TestCoreProfileWiresTheSkillSpine is the headline CLASS-B guarantee: the `core`
// profile really ships every skill in coreSkills, plus the L1 instruction spine and
// the diagram-explain output-style.
func TestCoreProfileWiresTheSkillSpine(t *testing.T) {
	names := resolvedNames(t, "core", "claude")
	for _, want := range append([]string{"agents-spine", "agent-rules", "diagram-explain"}, coreSkills...) {
		if !strings.Contains(names, want) {
			t.Errorf("core does not wire %q — resolved: %s", want, names)
		}
	}
}

// TestCoreProfileWiresTheSpineOnEveryTool is CLASS B per tool: core resolves the same
// spine under codex and opencode, not just claude. (HOW each item LANDS per tool — a
// Claude output-styles FILE vs an AGENTS.md section — is the adapter MECHANISM, proven
// on the fixture in outputstyle_integration_test.go.)
func TestCoreProfileWiresTheSpineOnEveryTool(t *testing.T) {
	for _, tool := range []string{"codex", "opencode"} {
		t.Run(tool, func(t *testing.T) {
			names := resolvedNames(t, "core", tool)
			for _, want := range append([]string{"agents-spine", "agent-rules", "diagram-explain"}, coreSkills...) {
				if !strings.Contains(names, want) {
					t.Errorf("core/%s does not wire %q — resolved: %s", tool, want, names)
				}
			}
		})
	}
}

// TestCoreProfileLockPinsEveryItem locks core and asserts the lock pins each resolved
// item per-item (the L1 instructions + every vendored skill), so a re-install against
// the committed lock is reproducible. CLASS B — and `lock` resolves and pins from the
// catalog without fetching a binary, so it stays off the fetch path.
func TestCoreProfileLockPinsEveryItem(t *testing.T) {
	f := builtRegistry(t)
	withRemoteEnv(t, f)

	if _, _, err := runLock(t, "--profile", "core", "--tool", "claude"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	wd, _ := os.Getwd()
	s := string(mustRead(t, filepath.Join(wd, "patronus.lock")))
	if !strings.Contains(s, `"version": 2`) {
		t.Errorf("lock should be schema v2:\n%s", s)
	}
	for _, want := range append([]string{"agents-spine", "agent-rules", "diagram-explain", "tarballSha256"}, coreSkills...) {
		if !strings.Contains(s, want) {
			t.Errorf("lock missing %q:\n%s", want, s)
		}
	}
}

// TestCoreOmitsTddEnforcementButShipsTheSkill is CLASS B: core no longer ENFORCES
// test-first — no tdd-guard binary, no tdd-guard-hook — but the tdd SKILL (guidance)
// does ship. Enforcement is opt-in via tdd-enforced.
func TestCoreOmitsTddEnforcementButShipsTheSkill(t *testing.T) {
	names := resolvedNames(t, "core", "claude")
	for _, unwanted := range []string{"tdd-guard-hook", "tdd-guard"} {
		if strings.Contains(names, unwanted) {
			t.Errorf("core must NOT carry the enforcement item %q — resolved: %s", unwanted, names)
		}
	}
	for _, want := range []string{"tdd", "verification-before-completion"} {
		if !strings.Contains(names, want) {
			t.Errorf("core should keep the %q skill — resolved: %s", want, names)
		}
	}
}

// TestTddEnforcedProfileAddsEnforcement is the CLASS-B inverse: the opt-in
// tdd-enforced profile (extends core) ADDS the test-first gate — the tdd-guard recipe
// and the tdd-guard-hook — on top of the full core spine.
func TestTddEnforcedProfileAddsEnforcement(t *testing.T) {
	names := resolvedNames(t, "tdd-enforced", "claude")
	for _, want := range []string{"tdd-guard", "tdd-guard-hook", "tdd"} {
		if !strings.Contains(names, want) {
			t.Errorf("tdd-enforced should wire %q — resolved: %s", want, names)
		}
	}
}

// TestCcusageStatuslineIsClaudeOnly is CLASS B: the ccusage statusline setting is
// @claude-flavoured, so core carries it on claude and NOT on codex/opencode.
func TestCcusageStatuslineIsClaudeOnly(t *testing.T) {
	if names := resolvedNames(t, "core", "claude"); !strings.Contains(names, "ccusage-statusline") {
		t.Errorf("core/claude should wire ccusage-statusline — resolved: %s", names)
	}
	for _, tool := range []string{"codex", "opencode"} {
		if names := resolvedNames(t, "core", tool); strings.Contains(names, "ccusage-statusline") {
			t.Errorf("core/%s must NOT wire the claude-only ccusage-statusline — resolved: %s", tool, names)
		}
	}
}

// TestCoreGuardrails is CLASS B for the L9 guardrail set via safe-git (core +
// git-guardrails). It asserts against the PLAN, which places nothing and fetches
// nothing — and the plan DOES surface these rows: the three guardrail hooks, their
// placed scripts, and the gitleaks binary as a FETCH.
//
// (It is the settings.json MERGEs the plan under-reports, not hook scripts or recipe
// FETCHes — see pat-resg.)
func TestCoreGuardrails(t *testing.T) {
	f := builtRegistry(t)
	withRemoteEnv(t, f)

	out, _, err := runInstall(t, "--profile", "safe-git", "--tool", "claude", "--global", "--dry-run", "--verbose")
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	for _, want := range []string{
		"block-secrets",
		"gitleaks-guard",
		"git-guardrails",
		"git-guardrails.sh", // a placed hook script (named after the artifact)
		"block-secrets.sh",  // the regex guard's placed script
		"gitleaks",          // the row for the binary
	} {
		if !strings.Contains(out, want) {
			t.Errorf("safe-git dry-run plan missing %q:\n%s", want, out)
		}
	}
	// The gitleaks binary is PLANNED as a FETCH — and never performed (dry run).
	if !strings.Contains(out, "FETCH") {
		t.Errorf("expected a FETCH row for the gitleaks binary:\n%s", out)
	}
}

// TestCoreSurfacesPackageInstalls is CLASS B: core really wires the ccusage
// package-manager recipe, and tdd-enforced really wires tdd-guard — each surfacing
// its global-install command in the plan for the user to run.
//
// It asserts the COMMAND is surfaced, not that it is marked advisory. The
// "display-only, never auto-run" invariant lives on diff.Exec.Advisory, which is set
// at plan time and honored at apply time (install.go) — and it is already proven on
// invented recipes by internal/recipe's TestComputeInstallOnly_EmitsAdvisory. The
// literal "ADVISORY (run yourself)" string is printed only on the DEPLOY path, which
// this test cannot take (core wires the gitleaks + tk recipes with real upstream
// pins). Duplicating the mechanism here would add no coverage; what only the real
// catalog can say is that core NAMES these recipes.
func TestCoreSurfacesPackageInstalls(t *testing.T) {
	for _, tc := range []struct{ profileName, recipe, command string }{
		{"core", "ccusage", "npm install -g ccusage"},
		{"tdd-enforced", "tdd-guard", "npm install -g tdd-guard"},
	} {
		t.Run(tc.profileName, func(t *testing.T) {
			if names := resolvedNames(t, tc.profileName, "claude"); !strings.Contains(names, tc.recipe) {
				t.Errorf("%s should wire the %q recipe — resolved: %s", tc.profileName, tc.recipe, names)
			}

			f := builtRegistry(t)
			withRemoteEnv(t, f)
			withFakeRunner(t)

			out, e, err := runInstall(t, "--profile", tc.profileName, "--tool", "claude", "--global")
			if err != nil {
				t.Fatalf("plan: %v\n%s", err, e)
			}
			if !strings.Contains(out, tc.command) {
				t.Errorf("%s's plan should surface the install command %q:\n%s", tc.profileName, tc.command, out)
			}
		})
	}
}

// TestScalarSettingRemoveRoundTrips is the CLASS-A counterpart, on the FIXTURE: a
// scalar SETTING merges into settings.json and removes cleanly — the key is gone, and
// the sibling hooks in the same file survive. That is the mechanism ccusage-statusline
// relies on, proven with an item this suite invented, deployed for real.
func TestScalarSettingRemoveRoundTrips(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	if _, e, err := runInstall(t, "--profile", "fix-all", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install profile: %v\n%s", err, e)
	}
	if _, e, err := runInstall(t, "fix-setting", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install fix-setting: %v\n%s", err, e)
	}
	settings := filepath.Join(home, ".claude", "settings.json")
	if !strings.Contains(string(mustRead(t, settings)), "fixtureLine") {
		t.Fatal("the scalar setting was not merged on install")
	}

	if _, e, err := execRemove(t, "fix-setting", "--global", "--deploy"); err != nil {
		t.Fatalf("remove fix-setting: %v\n%s", err, e)
	}
	root := map[string]any{}
	if err := json.Unmarshal(mustRead(t, settings), &root); err != nil {
		t.Fatalf("settings corrupt after remove: %v", err)
	}
	if _, present := root["fixtureLine"]; present {
		t.Errorf("the scalar setting should be gone after remove: %v", root["fixtureLine"])
	}
	// The hooks survive — removing the scalar setting did not clobber them.
	if _, ok := root["hooks"].(map[string]any)["PreToolUse"].([]any); !ok {
		t.Errorf("hooks should survive removing the scalar setting: %v", root["hooks"])
	}
}
