package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// openCodeFixture is a project with a .git marker, a resolver and a change set
// holding one invented OpenCode skill CREATE.
func openCodeFixture(t *testing.T) (home, project string, res toolpath.Resolver, cs *diff.ChangeSet) {
	t.Helper()
	home, project = t.TempDir(), t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	if err := os.Mkdir(filepath.Join(project, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	res = toolpath.New(os.LookupEnv, home, project)
	cs = &diff.ChangeSet{Diffs: []diff.FileDiff{{Path: filepath.Join(project, ".opencode/skills/fixture-oc/SKILL.md"), Action: diff.Create, Artifact: "fixture-oc", Type: "skill", Tool: "opencode", Scope: "local", After: []byte("---\nname: fixture-oc\ndescription: invented OpenCode test\n---\nfixture\n")}}}
	return home, project, res, cs
}

// admitOpenCode runs the shared admission for the opencode descriptor with the
// command's owner loader, collecting warnings.
func admitOpenCode(t *testing.T, cs *diff.ChangeSet, res toolpath.Resolver, home, project string) (error, []string) {
	t.Helper()
	reg := testRegistry(t)
	d, ok := reg.Lookup("opencode")
	if !ok {
		t.Fatal("opencode descriptor missing")
	}
	var warned []string
	_, err := admitTarget(reg, d, cs, res, home, project, func(m string) { warned = append(warned, m) })
	for _, fd := range cs.Diffs {
		if fd.Tool == "opencode" && fd.Warning != "" {
			warned = append(warned, strings.Split(fd.Warning, "\n")...)
		}
	}
	return err, warned
}

// TestOpenCodeDescriptorBindsProfileThroughRegistry: core-profile-oc binds to
// the opencode target through the one registry rule with the historical
// text; no OpenCode-only binding function exists; the descriptor is a valid
// registry member with roots and a validator.
func TestOpenCodeDescriptorBindsProfileThroughRegistry(t *testing.T) {
	reg := testRegistry(t)
	if err := reg.ProfileTarget("core-profile-oc", []string{"opencode"}, "opencode"); err != nil {
		t.Fatalf("opencode refused: %v", err)
	}
	for _, target := range []string{"claude", "codex", "pi", "all", ""} {
		err := reg.ProfileTarget("core-profile-oc", []string{"opencode"}, target)
		if want := "core-profile-oc requires --target opencode (got \"" + target + "\")"; harnessErr(err) != want {
			t.Fatalf("target %q: %q", target, harnessErr(err))
		}
	}
	d, _ := reg.Lookup("opencode")
	if d.SkillRoots == nil || d.Validator == nil || d.ResourcePhase != harness.ResourceAfterContext || !d.AdmitPlan || !d.AdmitRemoval || d.SkillMask.IsZero() {
		t.Fatalf("descriptor not filled: %+v", d)
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok && (fn.Name.Name == "opencodeProfileTarget" || fn.Name.Name == "opencodePreflightPlan") {
					t.Fatalf("OpenCode-only dispatch function %s exists", fn.Name.Name)
				}
				return true
			})
		}
	}
	if _, err := os.Stat("opencode_preflight.go"); !os.IsNotExist(err) {
		t.Fatal("opencode_preflight.go exists")
	}
}

