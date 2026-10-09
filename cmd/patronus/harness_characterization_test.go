package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// This file pins the Codex and Pi admission behaviour at the base revision with
// invented fixtures (ADR-0002). Every row records an exact error string observed
// before the harness seam existed; a later refactor must keep each row green
// without editing it. Paths use the placeholders $H (home) and $P (project).

const (
	harnessCodexSkill = "---\nname: fixture-cx\ndescription: invented Codex test\n---\nfixture\n"
	harnessPiSkill    = "---\nname: fixture-pi\ndescription: invented Pi test\n---\nfixture\n"
)

// harnessGolden is one admission scenario: the planned diffs, the files that
// exist before admission, the recorded owners (saved under the project state)
// and the exact expected error ("" means admitted).
type harnessGolden struct {
	name   string
	diffs  []diff.FileDiff
	files  map[string]string
	owners []state.Item
	want   string
}

func harnessExpand(s, home, project string) string {
	return strings.NewReplacer("$H", home, "$P", project).Replace(s)
}

func harnessExpandDiffs(diffs []diff.FileDiff, home, project string) []diff.FileDiff {
	out := make([]diff.FileDiff, 0, len(diffs))
	for _, d := range diffs {
		d.Path = harnessExpand(d.Path, home, project)
		out = append(out, d)
	}
	return out
}

func harnessExpandOwners(owners []state.Item, home, project string) []state.Item {
	out := make([]state.Item, 0, len(owners))
	for _, it := range owners {
		files := make([]state.FileState, 0, len(it.Files))
		for _, f := range it.Files {
			f.Path = harnessExpand(f.Path, home, project)
			files = append(files, f)
		}
		it.Files = files
		out = append(out, it)
	}
	return out
}

func harnessWriteFiles(t *testing.T, files map[string]string, home, project string) {
	t.Helper()
	for path, body := range files {
		codexWrite(t, harnessExpand(path, home, project), []byte(body))
	}
}

func harnessSaveOwners(t *testing.T, owners []state.Item, home, project string) {
	t.Helper()
	if len(owners) == 0 {
		return
	}
	s := &state.State{Version: state.Version, Items: harnessExpandOwners(owners, home, project)}
	if err := state.Save(filepath.Join(project, ".patronus/state.json"), s); err != nil {
		t.Fatal(err)
	}
}

