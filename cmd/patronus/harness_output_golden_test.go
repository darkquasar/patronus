package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// Output goldens (HP-05): exact stdout and stderr of the install command for
// invented Codex and Pi selections, including the stream and order of every
// admission warning. $H and $P stand for the temporary home and catalog root.

const (
	harnessCodexWarning = "warning: Codex static admission/placement only; skill loading, hooks, trust and fresh-session activation remain runtime-unverified\n"
	harnessPiInventory  = "Pi static inventory only: checks cover inspected files, configuration and declared resources; extension callbacks and runtime-only discovery/registrations remain runtime-unverified. Cold-start/reload qualification is separate."
	harnessPiMixed      = "warning: Pi mixed-context conflict: operator-prepared combined file and separately scoped interactive consent required before any selected write\n"
)

// harnessOutputFixture is harnessAnyCatalog plus an invented instruction admitted
// for every target, with a fully isolated home.
func harnessOutputFixture(t *testing.T) (home, root string, norm func(string) string) {
	t.Helper()
	root = harnessAnyCatalog(t)
	codexWrite(t, filepath.Join(root, "artifacts/instructions/fixture-ctx/patronus.yaml"), []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: instruction\nrole: instruction\nname: fixture-ctx\ndescription: \"Invented instruction.\"\nversion: 1.0.0\nentry: INSTRUCTIONS.md\ntargets: [claude, codex, opencode, pi]\ndefaults:\n  scope: global\n"))
	codexWrite(t, filepath.Join(root, "artifacts/instructions/fixture-ctx/INSTRUCTIONS.md"), []byte("Invented contribution\n"))
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, ".pi", "agent"))
	t.Setenv("PI_OFFLINE", "true")
	t.Setenv("PI_PACKAGE_DIR", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("PI_SUBAGENT_EXTRA_AGENT_DIRS", "")
	t.Setenv("PI_MCP_CONFIG_MODE", "")
	t.Chdir(root)
	return home, root, func(s string) string { return strings.NewReplacer(home, "$H", root, "$P").Replace(s) }
}

func TestHarnessOutputGoldenCodexDryRun(t *testing.T) {
	_, _, norm := harnessOutputFixture(t)
	out, errOut, err := runInstall(t, "fixture-any", "--target", "codex", "--global")
	if err != nil {
		t.Fatal(err)
	}
	wantOut := "┌─────────────┬───────────────────────────────────────┬───────────┬───────┬────────────┬───────┬────────┐\n" +
		"│ Artifact    │ Impacted path(s)                      │ Operation │ Type  │ Role       │ Tool  │ Scope  │\n" +
		"├─────────────┼───────────────────────────────────────┼───────────┼───────┼────────────┼───────┼────────┤\n" +
		"│ fixture-any │ ~/.agents/skills/fixture-any/SKILL.md │ CREATE    │ skill │ capability │ codex │ global │\n" +
		"└─────────────┴───────────────────────────────────────┴───────────┴───────┴────────────┴───────┴────────┘\n" +
		"\n~/\n└── .agents/\n    └── skills/\n        └── fixture-any/\n            └── SKILL.md  (new)  # CREATE — role: capability\n\n" +
		"Plan: 1 CREATE\n(dry run — no files were written)\n"
	if got := norm(out); got != wantOut {
		t.Fatalf("codex stdout\n got: %q\nwant: %q", got, wantOut)
	}
	if got := norm(errOut); got != harnessCodexWarning {
		t.Fatalf("codex stderr\n got: %q\nwant: %q", got, harnessCodexWarning)
	}
}