// TestOpenCodeValidatorEnv (P-OC1 as decided): OPENCODE_CONFIG stops admission
// when the selection writes opencode.json and warns otherwise;
// OPENCODE_CONFIG_CONTENT warns; the static placement advisory (descriptor
// data) always flows through the shared warning channel, first.
func TestOpenCodeValidatorEnv(t *testing.T) {
	const placement = "OpenCode static placement only: skill loading, plugin activation, permission effects and fresh-session behaviour remain runtime-unverified"
	settingsDiff := func(project string) diff.FileDiff {
		return diff.FileDiff{Path: filepath.Join(project, "opencode.json"), Action: diff.Merge, Artifact: "fixture-setting-oc", Type: "setting", Tool: "opencode", Scope: "local", Before: []byte("{}\n"), After: []byte("{\"x\":1}\n"), Setting: &diff.SettingEdit{Target: diff.FileTargetRef{File: "opencode.json", Format: "jsonc"}, Dotted: "x", ScalarValue: 1}}
	}
	for _, tc := range []struct {
		name     string
		env      map[string]string
		settings bool
		wantErr  string
		wantWarn []string
	}{
		{"clean", nil, false, "", []string{placement}},
		{"OPENCODE_CONFIG with a settings write", map[string]string{"OPENCODE_CONFIG": "/invented/override.json"}, true, "opencode: OPENCODE_CONFIG is set; Patronus would merge into a file OpenCode does not read; unset it or point Patronus at the override", nil},
		{"OPENCODE_CONFIG with a skills-only selection", map[string]string{"OPENCODE_CONFIG": "/invented/override.json"}, false, "", []string{placement, "opencode: OPENCODE_CONFIG is set; placed settings are not read while it is in effect"}},
		{"OPENCODE_CONFIG_CONTENT", map[string]string{"OPENCODE_CONFIG_CONTENT": "{\"invented\":true}"}, false, "", []string{placement, "opencode: OPENCODE_CONFIG_CONTENT is set; inline config overrides placed settings"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, project, res, cs := openCodeFixture(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if tc.settings {
				cs.Diffs = append(cs.Diffs, settingsDiff(project))
			}
			err, warned := admitOpenCode(t, cs, res, home, project)
			if got := harnessErr(err); got != tc.wantErr {
				t.Fatalf("error\n got: %s\nwant: %s", got, tc.wantErr)
			}
			if got, want := strings.Join(warned, "|"), strings.Join(tc.wantWarn, "|"); got != want {
				t.Fatalf("warnings\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

// TestOpenCodeValidatorRefusesCommentedConfig (OP-09): any write into an
// opencode.json(c) whose current bytes carry JSONC comments is refused, keyed on
// the target file, so Merge, Restore and a forced Delete alike; a comment-free
// file is admitted with identical bytes.
func TestOpenCodeValidatorRefusesCommentedConfig(t *testing.T) {
	for _, tc := range []struct {
		name   string
		before string
		action diff.Action
		want   string
	}{
		{"merge over line comment", "{\n  // keep me\n  \"x\": 1\n}\n", diff.Merge, "refused"},
		{"restore over block comment", "{ /* block */ \"x\": 1 }\n", diff.Restore, "refused"},
		{"forced delete over comment", "// top\n{\"x\": 1}\n", diff.Delete, "refused"},
		{"merge over comment-free file", "{\"x\": 1}\n", diff.Merge, ""},
		{"url string with slashes is not a comment", "{\"url\": \"https://invented.example/mcp\"}\n", diff.Merge, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, project, res, _ := openCodeFixture(t)
			path := filepath.Join(project, "opencode.json")
			codexWrite(t, path, []byte(tc.before))
			fd := diff.FileDiff{Path: path, Action: tc.action, Artifact: "fixture-setting-oc", Type: "setting", Tool: "opencode", Scope: "local", Before: []byte(tc.before), After: []byte("{\"x\": 2}\n")}
			if tc.action == diff.Merge {
				fd.Setting = &diff.SettingEdit{Target: diff.FileTargetRef{File: "opencode.json", Format: "jsonc"}, Dotted: "x", ScalarValue: 2, PriorValue: 1, PriorPresent: true}
			}
			err, _ := admitOpenCode(t, &diff.ChangeSet{Diffs: []diff.FileDiff{fd}}, res, home, project)
			got := harnessErr(err)
			want := ""
			if tc.want != "" {
				want = "opencode: " + path + " carries JSONC comments the structured writer cannot preserve; remove the comments or edit by hand"
			}
			if got != want {
				t.Fatalf("admission\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

// TestOpenCodeHarnessOwnedFilesNeverClaimed: OpenCode writes
// .opencode/{package.json,package-lock.json,node_modules,.gitignore} for local
// plugins; their presence never blocks an unrelated install and a selection
// naming one is refused.
func TestOpenCodeHarnessOwnedFilesNeverClaimed(t *testing.T) {
	home, project, res, cs := openCodeFixture(t)
	for _, rel := range []string{".opencode/package.json", ".opencode/package-lock.json", ".opencode/node_modules/x/index.js", ".opencode/.gitignore"} {
		codexWrite(t, filepath.Join(project, rel), []byte("harness-owned\n"))
	}
	if err, _ := admitOpenCode(t, cs, res, home, project); err != nil {
		t.Fatalf("unrelated skill blocked: %v", err)
	}
	for _, rel := range []string{".opencode/package.json", ".opencode/node_modules/x/index.js", ".opencode/.gitignore"} {
		path := filepath.Join(project, rel)
		claim := &diff.ChangeSet{Diffs: []diff.FileDiff{{Path: path, Action: diff.Create, Artifact: "fixture-oc", Type: "skill", Tool: "opencode", Scope: "local", After: []byte("x")}}}
		err, _ := admitOpenCode(t, claim, res, home, project)
		if want := "opencode: " + path + " is written by OpenCode for local plugins; Patronus never claims it"; harnessErr(err) != want {
			t.Fatalf("%s: %q", rel, harnessErr(err))
		}
	}
}

// TestOpenCodeLaneIsAdmitted: with the descriptor filled, the shared admission
// refuses for opencode what the legacy lane let through: an unknown unowned
// destination, a context file whose recorded section belongs to another
// target (mixed context, no consent modelled), and a composite destination
// (the crossTargetMessage default branch names the path and the opencode
// prefix; G3).
func TestOpenCodeLaneIsAdmitted(t *testing.T) {
	t.Run("conflict", func(t *testing.T) {
		home, _, _ := harnessOutputFixture(t)
		dest := filepath.Join(home, ".config/opencode/skills/fixture-any/SKILL.md")
		codexWrite(t, dest, []byte("---\nname: fixture-any\ndescription: edited by the user\n---\nOther body\n"))
		codexWrite(t, filepath.Join(home, ".patronus/state.json"), []byte(`{"version":2,"items":[]}`))
		_, _, err := runInstall(t, "fixture-any", "--target", "opencode", "--global")
		if want := "opencode destination " + dest + " has unknown ownership; explicit adoption/migration required"; harnessErr(err) != want {
			t.Fatalf("conflict\n got: %s\nwant: %s", harnessErr(err), want)
		}
	})
	t.Run("recorded other-target section", func(t *testing.T) {
		home, _, _ := harnessOutputFixture(t)
		context := filepath.Join(home, ".config/opencode/AGENTS.md")
		existing := adapter.AppendSection([]byte("User prose\n"), "fixture-ctx", []byte("old"))
		codexWrite(t, context, existing)
		seed := &state.State{Version: state.Version, Items: []state.Item{{Artifact: "fixture-ctx", Type: "instruction", Tool: "codex", Scope: "global", Files: []state.FileState{{Path: context, Action: "APPEND", Checksum: state.Checksum(existing), Section: "fixture-ctx", Prior: []byte("User prose\n")}}}}}
		if err := state.Save(filepath.Join(home, ".patronus/state.json"), seed); err != nil {
			t.Fatal(err)
		}
		_, _, err := runInstall(t, "fixture-ctx", "--target", "opencode", "--global")
		if want := "opencode mixed-context " + context + " owned by fixture-ctx (codex); explicit migration/consent required"; harnessErr(err) != want {
			t.Fatalf("mixed context\n got: %s\nwant: %s", harnessErr(err), want)
		}
	})
	t.Run("composite destination (G3 default branch)", func(t *testing.T) {
		home, project, res, _ := openCodeFixture(t)
		context := filepath.Join(project, "AGENTS.md")
		cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Path: context, Action: diff.Append, Artifact: "fixture-ctx", Type: "instruction", Tool: "opencode+codex", Scope: "local", Section: &diff.SectionEdit{Name: "oc:fixture-ctx"}, After: []byte("x")}}}
		err, _ := admitOpenCode(t, cs, res, home, project)
		if want := "opencode cross-target destination conflict " + context + " (opencode+codex)"; harnessErr(err) != want {
			t.Fatalf("composite\n got: %s\nwant: %s", harnessErr(err), want)
		}
	})
}