func harnessErr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// piFixture mirrors codexFixture for Pi: isolated roots and the environment Pi
// discovery consults, with no native package manager reachable.
func piFixture(t *testing.T) (string, string, toolpath.Resolver) {
	t.Helper()
	home, project := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, ".pi", "agent"))
	t.Setenv("PI_PACKAGE_DIR", "")
	t.Setenv("PI_OFFLINE", "true")
	t.Setenv("PI_SUBAGENT_EXTRA_AGENT_DIRS", "")
	t.Setenv("PI_MCP_CONFIG_MODE", "")
	if err := os.Mkdir(filepath.Join(project, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	return home, project, toolpath.New(os.LookupEnv, home, project)
}

func codexSkillDiff() diff.FileDiff {
	return diff.FileDiff{Path: "$P/.agents/skills/fixture-cx/SKILL.md", Action: diff.Create, Artifact: "fixture-cx", Type: "skill", Tool: "codex", Scope: "local", After: []byte(harnessCodexSkill)}
}

func codexSectionDiff(before []byte) diff.FileDiff {
	return diff.FileDiff{Path: "$P/AGENTS.md", Action: diff.Append, Artifact: "fixture-cx", Type: "instruction", Tool: "codex", Scope: "local", Section: &diff.SectionEdit{Name: "fixture-cx", Body: []byte("new")}, Before: before, After: adapter.AppendSection(before, "fixture-cx", []byte("new"))}
}

func codexOwner(tool, artifact, scope, path, section, action string, before []byte) state.Item {
	return state.Item{Artifact: artifact, Type: "skill", Tool: tool, Scope: scope, Files: []state.FileState{{Path: path, Checksum: codexChecksum(before), Section: section, Action: action}}}
}

func TestHarnessGoldenCodexPreflight(t *testing.T) {
	ownedSection := []byte("User prose\n\n<!-- patronus:start fixture-cx -->\nold\n<!-- patronus:end fixture-cx -->\n")
	siblingSection := adapter.AppendSection(ownedSection, "sibling-cx", []byte("sibling"))
	composite := codexSkillDiff()
	composite.Tool = "codex+pi"
	rawTOML := diff.FileDiff{Path: "$P/.codex/config.toml", Action: diff.Create, Artifact: "fixture-cx", Tool: "codex", Scope: "local", After: []byte("[x]\n")}
	existingSkill := codexSkillDiff()
	existingSkill.Before = []byte(harnessCodexSkill)
	existingSkill.Action = diff.Skip
	edited := codexSkillDiff()
	edited.Before = []byte(harnessCodexSkill + "edited\n")
	edited.Action = diff.Conflict
	relocated := codexSectionDiff(nil)
	restore := codexSectionDiff(siblingSection)
	restore.Action = diff.Restore
	restore.Section = nil
	restore.After = nil
	unappendNoSection := codexSectionDiff(ownedSection)
	unappendNoSection.Action = diff.Unappend
	unappendNoSection.Section = nil
	ambiguousSelected := codexSectionDiff(ownedSection)
	ambiguousSelected.Section = &diff.SectionEdit{Name: "other-cx", Body: []byte("new")}
	unappendMissing := codexSectionDiff(siblingSection)
	unappendMissing.Action = diff.Unappend
	unappendMissing.Before = adapter.AppendSection([]byte("User prose\n"), "sibling-cx", []byte("sibling"))
	unappendMissing.After = []byte("User prose\n")
	unappendChangesProse := codexSectionDiff(ownedSection)
	unappendChangesProse.Action = diff.Unappend
	unappendChangesProse.After = []byte("Different prose\n")
	tomlSetting := &diff.SettingEdit{Target: diff.FileTargetRef{File: ".codex/config.toml", Format: "toml"}, Dotted: "mcp_servers.fixture", ScalarValue: map[string]any{"url": "https://fixture.invalid/mcp"}}
	tomlLeaf := &diff.SettingEdit{Target: tomlSetting.Target, Dotted: "mcp_servers.fixture.url", ScalarValue: "https://fixture.invalid/mcp"}
	tomlOther := &diff.SettingEdit{Target: tomlSetting.Target, Dotted: "mcp_servers.other", ScalarValue: map[string]any{"url": "https://other.invalid/mcp"}}
	inverse := diff.FileDiff{Path: "$P/.codex/config.toml", Action: diff.Restore, Artifact: "fixture-cx", Tool: "codex", Scope: "local", Before: []byte("[mcp_servers.fixture]\nurl = 'https://fixture.invalid/mcp'\n")}
	inverseDrift := inverse
	inverseDrift.Before = []byte("[mcp_servers.fixture]\nurl = 'https://changed.invalid/mcp'\n")
	inverseBroken := inverse
	inverseBroken.Before = []byte("not = [toml\n")
	settingOwner := func(artifact, tool string, edit *diff.SettingEdit) state.Item {
		return state.Item{Artifact: artifact, Tool: tool, Scope: "local", Files: []state.FileState{{Path: "$P/.codex/config.toml", Action: "MERGE", Setting: edit}}}
	}
	for _, tc := range []harnessGolden{
		{name: "composite cross-target destination", diffs: []diff.FileDiff{composite},
			want: "codex cross-target destination $P/.agents/skills/fixture-cx/SKILL.md (codex+pi): explicit mixed-context migration required"},
		{name: "raw TOML mutation", diffs: []diff.FileDiff{rawTOML},
			want: "codex config $P/.codex/config.toml lacks structured SettingEdit ownership; raw TOML mutation refused"},
		{name: "raw TOML wins over unknown ownership", diffs: []diff.FileDiff{rawTOML}, files: map[string]string{"$P/.codex/config.toml": "model = 'x'\n"},
			want: "codex config $P/.codex/config.toml lacks structured SettingEdit ownership; raw TOML mutation refused"},
		{name: "incompatible owner", diffs: []diff.FileDiff{existingSkill}, files: map[string]string{"$P/.agents/skills/fixture-cx/SKILL.md": harnessCodexSkill},
			owners: []state.Item{codexOwner("pi", "fixture-cx", "local", "$P/.agents/skills/fixture-cx/SKILL.md", "", "CREATE", []byte(harnessCodexSkill))},
			want:   "codex destination $P/.agents/skills/fixture-cx/SKILL.md has incompatible owner fixture-cx (pi)"},
		{name: "incompatible owner wins over selected same-name conflict", diffs: []diff.FileDiff{codexSkillDiff()}, files: map[string]string{"$H/.agents/skills/fixture-cx/SKILL.md": harnessCodexSkill + "other\n"},
			owners: []state.Item{codexOwner("codex", "other-cx", "local", "$P/.agents/skills/fixture-cx/SKILL.md", "", "CREATE", nil)},
			want:   "codex destination $P/.agents/skills/fixture-cx/SKILL.md has incompatible owner other-cx (codex)"},
		{name: "discovery conflict wins over incompatible owner", diffs: []diff.FileDiff{existingSkill}, files: map[string]string{"$P/.agents/skills/fixture-cx/SKILL.md": harnessCodexSkill, "$H/.agents/skills/fixture-cx/SKILL.md": harnessCodexSkill + "other\n"},
			owners: []state.Item{codexOwner("codex", "other-cx", "local", "$P/.agents/skills/fixture-cx/SKILL.md", "", "CREATE", []byte(harnessCodexSkill))},
			want:   "codex skill name \"fixture-cx\" conflicts at $H/.agents/skills/fixture-cx/SKILL.md and $P/.agents/skills/fixture-cx/SKILL.md; incompatible same-name resources cannot be shadowed"},
		{name: "ownership drift", diffs: []diff.FileDiff{edited}, files: map[string]string{"$P/.agents/skills/fixture-cx/SKILL.md": harnessCodexSkill + "edited\n"},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/.agents/skills/fixture-cx/SKILL.md", "", "CREATE", []byte(harnessCodexSkill))},
			want:   "codex ownership drift at $P/.agents/skills/fixture-cx/SKILL.md; preserve edits, reconcile before mutation (no --force bypass)"},
		{name: "unknown ownership of existing file", diffs: []diff.FileDiff{existingSkill}, files: map[string]string{"$P/.agents/skills/fixture-cx/SKILL.md": harnessCodexSkill},
			want: "codex destination $P/.agents/skills/fixture-cx/SKILL.md has unknown ownership; explicit adoption/migration required"},
		{name: "owned unchanged skill admitted", diffs: []diff.FileDiff{existingSkill}, files: map[string]string{"$P/.agents/skills/fixture-cx/SKILL.md": harnessCodexSkill},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/.agents/skills/fixture-cx/SKILL.md", "", "CREATE", []byte(harnessCodexSkill))}},
		{name: "shadow file", diffs: []diff.FileDiff{codexSectionDiff(nil)}, files: map[string]string{"$P/AGENTS.override.md": "shadow\n"},
			want: "codex context $P/AGENTS.md is shadowed by $P/AGENTS.override.md; reconcile before mutation"},
		{name: "shadow wins over relocation", diffs: []diff.FileDiff{relocated}, files: map[string]string{"$P/AGENTS.override.md": "shadow\n"},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/docs/AGENTS.md", "fixture-cx", "APPEND", nil)},
			want:   "codex context $P/AGENTS.md is shadowed by $P/AGENTS.override.md; reconcile before mutation"},
		{name: "legacy ownership", diffs: []diff.FileDiff{codexSkillDiff()}, files: map[string]string{"$H/.codex/skills/fixture-cx/SKILL.md": harnessCodexSkill},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "global", "$H/.codex/skills/fixture-cx/SKILL.md", "", "CREATE", []byte(harnessCodexSkill))},
			want:   "codex legacy ownership at $H/.codex/skills/fixture-cx/SKILL.md requires explicit migration/retirement (patronus migrate codex-skills); automatic migration is Partial even if the old file is absent"},
		{name: "legacy ownership drift", diffs: []diff.FileDiff{codexSkillDiff()}, files: map[string]string{"$H/.codex/skills/fixture-cx/SKILL.md": harnessCodexSkill + "edited\n"},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "global", "$H/.codex/skills/fixture-cx/SKILL.md", "", "CREATE", []byte(harnessCodexSkill))},
			want:   "codex legacy ownership checksum drift at $H/.codex/skills/fixture-cx/SKILL.md; preserve edits, explicit migration required (no --force bypass)"},
		{name: "legacy discovered skill", diffs: []diff.FileDiff{codexSkillDiff()}, files: map[string]string{"$H/.codex/skills/fixture-cx/SKILL.md": harnessCodexSkill},
			want: "codex legacy skill $H/.codex/skills/fixture-cx/SKILL.md: automatic migration is Partial; preserve files and reconcile ownership/checksums and retire the old root explicitly before installing fixture-cx"},
		{name: "same-name conflict", diffs: []diff.FileDiff{codexSkillDiff()}, files: map[string]string{"$H/.agents/skills/fixture-cx/SKILL.md": harnessCodexSkill + "other\n"},
			want: "codex skill \"fixture-cx\" conflicts with $H/.agents/skills/fixture-cx/SKILL.md; incompatible same-name resource"},
		{name: "same-name identical bytes admitted", diffs: []diff.FileDiff{codexSkillDiff()}, files: map[string]string{"$H/.agents/skills/fixture-cx/SKILL.md": harnessCodexSkill}},
		{name: "section relocation", diffs: []diff.FileDiff{relocated},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/docs/AGENTS.md", "fixture-cx", "APPEND", nil)},
			want:   "codex context ownership at $P/docs/AGENTS.md cannot be silently relocated to $P/AGENTS.md"},
		{name: "mixed context", diffs: []diff.FileDiff{codexSectionDiff(ownedSection)}, files: map[string]string{"$P/AGENTS.md": string(ownedSection)},
			owners: []state.Item{codexOwner("pi", "fixture-pi", "local", "$P/AGENTS.md", "pi:fixture-pi", "APPEND", ownedSection)},
			want:   "codex mixed-context $P/AGENTS.md owned by fixture-pi (pi); explicit migration/consent required"},
		{name: "section drift", diffs: []diff.FileDiff{codexSectionDiff(ownedSection)}, files: map[string]string{"$P/AGENTS.md": string(ownedSection)},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", []byte("recorded differently\n"))},
			want:   "codex context ownership drift at $P/AGENTS.md; preserve edits before mutation"},
		{name: "ambiguous section ownership", diffs: []diff.FileDiff{codexSectionDiff(ownedSection)}, files: map[string]string{"$P/AGENTS.md": string(ownedSection)},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "CREATE", ownedSection)},
			want:   "codex context $P/AGENTS.md has ambiguous section ownership"},
		{name: "lacks verified composed inverse", diffs: []diff.FileDiff{restore}, files: map[string]string{"$P/AGENTS.md": string(siblingSection)},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", siblingSection), codexOwner("codex", "sibling-cx", "local", "$P/AGENTS.md", "sibling-cx", "APPEND", siblingSection)},
			want:   "codex context $P/AGENTS.md lacks a verified composed section inverse; retain sections and ownership"},
		{name: "unknown context ownership", diffs: []diff.FileDiff{codexSectionDiff([]byte("User prose\n"))}, files: map[string]string{"$P/AGENTS.md": "User prose\n"},
			want: "codex context $P/AGENTS.md has unknown ownership; preserve prose and obtain explicit migration/consent"},
		{name: "unknown section", diffs: []diff.FileDiff{codexSectionDiff(append(append([]byte(nil), ownedSection...), []byte("<!-- patronus:start unknown -->\nother\n<!-- patronus:end unknown -->\n")...))}, files: map[string]string{"$P/AGENTS.md": string(ownedSection) + "<!-- patronus:start unknown -->\nother\n<!-- patronus:end unknown -->\n"},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", append(append([]byte(nil), ownedSection...), []byte("<!-- patronus:start unknown -->\nother\n<!-- patronus:end unknown -->\n")...))},
			want:   "codex context $P/AGENTS.md: unknown, duplicate or nested section \"unknown\""},
		{name: "pi section in codex context", diffs: []diff.FileDiff{codexSectionDiff(append(append([]byte(nil), ownedSection...), []byte("<!-- patronus:start pi:fixture-pi -->\nother\n<!-- patronus:end pi:fixture-pi -->\n")...))}, files: map[string]string{"$P/AGENTS.md": string(ownedSection) + "<!-- patronus:start pi:fixture-pi -->\nother\n<!-- patronus:end pi:fixture-pi -->\n"},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", append(append([]byte(nil), ownedSection...), []byte("<!-- patronus:start pi:fixture-pi -->\nother\n<!-- patronus:end pi:fixture-pi -->\n")...))},
			want:   "codex context $P/AGENTS.md: unknown, duplicate or nested section \"pi:fixture-pi\""},
		{name: "duplicate section", diffs: []diff.FileDiff{codexSectionDiff(append(append([]byte(nil), ownedSection...), ownedSection...))}, files: map[string]string{"$P/AGENTS.md": string(ownedSection) + string(ownedSection)},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", append(append([]byte(nil), ownedSection...), ownedSection...))},
			want:   "codex context $P/AGENTS.md: unknown, duplicate or nested section \"fixture-cx\""},
		{name: "malformed section marker", diffs: []diff.FileDiff{codexSectionDiff([]byte("User prose\n<!-- patronus:start fixture-cx\nold\n"))}, files: map[string]string{"$P/AGENTS.md": "User prose\n<!-- patronus:start fixture-cx\nold\n"},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", []byte("User prose\n<!-- patronus:start fixture-cx\nold\n"))},
			want:   "codex context $P/AGENTS.md: malformed section marker"},
		{name: "missing section end", diffs: []diff.FileDiff{codexSectionDiff([]byte("User prose\n<!-- patronus:start fixture-cx -->\nold\n"))}, files: map[string]string{"$P/AGENTS.md": "User prose\n<!-- patronus:start fixture-cx -->\nold\n"},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", []byte("User prose\n<!-- patronus:start fixture-cx -->\nold\n"))},
			want:   "codex context $P/AGENTS.md: missing section end \"fixture-cx\""},
		{name: "unappend without selected section", diffs: []diff.FileDiff{unappendNoSection}, files: map[string]string{"$P/AGENTS.md": string(ownedSection)},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", ownedSection)},
			want:   "codex context $P/AGENTS.md lacks selected section identity"},
		{name: "ambiguous selected section identity", diffs: []diff.FileDiff{ambiguousSelected}, files: map[string]string{"$P/AGENTS.md": string(ownedSection)},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", ownedSection)},
			want:   "codex context $P/AGENTS.md has ambiguous selected section identity"},
		{name: "lacks verified selected section", diffs: []diff.FileDiff{unappendMissing}, files: map[string]string{"$P/AGENTS.md": string(unappendMissing.Before)},
			owners: []state.Item{codexOwner("codex", "sibling-cx", "local", "$P/AGENTS.md", "sibling-cx", "APPEND", unappendMissing.Before)},
			want:   "codex context $P/AGENTS.md lacks verified selected section fixture-cx"},
		{name: "inverse changes unselected prose", diffs: []diff.FileDiff{unappendChangesProse}, files: map[string]string{"$P/AGENTS.md": string(ownedSection)},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", ownedSection)},
			want:   "codex context $P/AGENTS.md inverse changes unselected sections or prose"},
		{name: "owned section append admitted", diffs: []diff.FileDiff{codexSectionDiff(ownedSection)}, files: map[string]string{"$P/AGENTS.md": string(ownedSection)},
			owners: []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", ownedSection)}},
		{name: "inverse setting foreign overlap", diffs: []diff.FileDiff{inverse}, files: map[string]string{"$P/.codex/config.toml": string(inverse.Before)},
			owners: []state.Item{settingOwner("fixture-cx", "codex", tomlSetting), settingOwner("foreign", "pi", tomlLeaf)},
			want:   "codex inverse ownership $P/.codex/config.toml: setting ownership conflict: fixture-cx (codex/local) at mcp_servers.fixture overlaps foreign (pi/local) at mcp_servers.fixture.url"},
		{name: "inverse setting ownership drift", diffs: []diff.FileDiff{inverseDrift}, files: map[string]string{"$P/.codex/config.toml": string(inverseDrift.Before)},
			owners: []state.Item{settingOwner("fixture-cx", "codex", tomlSetting), settingOwner("other-cx", "codex", tomlOther)},
			want:   "codex setting ownership drift at $P/.codex/config.toml (mcp_servers.fixture); no --force bypass"},
		{name: "inverse setting unparseable", diffs: []diff.FileDiff{inverseBroken}, files: map[string]string{"$P/.codex/config.toml": string(inverseBroken.Before)},
			owners: []state.Item{settingOwner("fixture-cx", "codex", tomlSetting)},
			want:   "codex inverse setting $P/.codex/config.toml: config: parse toml: toml: expected 'true'"},
		{name: "inverse setting admitted", diffs: []diff.FileDiff{inverse}, files: map[string]string{"$P/.codex/config.toml": string(inverse.Before)},
			owners: []state.Item{settingOwner("fixture-cx", "codex", tomlSetting), settingOwner("other-cx", "codex", tomlOther)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, project, res, _ := codexFixture(t)
			harnessWriteFiles(t, tc.files, home, project)
			harnessSaveOwners(t, tc.owners, home, project)
			cs := &diff.ChangeSet{Diffs: harnessExpandDiffs(tc.diffs, home, project)}
			got := harnessErr(codexPreflightPlan(cs, res, home, project))
			if want := harnessExpand(tc.want, home, project); got != want {
				t.Fatalf("codex golden\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

func piSkillDiff() diff.FileDiff {
	return diff.FileDiff{Path: "$P/.pi/skills/fixture-pi/SKILL.md", Action: diff.Create, Artifact: "fixture-pi", Type: "skill", Tool: "pi", Scope: "local", After: []byte(harnessPiSkill)}
}

func piSectionDiff(before []byte) diff.FileDiff {
	return diff.FileDiff{Path: "$P/AGENTS.md", Action: diff.Append, Artifact: "fixture-pi", Type: "instruction", Tool: "pi", Scope: "local", Section: &diff.SectionEdit{Name: "pi:fixture-pi", Body: []byte("new")}, Before: before, After: adapter.AppendSection(before, "pi:fixture-pi", []byte("new"))}
}

func TestHarnessGoldenPiPreflight(t *testing.T) {
	ownedSection := []byte("User prose\n\n<!-- patronus:start pi:fixture-pi -->\nold\n<!-- patronus:end pi:fixture-pi -->\n")
	composite := piSkillDiff()
	composite.Tool = "pi+codex"
	escape := piSkillDiff()
	escape.Path = "$P/elsewhere/fixture-pi/SKILL.md"
	conflict := piSkillDiff()
	conflict.Action = diff.Conflict
	conflict.Before = []byte("existing\n")
	existing := piSkillDiff()
	existing.Action = diff.Skip
	existing.Before = []byte(harnessPiSkill)
	agent := diff.FileDiff{Path: "$P/.pi/agents/fixture-pi.md", Action: diff.Create, Artifact: "fixture-pi", Type: "agent", Tool: "pi", Scope: "local", After: []byte("no frontmatter\n")}
	for _, tc := range []harnessGolden{
		{name: "composite cross-target destination", diffs: []diff.FileDiff{composite},
			want: "pi cross-target destination conflict $P/.pi/skills/fixture-pi/SKILL.md (pi+codex)"},
		{name: "root escape", diffs: []diff.FileDiff{escape},
			want: "pi destination $P/elsewhere/fixture-pi/SKILL.md escapes selected root $P/.pi"},
		{name: "root escape wins over resource conflict", diffs: []diff.FileDiff{escape}, files: map[string]string{"$H/.pi/agent/skills/fixture-pi/SKILL.md": harnessPiSkill},
			want: "pi destination $P/elsewhere/fixture-pi/SKILL.md escapes selected root $P/.pi"},
		{name: "agent validation", diffs: []diff.FileDiff{agent},
			want: "artifact \"fixture-pi\" target pi field document line 1: exact opening --- required; use the supported native shape"},
		{name: "same-name conflict", diffs: []diff.FileDiff{piSkillDiff()}, files: map[string]string{"$H/.pi/agent/skills/fixture-pi/SKILL.md": harnessPiSkill},
			want: "pi skill identity \"fixture-pi\" conflict: $H/.pi/agent/skills/fixture-pi/SKILL.md (native user) and $P/.pi/skills/fixture-pi/SKILL.md (selected local); silent shadowing refused"},
		{name: "resource conflict wins over unknown owner", diffs: []diff.FileDiff{conflict}, files: map[string]string{"$H/.pi/agent/skills/fixture-pi/SKILL.md": harnessPiSkill, "$P/.pi/skills/fixture-pi/SKILL.md": "existing\n"},
			want: "pi skill identity \"fixture-pi\" conflict: $H/.pi/agent/skills/fixture-pi/SKILL.md (native user) and $P/.pi/skills/fixture-pi/SKILL.md (selected local); silent shadowing refused"},
		{name: "incompatible owner", diffs: []diff.FileDiff{conflict}, files: map[string]string{"$P/.pi/skills/fixture-pi/SKILL.md": "existing\n"},
			owners: []state.Item{codexOwner("codex", "fixture-pi", "local", "$P/.pi/skills/fixture-pi/SKILL.md", "", "CREATE", []byte("existing\n"))},
			want:   "pi destination $P/.pi/skills/fixture-pi/SKILL.md owned by fixture-pi (codex), selected fixture-pi"},
		{name: "unknown owner refused only for conflicts", diffs: []diff.FileDiff{conflict}, files: map[string]string{"$P/.pi/skills/fixture-pi/SKILL.md": "existing\n"},
			want: "pi resource conflict at $P/.pi/skills/fixture-pi/SKILL.md: existing source and selected fixture-pi; resolve migration before applying selection"},
		{name: "unknown owner admitted when not a conflict", diffs: []diff.FileDiff{existing}, files: map[string]string{"$P/.pi/skills/fixture-pi/SKILL.md": harnessPiSkill}},
		{name: "owned unchanged conflict admitted", diffs: []diff.FileDiff{conflict}, files: map[string]string{"$P/.pi/skills/fixture-pi/SKILL.md": "existing\n"},
			owners: []state.Item{codexOwner("pi", "fixture-pi", "local", "$P/.pi/skills/fixture-pi/SKILL.md", "", "CREATE", []byte("existing\n"))}},
		{name: "shadow file", diffs: []diff.FileDiff{piSectionDiff(nil)}, files: map[string]string{"$P/AGENTS.override.md": "shadow\n"},
			want: "pi context $P/AGENTS.md is shadowed by $P/AGENTS.override.md"},
		{name: "shadow wins over relocation", diffs: []diff.FileDiff{piSectionDiff(nil)}, files: map[string]string{"$P/AGENTS.override.md": "shadow\n"},
			owners: []state.Item{codexOwner("pi", "fixture-pi", "local", "$P/docs/AGENTS.md", "pi:fixture-pi", "APPEND", nil)},
			want:   "pi context $P/AGENTS.md is shadowed by $P/AGENTS.override.md"},
		{name: "section relocation", diffs: []diff.FileDiff{piSectionDiff(nil)},
			owners: []state.Item{codexOwner("pi", "fixture-pi", "local", "$P/docs/AGENTS.md", "pi:fixture-pi", "APPEND", nil)},
			want:   "pi context ownership at $P/docs/AGENTS.md would be shadowed/relocated to $P/AGENTS.md"},
		{name: "section drift", diffs: []diff.FileDiff{piSectionDiff(ownedSection)}, files: map[string]string{"$P/AGENTS.md": string(ownedSection)},
			owners: []state.Item{codexOwner("pi", "fixture-pi", "local", "$P/AGENTS.md", "pi:fixture-pi", "APPEND", []byte("recorded differently\n"))},
			want:   "pi context drift at $P/AGENTS.md section pi:fixture-pi: recorded bytes differ; preserve edits and reconcile before apply"},
		{name: "unknown section", diffs: []diff.FileDiff{piSectionDiff(ownedSection)}, files: map[string]string{"$P/AGENTS.md": string(ownedSection)},
			want: "pi context $P/AGENTS.md section pi:fixture-pi has unknown ownership; refuse replacement"},
		{name: "owned section append admitted", diffs: []diff.FileDiff{piSectionDiff(ownedSection)}, files: map[string]string{"$P/AGENTS.md": string(ownedSection)},
			owners: []state.Item{codexOwner("pi", "fixture-pi", "local", "$P/AGENTS.md", "pi:fixture-pi", "APPEND", ownedSection)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, project, res := piFixture(t)
			harnessWriteFiles(t, tc.files, home, project)
			harnessSaveOwners(t, tc.owners, home, project)
			cs := &diff.ChangeSet{Diffs: harnessExpandDiffs(tc.diffs, home, project)}
			_, err := piPreflightPlan(cs, res, home, project)
			if got, want := harnessErr(err), harnessExpand(tc.want, home, project); got != want {
				t.Fatalf("pi golden\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

// Pi mixed-context review: a foreign owner at the context path yields a review
// (and no error); a CLAUDE.md source in the same directory marks the context mixed.
func TestHarnessGoldenPiMixedContextReview(t *testing.T) {
	home, project, res := piFixture(t)
	before := []byte("Shared prose\n")
	codexWrite(t, filepath.Join(project, "AGENTS.md"), before)
	harnessSaveOwners(t, []state.Item{codexOwner("codex", "fixture-cx", "local", "$P/AGENTS.md", "fixture-cx", "APPEND", before)}, home, project)
	d := piSectionDiff(before)
	cs := &diff.ChangeSet{Diffs: harnessExpandDiffs([]diff.FileDiff{d}, home, project)}
	reviews, err := piPreflightPlan(cs, res, home, project)
	if err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 1 {
		t.Fatalf("reviews: %+v", reviews)
	}
	r := reviews[0]
	if r.Consent.Path != filepath.Join(project, "AGENTS.md") || r.Consent.PiSource != filepath.Join(project, "AGENTS.md") {
		t.Fatalf("review binding: %+v", r.Consent)
	}
	if want := filepath.Join(project, "CLAUDE.md") + " (absent; no effective canonical Claude source found)"; r.Consent.ClaudeSource != want {
		t.Fatalf("claude source %q want %q", r.Consent.ClaudeSource, want)
	}
	if len(r.OtherOwners) != 1 || r.OtherOwners[0] != "fixture-cx (codex/local) at "+filepath.Join(project, "AGENTS.md") {
		t.Fatalf("other owners: %q", r.OtherOwners)
	}
	if string(r.Proposed) != string(cs.Diffs[0].After) {
		t.Fatal("proposed bytes differ from After")
	}
}

// Pi mcp-adapter.json collision through DiscoverPiMCP over existing and
// proposed sources; the proposed-source check reports the normalized duplicate.
func TestHarnessGoldenPiMCPCollision(t *testing.T) {
	home, project, res := piFixture(t)
	path := filepath.Join(home, ".pi/agent/mcp-adapter.json")
	prior := `{"mcpServers":{"fixture-name":{"url":"http://127.0.0.1/mcp"}}}`
	after := `{"mcpServers":{"fixture-name":{"url":"http://127.0.0.1/mcp"},"fixture_name":{"url":"http://127.0.0.1/mcp"}}}`
	codexWrite(t, path, []byte(prior))
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Artifact: "fixture_name", Tool: "pi", Scope: "global", Path: path, Action: diff.Merge, Before: []byte(prior), After: []byte(after)}}}
	_, err := piPreflightPlan(cs, res, home, project)
	want := "pi MCP duplicate normalized server \"fixture_name\": " + path + " and " + path
	if got := harnessErr(err); got != want {
		t.Fatalf("mcp collision\n got: %s\nwant: %s", got, want)
	}
}

// PiRootOverride failure: the warning is appended before the error is returned
// and the planner prints nothing itself.
func TestHarnessGoldenPiRootOverrideFailure(t *testing.T) {
	home, project, res := piFixture(t)
	t.Setenv("PI_CODING_AGENT_DIR", "file://"+filepath.Join(home, ".pi/agent"))
	cs := &diff.ChangeSet{Diffs: harnessExpandDiffs([]diff.FileDiff{piSkillDiff()}, home, project)}
	_, err := piPreflightPlan(cs, res, home, project)
	if got, want := harnessErr(err), "pi root override: whitespace or URI spelling has unqualified host/MCP path semantics"; got != want {
		t.Fatalf("root override\n got: %s\nwant: %s", got, want)
	}
	if want := "Pi static inventory only: checks cover inspected files, configuration and declared resources; extension callbacks and runtime-only discovery/registrations remain runtime-unverified. Cold-start/reload qualification is separate."; cs.Diffs[0].Warning != want {
		t.Fatalf("warning not appended before failure: %q", cs.Diffs[0].Warning)
	}
}

// Mutation safe-path gate (HP-03b): today only codex and pi diffs are checked;
// a claude or opencode diff at the same unsafe (symlinked) path is not refused.
func TestHarnessMutationSafePathGate(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	m, err := beginMutation(home, project)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { var cerr error; m.close(&cerr) })
	real := filepath.Join(project, "real")
	if err := os.Mkdir(real, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(project, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	unsafe := filepath.Join(link, "file.md")
	for _, tc := range []struct {
		tool string
		want string
	}{
		{"codex", "codex admission: pi path " + link + ": unapproved symlink/redirection"},
		{"pi", "pi path " + link + ": unapproved symlink/redirection"},
		{"claude", ""},
		{"opencode", ""},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			d := diff.FileDiff{Path: unsafe, Action: diff.Create, Artifact: "fixture", Tool: tc.tool, Scope: "local", After: []byte("x")}
			if got := harnessErr(m.checkFile(d, nil)); got != tc.want {
				t.Fatalf("checkFile(%s)\n got: %s\nwant: %s", tc.tool, got, tc.want)
			}
		})
	}
}

// Owner-load wrapping (HP-06): each caller of the two-scope state load wraps an
// unreadable .patronus/state.json differently.
func TestHarnessOwnerLoadWrapping(t *testing.T) {
	t.Run("codexPreflightPlan", func(t *testing.T) {
		home, project, res, cs := codexFixture(t)
		codexWrite(t, filepath.Join(project, ".patronus/state.json"), []byte("{broken"))
		err := codexPreflightPlan(cs, res, home, project)
		if got := harnessErr(err); !strings.HasPrefix(got, "codex ownership "+filepath.Join(project, ".patronus/state.json")+": ") {
			t.Fatalf("codex wrap: %q", got)
		}
	})
	t.Run("piPreflightPlan", func(t *testing.T) {
		home, project, res := piFixture(t)
		codexWrite(t, filepath.Join(project, ".patronus/state.json"), []byte("{broken"))
		cs := &diff.ChangeSet{Diffs: harnessExpandDiffs([]diff.FileDiff{piSkillDiff()}, home, project)}
		_, err := piPreflightPlan(cs, res, home, project)
		got := harnessErr(err)
		if got == "" || strings.HasPrefix(got, "pi ") || strings.HasPrefix(got, "codex ") || strings.HasPrefix(got, "load ") || strings.HasPrefix(got, "read ") {
			t.Fatalf("pi load error is returned unwrapped today: %q", got)
		}
		if _, direct := state.Load(filepath.Join(project, ".patronus/state.json")); direct == nil || direct.Error() != got {
			t.Fatalf("pi wrap differs from state.Load: %q vs %v", got, direct)
		}
	})
	t.Run("computePlan", func(t *testing.T) {
		home, project, _ := piFixture(t)
		root := fixtureCatalog(t)
		t.Chdir(root)
		codexWrite(t, filepath.Join(home, ".patronus/state.json"), []byte("{broken"))
		_, _, err := runInstall(t, "fix-setting", "--target", "claude", "--global")
		if got := harnessErr(err); !strings.HasPrefix(got, "read setting ownership in "+home+": ") {
			t.Fatalf("computePlan wrap: %q", got)
		}
		_ = project
	})
	t.Run("runDeployLocked", func(t *testing.T) {
		home, project, res := piFixture(t)
		codexWrite(t, filepath.Join(home, ".patronus/state.json"), []byte("{broken"))
		m, err := beginMutation(home, project)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { var cerr error; m.close(&cerr) })
		cs := &diff.ChangeSet{Diffs: harnessExpandDiffs([]diff.FileDiff{piSkillDiff()}, home, project)}
		cmd := newInstallCmd(testRegistry(t))
		err = runDeployLocked(cmd, cs, res, deployOptions{Harness: testRegistry(t), home: home, projectDir: project, mutation: m, target: "pi"}, &fakeRunner{})
		if got := harnessErr(err); !strings.HasPrefix(got, "load global state: ") {
			t.Fatalf("runDeployLocked wrap: %q", got)
		}
	})
}

// remove.go allowlist sites (629, 638, 664). Partial undo: a pi item with an
// unsettled (checksumless, retained) effect forgets its settled effects and
// reports an unresolved conflict; a codex item's drift is refused by preflight
// before settlement; a legacy lane item (claude, opencode) keeps every row and
// reports no error. Complete undo retires the row for every tool.
func TestHarnessRemoveAllowlistSites(t *testing.T) {
	const drift = "codex ownership drift at $P/.pi/skills/fixture-item/drifted.md; preserve edits, reconcile before mutation (no --force bypass)"
	for _, tc := range []struct {
		tool, secondChecksum, wantErr, wantPaths string
	}{
		{"codex", codexChecksum([]byte("recorded\n")), drift, "clean.md,drifted.md"},
		{"pi", "", "removal conflict unresolved: fixture-item (pi/local); ownership retained; verified committed=[\"DELETE $P/.pi/skills/fixture-item/clean.md\"] skipped=[\"SKIP $P/.pi/skills/fixture-item/drifted.md\"] failed=none; re-preview current bytes before repair", "drifted.md"},
		{"claude", codexChecksum([]byte("recorded\n")), "", "clean.md,drifted.md"},
		{"opencode", codexChecksum([]byte("recorded\n")), "", "clean.md,drifted.md"},
	} {
		t.Run(tc.tool+"/partial", func(t *testing.T) {
			home, project := t.TempDir(), fixtureCatalog(t)
			t.Setenv("HOME", home)
			t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, ".pi", "agent"))
			t.Chdir(project)
			clean := filepath.Join(project, ".pi/skills/fixture-item/clean.md")
			drifted := filepath.Join(project, ".pi/skills/fixture-item/drifted.md")
			codexWrite(t, clean, []byte("clean\n"))
			codexWrite(t, drifted, []byte("edited\n"))
			s := &state.State{Version: state.Version, Items: []state.Item{{Artifact: "fixture-item", Type: "skill", Tool: tc.tool, Scope: "local", Root: filepath.Join(project, ".pi"), Files: []state.FileState{
				{Path: clean, Checksum: codexChecksum([]byte("clean\n")), Action: "CREATE"},
				{Path: drifted, Checksum: tc.secondChecksum, Action: "CREATE"},
			}}}}
			if err := state.Save(filepath.Join(project, ".patronus/state.json"), s); err != nil {
				t.Fatal(err)
			}
			_, _, err := execRemove(t, "fixture-item", "--local", "--deploy")
			if got, want := harnessErr(err), harnessExpand(tc.wantErr, home, project); got != want {
				t.Fatalf("partial remove %s\n got: %s\nwant: %s", tc.tool, got, want)
			}
			after, err := state.Load(filepath.Join(project, ".patronus/state.json"))
			if err != nil {
				t.Fatal(err)
			}
			var paths []string
			for _, it := range after.Items {
				for _, f := range it.Files {
					paths = append(paths, filepath.Base(f.Path))
				}
			}
			if got := strings.Join(paths, ","); got != tc.wantPaths {
				t.Fatalf("retained paths %s: %q want %q", tc.tool, got, tc.wantPaths)
			}
		})
		t.Run(tc.tool+"/complete", func(t *testing.T) {
			home, project := t.TempDir(), fixtureCatalog(t)
			t.Setenv("HOME", home)
			t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, ".pi", "agent"))
			t.Chdir(project)
			clean := filepath.Join(project, ".pi/skills/fixture-item/clean.md")
			codexWrite(t, clean, []byte("clean\n"))
			s := &state.State{Version: state.Version, Items: []state.Item{{Artifact: "fixture-item", Type: "skill", Tool: tc.tool, Scope: "local", Root: filepath.Join(project, ".pi"), Files: []state.FileState{{Path: clean, Checksum: codexChecksum([]byte("clean\n")), Action: "CREATE"}}}}}
			if err := state.Save(filepath.Join(project, ".patronus/state.json"), s); err != nil {
				t.Fatal(err)
			}
			if _, _, err := execRemove(t, "fixture-item", "--local", "--deploy"); err != nil {
				t.Fatalf("complete remove %s: %v", tc.tool, err)
			}
			after, err := state.Load(filepath.Join(project, ".patronus/state.json"))
			if err != nil {
				t.Fatal(err)
			}
			if len(after.Items) != 0 {
				t.Fatalf("row retained after complete removal (%s): %+v", tc.tool, after.Items)
			}
		})
	}
}
