package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/spf13/cobra"
)

// The production registry: four descriptors with their behaviour as flags,
// Targeted() naming the lock-targeted trio, and each descriptor's native skill
// roots agreeing with the adapter it mirrors.
func TestDefaultRegistryDescriptors(t *testing.T) {
	reg := testRegistry(t)
	type flags struct {
		itemSuffix, contextFile, sectionPrefix, globalMarker, admissionPrefix, errorPrefix string
		profiles, shadowFiles                                                              string
		resourcePhase                                                                      harness.ResourcePhase
		lockTargeted, lockCoexistence, ownsEffects, strictUnowned, strictSections          bool
		safePathCheck, mixedContextConsent, roots, validator                               bool
	}
	for _, tc := range []struct {
		name string
		want flags
	}{
		{"codex", flags{"cx", "AGENTS.md", "", "~/.codex", "codex admission", "codex", "core-profile-cx", "AGENTS.override.md", harness.ResourceAfterContext, true, true, true, true, true, true, false, true, true}},
		{"pi", flags{"pi", "AGENTS.md", "pi:", "~/.pi/agent", "", "pi", "core-profile-pi", "", harness.ResourceBeforeOwnership, true, false, true, false, false, true, true, true, true}},
		{"claude", flags{"cl", "CLAUDE.md", "", "~/.claude", "", "", "", "", harness.ResourceAfterContext, false, false, false, false, false, false, false, true, false}},
		{"opencode", flags{"oc", "AGENTS.md", "oc:", "~/.config/opencode", "opencode admission", "opencode", "core-profile-oc", "", harness.ResourceAfterContext, true, true, true, true, false, true, false, true, true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, ok := reg.Lookup(tc.name)
			if !ok {
				t.Fatalf("descriptor %s missing", tc.name)
			}
			got := flags{d.ItemSuffix, d.ContextFile, d.SectionPrefix, d.GlobalMarker, d.AdmissionPrefix, d.ErrorPrefix, strings.Join(d.Profiles, ","), strings.Join(d.ShadowFiles, ","), d.ResourcePhase, d.LockTargeted, d.LockCoexistence, d.OwnsEffects, d.StrictUnowned, d.StrictSections, d.SafePathCheck, d.MixedContextConsent, d.SkillRoots != nil, d.Validator != nil}
			if got != tc.want {
				t.Fatalf("descriptor %s\n got: %+v\nwant: %+v", tc.name, got, tc.want)
			}
		})
	}
	var targeted []string
	for _, d := range reg.Targeted() {
		targeted = append(targeted, d.Name)
	}
	if strings.Join(targeted, ",") != "codex,pi,opencode" {
		t.Fatalf("Targeted = %v", targeted)
	}
	if err := reg.ValidateTarget("invented"); err == nil || err.Error() != "unknown target \"invented\"; use claude|codex|opencode|pi|all" {
		t.Fatalf("ValidateTarget text: %v", err)
	}
}

// HS-O1: adapters stay the single source of paths. Every skill directory the
// adapter layout names is one of the descriptor's native (non-legacy) roots.
func TestDefaultRegistryRootsAgreeWithAdapters(t *testing.T) {
	reg := testRegistry(t)
	home, project := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, ".pi", "agent"))
	t.Setenv("PI_PACKAGE_DIR", "")
	t.Setenv("PI_OFFLINE", "true")
	if err := os.Mkdir(filepath.Join(project, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	adapters, err := loadAdapters(filepath.Join(applicationSourceRoot(t), "adapters"))
	if err != nil {
		t.Fatal(err)
	}
	byTool := map[string]*manifest.Adapter{}
	for _, ad := range adapters {
		byTool[ad.Tool] = ad
	}
	for _, name := range []string{"codex", "pi", "claude", "opencode"} {
		t.Run(name, func(t *testing.T) {
			d, _ := reg.Lookup(name)
			ad, ok := byTool[name]
			if !ok || ad.Layout.Skill == nil {
				t.Fatalf("adapter %s has no skill layout", name)
			}
			config := filepath.Join(home, strings.TrimPrefix(d.GlobalMarker, "~/"))
			roots, err := d.SkillRoots(harness.RootInputs{Home: home, Project: project, Config: config, Env: os.LookupEnv})
			if err != nil {
				t.Fatal(err)
			}
			native := map[string]bool{}
			for _, r := range roots {
				if !r.Legacy {
					native[r.Path] = true
				}
			}
			wantGlobal := filepath.Join(home, strings.TrimPrefix(filepath.Dir(filepath.Dir(ad.Layout.Skill.Global.Path)), "~/"))
			wantProject := filepath.Join(project, filepath.Dir(filepath.Dir(ad.Layout.Skill.Project.Path)))
			for _, want := range []string{wantGlobal, wantProject} {
				if !native[want] {
					t.Fatalf("adapter root %s missing from %s native roots %v", want, name, roots)
				}
			}
			// P6: section naming and MCP ownership are adapter data; the descriptor
			// flags must agree with the adapter that carries them.
			prefix := ""
			if ad.Layout.Instruction != nil {
				prefix = ad.Layout.Instruction.SectionPrefix
			}
			if d.SectionPrefix != prefix {
				t.Fatalf("%s SectionPrefix %q disagrees with adapter %q", name, d.SectionPrefix, prefix)
			}
			// Leaf MCP ownership is adapter data consumed by the recipe engine; it
			// is independent of the descriptor's section strictness (Codex: leaf and
			// strict sections; OpenCode: leaf and lenient sections).
			if ad.Layout.Mcp != nil && ad.Layout.Mcp.Ownership != "" && ad.Layout.Mcp.Ownership != manifest.OwnershipLeaf {
				t.Fatalf("%s mcp ownership %q is not a known value", name, ad.Layout.Mcp.Ownership)
			}
		})
	}
}

// HP-13: a registry construction error fails every subcommand closed through
// PersistentPreRunE, with the error text intact.
func TestRootCommandSurfacesRegistryError(t *testing.T) {
	boom := errors.New("invented registry failure")
	root := newRootCmdWith(func() (*harness.Registry, error) { return nil, boom })
	for _, args := range [][]string{{"install", "fixture"}, {"lock", "--profile", "x"}, {"update", "x"}, {"remove", "x"}, {"list"}} {
		cmd := root
		cmd.SetArgs(args)
		cmd.SetOut(&strings.Builder{})
		cmd.SetErr(&strings.Builder{})
		if err := cmd.Execute(); !errors.Is(err, boom) {
			t.Fatalf("%v did not fail closed: %v", args, err)
		}
	}
	if newRootCmd().PersistentPreRunE(&cobra.Command{}, nil) != nil {
		t.Fatal("production registry failed to build")
	}
}