func TestHarnessOutputGoldenPiDeploy(t *testing.T) {
	_, _, norm := harnessOutputFixture(t)
	out, errOut, err := runInstall(t, "fixture-any", "--target", "pi", "--global", "--deploy", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	wantOut := "Pi selection: global root=$H/.pi/agent; project root=$P/.pi; static resources become eligible at startup/reload, runtime-unverified.\n" +
		"Pi static placement: eligible at startup/reload; runtime-unverified. Config serialization may change formatting/comments.\n" +
		"  CREATE scope=global resource=fixture-any path=$H/.pi/agent/skills/fixture-any/SKILL.md\n" +
		"┌─────────────┬─────────────────────────────────────────┬───────────┬───────┬────────────┬──────┬────────┐\n" +
		"│ Artifact    │ Impacted path(s)                        │ Operation │ Type  │ Role       │ Tool │ Scope  │\n" +
		"├─────────────┼─────────────────────────────────────────┼───────────┼───────┼────────────┼──────┼────────┤\n" +
		"│ fixture-any │ ~/.pi/agent/skills/fixture-any/SKILL.md │ CREATE    │ skill │ capability │ pi   │ global │\n" +
		"└─────────────┴─────────────────────────────────────────┴───────────┴───────┴────────────┴──────┴────────┘\n" +
		"\n~/\n└── .pi/\n    └── agent/\n        └── skills/\n            └── fixture-any/\n                └── SKILL.md  (new)  # CREATE — role: capability\n\n" +
		"── fixture-any ──\nCREATE  ~/.pi/agent/skills/fixture-any/SKILL.md  [pi global]\n  @@ -1,0 +1,5 @@\n  +---\n  +name: fixture-any\n  +description: invented skill\n  +---\n  +Body\n\n\n" +
		"Plan: 1 CREATE\n" +
		harnessPiInventory + "\n" +
		"Pi planned placement, runtime-unverified: $H/.pi/agent/skills/fixture-any/SKILL.md scope=global; eligible after apply at next startup/reload (static apply does not activate packages).\n" +
		"\nApplied: 1 written, 0 skipped\n" +
		"Pi status: placed, runtime-unverified; eligible at next startup/reload (not proof of runtime registration).\n"
	if got := norm(out); got != wantOut {
		t.Fatalf("pi stdout\n got: %q\nwant: %q", got, wantOut)
	}
	if got, want := norm(errOut), "warning: "+harnessPiInventory+"\n"; got != want {
		t.Fatalf("pi stderr\n got: %q\nwant: %q", got, want)
	}
}

// A Pi instruction over an existing CLAUDE.md yields a mixed-context review: the
// planner's stderr warning precedes the static inventory warning; stdout is the
// ordinary dry-run plan.
func TestHarnessOutputGoldenPiMixedContextDryRun(t *testing.T) {
	home, _, norm := harnessOutputFixture(t)
	codexWrite(t, filepath.Join(home, ".pi/agent/CLAUDE.md"), []byte("Shared prose\n"))
	out, errOut, err := runInstall(t, "fixture-ctx", "--target", "pi", "--global")
	if err != nil {
		t.Fatal(err)
	}
	wantOut := "Pi selection: global root=$H/.pi/agent; project root=$P/.pi; static resources become eligible at startup/reload, runtime-unverified.\n" +
		"Pi static placement: eligible at startup/reload; runtime-unverified. Config serialization may change formatting/comments.\n" +
		"  APPEND scope=global resource=fixture-ctx path=$H/.pi/agent/CLAUDE.md\n" +
		"┌─────────────┬───────────────────────┬───────────┬─────────────┬─────────────┬──────┬────────┐\n" +
		"│ Artifact    │ Impacted path(s)      │ Operation │ Type        │ Role        │ Tool │ Scope  │\n" +
		"├─────────────┼───────────────────────┼───────────┼─────────────┼─────────────┼──────┼────────┤\n" +
		"│ fixture-ctx │ ~/.pi/agent/CLAUDE.md │ APPEND    │ instruction │ instruction │ pi   │ global │\n" +
		"└─────────────┴───────────────────────┴───────────┴─────────────┴─────────────┴──────┴────────┘\n" +
		"\n~/\n└── .pi/\n    └── agent/\n        └── CLAUDE.md  (appended)  # APPEND — role: instruction\n\n" +
		"── fixture-ctx ──\nAPPEND  ~/.pi/agent/CLAUDE.md  [pi global]\n  @@ -1,1 +1,5 @@\n   Shared prose\n  +\n  +<!-- patronus:start pi:fixture-ctx -->\n  +Invented contribution\n  +<!-- patronus:end pi:fixture-ctx -->\n\n\n" +
		"Plan: 1 APPEND\n(dry run — no files were written)\n"
	if got := norm(out); got != wantOut {
		t.Fatalf("pi mixed stdout\n got: %q\nwant: %q", got, wantOut)
	}
	if got, want := norm(errOut), harnessPiMixed+"warning: "+harnessPiInventory+"\n"; got != want {
		t.Fatalf("pi mixed stderr\n got: %q\nwant: %q", got, want)
	}
}

// A failing PiRootOverride surfaces as the command error; the planner prints
// nothing (no selection banner, inventory or placement line reaches stdout).
func TestHarnessOutputGoldenPiRootOverrideFailure(t *testing.T) {
	home, _, norm := harnessOutputFixture(t)
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, ".pi", "agent")+" ")
	out, errOut, err := runInstall(t, "fixture-any", "--target", "pi", "--global")
	if got, want := harnessErr(err), "pi root override: whitespace or URI spelling has unqualified host/MCP path semantics"; got != want {
		t.Fatalf("root override error\n got: %q\nwant: %q", got, want)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "Pi ") || strings.HasPrefix(line, "│") || strings.HasPrefix(line, "Plan:") {
			t.Fatalf("planner output printed before the failure: %q", norm(out))
		}
	}
	if got, want := norm(errOut), "Error: pi root override: whitespace or URI spelling has unqualified host/MCP path semantics\n"; got != want {
		t.Fatalf("root override stderr\n got: %q\nwant: %q", got, want)
	}
}
