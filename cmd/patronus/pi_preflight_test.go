package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/toolpath"
	"github.com/spf13/cobra"
)

func dp02File(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}
func dp02Artifact(t *testing.T, root, name, kind, body string) {
	t.Helper()
	dir := filepath.Join(root, "artifacts", name)
	entry := "entry.md"
	if kind == "skill" {
		entry = "SKILL.md"
	}
	dp02File(t, filepath.Join(dir, "patronus.yaml"), fmt.Sprintf("apiVersion: patronus/v2\nfamily: artifact\nname: %s\nversion: 1.0.0\ndescription: Invented fixture\nrole: capability\ntype: %s\ntargets: [pi]\nentry: %s\n", name, kind, entry))
	dp02File(t, filepath.Join(dir, entry), body)
}
func dp02Snapshot(t *testing.T, roots ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			out[path] = "mode:" + info.Mode().String()
			if !d.IsDir() {
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				out[path] += string(b)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		// The advisory coordination inode is not installed state. Ignore only
		// that protected file and otherwise-empty necessary parent directories.
		lockPath := filepath.Join(root, ".patronus/package-state/mutation.lock")
		if _, ok := out[lockPath]; ok {
			dp05AssertCoordinationLock(t, lockPath)
			delete(out, lockPath)
		}
		for _, dir := range []string{filepath.Dir(lockPath), filepath.Join(root, ".patronus")} {
			empty := true
			for path := range out {
				if path != dir && strings.HasPrefix(path, dir+string(os.PathSeparator)) {
					empty = false
				}
			}
			if empty {
				delete(out, dir)
			}
		}
	}
	return out
}
func dp02Setup(t *testing.T) directoryFixture {
	t.Helper()
	f := newDirectoryFixture(t)
	// Match the protected host coordination parent; state.Save otherwise makes
	// this directory 0755 and Acquire intentionally tightens it to 0700.
	if err := os.MkdirAll(filepath.Join(f.home, ".patronus"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("PI_PACKAGE_DIR", "")
	t.Setenv("PI_OFFLINE", "true")
	t.Setenv("PI_SUBAGENT_EXTRA_AGENT_DIRS", "")
	t.Setenv("PI_MCP_CONFIG_MODE", "")
	return f
}

func TestPiPreflightNativeMetadataCollisions(t *testing.T) {
	for _, tc := range []struct {
		name, kind, directory, body string
	}{
		{"fixture-role", "agent", "agents", "---\nname: fixture-role\ndescription: true\n---\nBody\n"},
		{"fixture-number", "agent", "agents", "---\nname: fixture-number\ndescription: 123\n---\nBody\n"},
		{"fixture-colon", "agent", "agents", "---\nname: fixture-colon\ndescription: Review: carefully\n---\nBody\n"},
		{"fixture.chain", "command", "prompts", "Prompt body\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := dp02Setup(t)
			dp02Artifact(t, f.root, tc.name, tc.kind, tc.body)
			dp02Artifact(t, f.root, "fresh-fixture", "skill", "---\nname: fresh-fixture\ndescription: Fixture\n---\nBody\n")
			dp02File(t, filepath.Join(f.root, "profiles/collision-fixture.yaml"), fmt.Sprintf("apiVersion: patronus/v2\nfamily: profile\nname: collision-fixture\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  capabilities: [fresh-fixture, %s]\n", tc.name))
			dp02File(t, filepath.Join(f.root, ".pi", tc.directory, tc.name+".md"), tc.body)
			dp02File(t, filepath.Join(f.home, ".patronus/state.json"), `{"version":1,"items":[]}`)
			dp02File(t, filepath.Join(f.root, "patronus.lock"), `{"version":2,"entries":[]}`)
			before := dp02Snapshot(t, f.home, f.root)
			_, _, err := runInstall(t, "--profile", "collision-fixture", "--target", "pi", "--global", "--deploy", "--force")
			if err == nil || !strings.Contains(err.Error(), "conflict") {
				t.Errorf("known static collision not reported: %v", err)
			}
			if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
				t.Fatal("collision changed destination/state/lock")
			}
		})
	}
}

func TestPiPreflightQuotedEmptyNativeMetadata(t *testing.T) {
	for _, metadata := range []string{
		"name: fixture-role\ndescription: \"\"\n  Existing description",
		"name: fixture-role\ndescription: ''\n  Existing description",
		"name: \"\"\n  fixture-role\ndescription: Existing description",
		"name: ''\n  fixture-role\ndescription: Existing description",
	} {
		t.Run(metadata, func(t *testing.T) {
			f := dp02Setup(t)
			dp02Artifact(t, f.root, "fixture-role", "agent", "---\nname: fixture-role\ndescription: Fixture\n---\nBody\n")
			dp02Artifact(t, f.root, "fresh-fixture", "skill", "---\nname: fresh-fixture\ndescription: Fixture\n---\nBody\n")
			dp02File(t, filepath.Join(f.root, "profiles/collision-fixture.yaml"), "apiVersion: patronus/v2\nfamily: profile\nname: collision-fixture\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  capabilities: [fresh-fixture, fixture-role]\n")
			dp02File(t, filepath.Join(f.root, ".pi/agents/fixture-role.md"), "---\n"+metadata+"\n---\nBody\n")
			dp02File(t, filepath.Join(f.home, ".patronus/state.json"), `{"version":1,"items":[]}`)
			dp02File(t, filepath.Join(f.root, "patronus.lock"), `{"version":2,"entries":[]}`)
			before := dp02Snapshot(t, f.home, f.root)
			_, _, err := runInstall(t, "--profile", "collision-fixture", "--target", "pi", "--global", "--deploy", "--force")
			if err == nil || !strings.Contains(err.Error(), "unsupported native metadata block") {
				t.Errorf("unsupported live native identity silently omitted: %v", err)
			}
			if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
				t.Fatal("unsupported identity changed destination/state/lock")
			}
		})
	}
}

func TestPiPreflightNativeScalarReinstall(t *testing.T) {
	for _, description := range []string{"true", "123", "Review: carefully"} {
		t.Run(description, func(t *testing.T) {
			f := dp02Setup(t)
			body := "---\nname: fixture-role\ndescription: " + description + "\n---\nBody\n"
			dp02Artifact(t, f.root, "fixture-role", "agent", body)
			for i := 0; i < 2; i++ {
				if _, _, err := runInstall(t, "fixture-role", "--target", "pi", "--global", "--deploy", "--yes"); err != nil {
					t.Fatalf("install %d: %v", i, err)
				}
			}
			if got := string(mustRead(t, filepath.Join(f.home, ".pi/agent/agents/fixture-role.md"))); got != body {
				t.Fatalf("native bytes changed: %q", got)
			}
		})
	}
}

func TestPiPreflightOwnedContextDrift(t *testing.T) {
	for _, flag := range []string{"--yes", "--force"} {
		t.Run(flag, func(t *testing.T) {
			f := dp02Setup(t)
			dp02Artifact(t, f.root, "context-fixture", "instruction", "Original contribution\n")
			if _, _, err := runInstall(t, "context-fixture", "--target", "pi", "--global", "--deploy", "--yes"); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(f.home, ".pi/agent/AGENTS.md")
			original := string(mustRead(t, path))
			edited := strings.Replace(original, "Original contribution", "Operator edited contribution", 1)
			if edited == original {
				t.Fatal("test did not edit installed section")
			}
			dp02File(t, path, edited)
			dp02Artifact(t, f.root, "fresh-fixture", "skill", "---\nname: fresh-fixture\ndescription: Fixture\n---\nBody\n")
			dp02File(t, filepath.Join(f.root, "profiles/drift-fixture.yaml"), "apiVersion: patronus/v2\nfamily: profile\nname: drift-fixture\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  capabilities: [fresh-fixture, context-fixture]\n")
			before := dp02Snapshot(t, f.home, f.root)
			_, _, err := runInstall(t, "--profile", "drift-fixture", "--target", "pi", "--global", "--deploy", flag)
			if err == nil || !strings.Contains(err.Error(), "drift") {
				t.Errorf("owned edited section not refused: %v", err)
			}
			if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
				t.Fatal("drift refusal changed destination/state/lock")
			}
		})
	}
}

func TestPiPreflightOwnedContextReinstall(t *testing.T) {
	f := dp02Setup(t)
	dp02Artifact(t, f.root, "context-fixture", "instruction", "Original contribution\n")
	path := filepath.Join(f.home, ".pi/agent/AGENTS.md")
	dp02File(t, path, "Existing operator prose\n")
	for i := 0; i < 2; i++ {
		if _, _, err := runInstall(t, "context-fixture", "--target", "pi", "--global", "--deploy", "--yes"); err != nil {
			t.Fatalf("install %d: %v", i, err)
		}
	}
	got := string(mustRead(t, path))
	if !strings.Contains(got, "Existing operator prose") || strings.Count(got, "Original contribution") != 1 {
		t.Fatalf("reinstall lost prose or duplicated contribution: %q", got)
	}
}

func TestPiPreflightInvalidNativeWholeProfile(t *testing.T) {
	f := dp02Setup(t)
	dp02Artifact(t, f.root, "valid-fixture", "skill", "---\nname: valid-fixture\ndescription: Fixture\n---\nBody\n")
	dp02Artifact(t, f.root, "invalid-fixture", "agent", "---\nname: invalid-fixture\ndescription: Fixture\ntools: helper.ts\n---\nBody\n")
	dp02File(t, filepath.Join(f.root, "profiles/selected-fixture.yaml"), "apiVersion: patronus/v2\nfamily: profile\nname: selected-fixture\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  capabilities: [valid-fixture, invalid-fixture]\n")
	dp02File(t, filepath.Join(f.home, ".patronus/state.json"), `{"version":1,"items":[]}`)
	dp02File(t, filepath.Join(f.root, "patronus.lock"), `{"version":2,"entries":[]}`)
	before := dp02Snapshot(t, f.home, f.root)
	_, _, err := runInstall(t, "--profile", "selected-fixture", "--target", "pi", "--global", "--deploy", "--force")
	if err == nil || !strings.Contains(err.Error(), "helper") && !strings.Contains(err.Error(), "executable module") {
		t.Fatalf("native error missing: %v", err)
	}
	if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
		t.Fatal("refusal changed destination/state/lock bytes or created files")
	}
}

func TestPiPreflightMixedWholeSelection(t *testing.T) {
	for _, flag := range []string{"", "--yes", "--force"} {
		t.Run(flag, func(t *testing.T) {
			f := dp02Setup(t)
			dp02Artifact(t, f.root, "fresh-fixture", "skill", "---\nname: fresh-fixture\ndescription: Fixture\n---\nBody\n")
			dp02Artifact(t, f.root, "context-fixture", "instruction", "Pi-specific instruction\n")
			dp02File(t, filepath.Join(f.root, "profiles/mixed-fixture.yaml"), "apiVersion: patronus/v2\nfamily: profile\nname: mixed-fixture\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  capabilities: [fresh-fixture, context-fixture]\n")
			prior := "Complete shared text\n\nClaude-specific section\n"
			dp02File(t, filepath.Join(f.home, ".pi/agent/CLAUDE.md"), prior)
			dp02File(t, filepath.Join(f.home, ".patronus/state.json"), `{"version":1,"items":[]}`)
			dp02File(t, filepath.Join(f.root, "patronus.lock"), `{"version":2,"entries":[]}`)
			before := dp02Snapshot(t, f.home, f.root)
			cmd := newInstallCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetIn(strings.NewReader("prepared combined file\ny\n"))
			args := []string{"--profile", "mixed-fixture", "--target", "pi", "--global", "--deploy"}
			if flag != "" {
				args = append(args, flag)
			}
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "mixed-context") {
				t.Fatalf("headless/flag consent admitted: %v\n%s", err, out.String())
			}
			if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
				t.Fatal("mixed refusal changed selected destination/state/lock")
			}
		})
	}
}

func TestPiContextConsent(t *testing.T) {
	f := dp02Setup(t)
	dp02Artifact(t, f.root, "context-fixture", "instruction", "Pi-specific contribution\n")
	prior := []byte("Operator-prepared complete shared instructions\n\nClaude-specific instructions\n")
	path := filepath.Join(f.root, "CLAUDE.md")
	dp02File(t, path, string(prior))
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	planned, err := planInstall(cmd, installPlanRequest{Names: []string{"context-fixture"}, Tool: "pi", Scope: "local", Home: f.home, ProjectDir: f.root})
	if err != nil {
		t.Fatal(err)
	}
	reviews, err := piPreflightPlan(planned.Changes, planned.Resolver, f.home, f.root)
	if err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 1 {
		t.Fatalf("mixed review missing: %+v", reviews)
	}
	review := reviews[0]
	if review.Consent.Path != path || review.Consent.PiSource != path || review.Consent.ClaudeSource != path || review.Consent.PriorHash != sha256.Sum256(prior) {
		t.Fatalf("wrong prepared prior/path/source binding: %+v", review)
	}
	if !bytes.Contains(review.Proposed, prior) || !bytes.Contains(review.Proposed, []byte("patronus:start pi:context-fixture")) {
		t.Fatalf("prepared text or Pi section lost: %s", review.Proposed)
	}
	var out bytes.Buffer
	consents, err := confirmPiContexts(reviews, true, false, false, strings.NewReader("prepared combined file\n"), &out)
	if err != nil || len(consents) != 1 {
		t.Fatalf("confirm: %+v %v", consents, err)
	}
	for _, part := range []string{path, "Pi effective source", "Claude effective source", fmt.Sprintf("%x", review.Consent.PriorHash), fmt.Sprintf("%x", review.Consent.ResultHash), "Pi-specific contribution"} {
		if !strings.Contains(out.String(), part) {
			t.Errorf("consent display missing %q", part)
		}
	}
	if err := validatePiConsentBytes(consents[0], path, prior, review.Proposed); err != nil {
		t.Fatal(err)
	}
	for _, args := range []struct {
		path          string
		prior, result []byte
	}{{path + ".other", prior, review.Proposed}, {path, []byte("changed"), review.Proposed}, {path, prior, []byte("changed result")}} {
		if err := validatePiConsentBytes(consents[0], args.path, args.prior, args.result); err == nil {
			t.Fatal("consent not bound to exact path/prior/result")
		}
	}
	for _, tc := range []struct {
		name                    string
		interactive, yes, force bool
		answer                  string
	}{{"decline", true, false, false, "no\n"}, {"ordinary yes", true, false, false, "yes\n"}, {"headless", false, false, false, "prepared combined file\n"}, {"yes flag", true, true, false, "prepared combined file\n"}, {"force", true, false, true, "prepared combined file\n"}} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := confirmPiContexts(reviews, tc.interactive, tc.yes, tc.force, strings.NewReader(tc.answer), &bytes.Buffer{}); err == nil {
				t.Fatal("invalid consent accepted")
			}
		})
	}
	if !bytes.Equal(mustRead(t, path), prior) {
		t.Fatal("preview/consent wrote prepared file")
	}
}

func TestPiContextGlobalClaudeProvenance(t *testing.T) {
	f := dp02Setup(t)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	dp02Artifact(t, f.root, "fixture-context", "instruction", "Pi contribution\n")
	piPath := filepath.Join(f.home, ".pi/agent/CLAUDE.md")
	dp02File(t, piPath, "Prepared Pi-root file\n")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	planned, err := planInstall(cmd, installPlanRequest{Names: []string{"fixture-context"}, Tool: "pi", Scope: "global", Home: f.home, ProjectDir: f.root})
	if err != nil {
		t.Fatal(err)
	}
	reviews, err := piPreflightPlan(planned.Changes, planned.Resolver, f.home, f.root)
	if err != nil {
		t.Fatal(err)
	}
	claudePath := filepath.Join(f.home, ".claude/CLAUDE.md")
	if len(reviews) != 1 || reviews[0].Consent.PiSource != piPath || !strings.Contains(reviews[0].Consent.ClaudeSource, claudePath) || !strings.Contains(reviews[0].Consent.ClaudeSource, "absent") {
		t.Fatalf("wrong global provenance: %+v", reviews)
	}
	dp02File(t, claudePath, "Separate Claude instructions\n")
	reviews, err = piPreflightPlan(planned.Changes, planned.Resolver, f.home, f.root)
	if err != nil || reviews[0].Consent.ClaudeSource != claudePath {
		t.Fatalf("separate Claude source not reported: %+v %v", reviews, err)
	}
	if err := os.Chmod(claudePath, 0000); err != nil {
		t.Fatal(err)
	}
	if _, err := piPreflightPlan(planned.Changes, planned.Resolver, f.home, f.root); err == nil {
		t.Fatal("unreadable Claude source admitted")
	}
	if err := os.Chmod(claudePath, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(f.home, "unknown-claude"))
	if _, err := piPreflightPlan(planned.Changes, planned.Resolver, f.home, f.root); err == nil || !strings.Contains(err.Error(), "uncertain") {
		t.Fatalf("unknown Claude override admitted: %v", err)
	}
	if string(mustRead(t, claudePath)) != "Separate Claude instructions\n" {
		t.Fatal("separate Claude file mutated")
	}
}

func TestPiPreflightCollisionAndPaths(t *testing.T) {
	f := dp02Setup(t)
	dp02Artifact(t, f.root, "sample-fixture", "skill", "---\nname: sample-fixture\ndescription: Fixture\n---\nBody\n")
	existing := filepath.Join(f.root, ".pi/skills/sample-fixture/SKILL.md")
	dp02File(t, existing, "---\nname: sample-fixture\ndescription: external\n---\nExternal\n")
	before := dp02Snapshot(t, f.home, f.root)
	_, _, err := runInstall(t, "sample-fixture", "--target", "pi", "--global", "--deploy", "--force")
	if err == nil || !strings.Contains(err.Error(), existing) {
		t.Fatalf("cross-scope collision absent: %v", err)
	}
	if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
		t.Fatal("collision refusal mutated files")
	}
}

func TestPiPreflightStaticExecutableSources(t *testing.T) {
	f := dp02Setup(t)
	t.Setenv("PI_OFFLINE", "")
	dp02Artifact(t, f.root, "fixture-role", "agent", "---\nname: fixture-role\ndescription: Fixture\n---\nBody\n")
	agent := filepath.Join(f.home, ".pi/agent")
	pkg := filepath.Join(f.home, "declared-package")
	explicit := filepath.Join(agent, "explicit.js")
	native := filepath.Join(agent, "extensions/native.ts")
	packageEntry := filepath.Join(pkg, "entry.js")
	for _, path := range []string{explicit, native, packageEntry} {
		dp02File(t, path, "throw new Error('fixture must never execute');\n")
	}
	dp02File(t, filepath.Join(pkg, "package.json"), `{"name":"invented-package","pi":{"extensions":["entry.js"],"skills":["skills"]}}`)
	dp02File(t, filepath.Join(pkg, "skills/other-skill/SKILL.md"), "---\nname: other-skill\ndescription: Fixture\n---\nBody\n")
	settings := filepath.Join(agent, "settings.json")
	dp02File(t, settings, fmt.Sprintf(`{"extensions":[%q],"packages":[%q]}`, explicit, pkg))
	marker := filepath.Join(f.home, "executed")
	bin := filepath.Join(f.home, "bin")
	for _, name := range []string{"npm", "pi", "git", "node"} {
		path := filepath.Join(bin, name)
		dp02File(t, path, fmt.Sprintf("#!/bin/sh\nprintf executed > %q\nexit 93\n", marker))
		if err := os.Chmod(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	out, warnings, err := runInstall(t, "fixture-role", "--target", "pi", "--global", "--deploy")
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"static inventory", "runtime-unverified", explicit, native, packageEntry, "global npm"} {
		if !strings.Contains(out+warnings, part) {
			t.Errorf("static boundary/source diagnostic missing %q: %s %s", part, out, warnings)
		}
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("runtime/resolver executed: %v", err)
	}
	for _, path := range []string{explicit, native, packageEntry} {
		if string(mustRead(t, path)) != "throw new Error('fixture must never execute');\n" {
			t.Fatalf("extension source changed: %s", path)
		}
	}
}

func TestPiPreflightExecutableSourcesKeepStaticGates(t *testing.T) {
	for _, name := range []string{"known collision", "missing extension", "malformed extensions", "unreadable extension", "symlink extension", "package escape", "malformed native manifest"} {
		t.Run(name, func(t *testing.T) {
			f := dp02Setup(t)
			t.Setenv("PI_OFFLINE", "")
			dp02Artifact(t, f.root, "valid-fixture", "skill", "---\nname: valid-fixture\ndescription: Fixture\n---\nBody\n")
			dp02Artifact(t, f.root, "fixture-role", "agent", "---\nname: fixture-role\ndescription: Fixture\n---\nBody\n")
			dp02File(t, filepath.Join(f.root, "profiles/selected-fixture.yaml"), "apiVersion: patronus/v2\nfamily: profile\nname: selected-fixture\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  capabilities: [valid-fixture, fixture-role]\n")
			agent := filepath.Join(f.home, ".pi/agent")
			pkg := filepath.Join(f.home, "fixture-package")
			entry := filepath.Join(agent, "entry.js")
			dp02File(t, entry, "throw new Error('never execute');")
			dp02File(t, filepath.Join(pkg, "entry.js"), "throw new Error('never execute');")
			dp02File(t, filepath.Join(pkg, "package.json"), `{"name":"fixture-package","pi":{"extensions":["entry.js"]}}`)
			settings := filepath.Join(agent, "settings.json")
			dp02File(t, settings, fmt.Sprintf(`{"extensions":["entry.js"],"packages":[%q]}`, pkg))
			switch name {
			case "known collision":
				dp02File(t, filepath.Join(f.root, ".pi/agents/fixture-role.md"), "---\nname: fixture-role\ndescription: Existing fixture\n---\nBody\n")
			case "missing extension":
				if err := os.Remove(entry); err != nil {
					t.Fatal(err)
				}
			case "malformed extensions":
				dp02File(t, settings, `{"extensions":null}`)
			case "symlink extension":
				if err := os.Remove(entry); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(pkg, "entry.js"), entry); err != nil {
					t.Fatal(err)
				}
			case "package escape":
				dp02File(t, filepath.Join(pkg, "package.json"), `{"name":"fixture-package","pi":{"extensions":["../entry.js"]}}`)
			case "malformed native manifest":
				dp02File(t, filepath.Join(agent, "extensions/package.json"), `{"pi":{"extensions":42}}`)
			}
			dp02File(t, filepath.Join(f.home, ".patronus/state.json"), `{"version":1,"items":[]}`)
			dp02File(t, filepath.Join(f.root, "patronus.lock"), `{"version":2,"entries":[]}`)
			before := dp02Snapshot(t, f.home, f.root)
			if name == "unreadable extension" {
				if err := os.Chmod(entry, 0); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := runInstall(t, "--profile", "selected-fixture", "--target", "pi", "--global", "--deploy", "--force", "--yes"); err == nil {
				t.Fatal("known static conflict admitted")
			}
			if name == "unreadable extension" {
				info, err := os.Stat(entry)
				if err != nil || info.Mode().Perm() != 0 {
					t.Fatalf("unreadable source mode changed: %v", err)
				}
				// Restore only the fixture's permission to compare every source byte.
				if err := os.Chmod(entry, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
				t.Fatal("static refusal changed destination/state/lock or sources")
			}
		})
	}
}

func TestPiPreflightRuntimeDiscoveryUnverified(t *testing.T) {
	f := dp02Setup(t)
	t.Setenv("PI_OFFLINE", "false")
	dp02Artifact(t, f.root, "sample-agent", "agent", "---\nname: sample-agent\ndescription: Fixture\n---\nBody\n")
	before := dp02Snapshot(t, f.home, f.root)
	out, warnings, err := runInstall(t, "sample-agent", "--target", "pi", "--global", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out+warnings, "global npm discovery is runtime-unverified") {
		t.Fatalf("missing runtime-only discovery boundary: %s %s", out, warnings)
	}
	if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
		t.Fatal("static preview caused writes")
	}
}

func TestPiPreflightDeployNative(t *testing.T) {
	f := dp02Setup(t)
	raw := "---\r\nname: sample-agent\r\ndescription: Fixture\r\ntools:\r\n---\r\nBody {unchanged}\r\n"
	dp02Artifact(t, f.root, "sample-agent", "agent", raw)
	out, _, err := runInstall(t, "sample-agent", "--target", "pi", "--global", "--deploy")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(mustRead(t, filepath.Join(f.home, ".pi/agent/agents/sample-agent.md"))); got != raw {
		t.Fatalf("deployed native bytes changed: %q", got)
	}
	if !strings.Contains(out, "runtime-unverified") || !strings.Contains(out, "startup/reload") {
		t.Fatalf("eligibility warning missing: %s", out)
	}
}

func TestPiPreflightNamedSkillClosure(t *testing.T) {
	f := dp02Setup(t)
	dp02Artifact(t, f.root, "fixture-role", "agent", "---\nname: fixture-role\ndescription: Fixture\nskills: fixture-skill\n---\nBody\n")
	dp02Artifact(t, f.root, "fixture-skill", "skill", "---\nname: fixture-skill\ndescription: Fixture\n---\nBody\n")
	before := dp02Snapshot(t, f.home, f.root)
	if _, _, err := runInstall(t, "fixture-role", "--target", "pi", "--global", "--deploy"); err == nil || !strings.Contains(err.Error(), "selected skill closure") {
		t.Fatalf("missing named skill admitted: %v", err)
	}
	if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
		t.Fatal("closure refusal wrote files")
	}
	if _, _, err := runInstall(t, "fixture-role", "fixture-skill", "--target", "pi", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
}

func TestPiPreflightMCPProposedCollision(t *testing.T) {
	f := dp02Setup(t)
	path := filepath.Join(f.home, ".pi/agent/mcp-adapter.json")
	prior := `{"mcpServers":{"fixture-name":{"url":"http://127.0.0.1/mcp"}}}`
	after := `{"mcpServers":{"fixture-name":{"url":"http://127.0.0.1/mcp"},"fixture_name":{"url":"http://127.0.0.1/mcp"}}}`
	dp02File(t, path, prior)
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Artifact: "fixture_name", Tool: "pi", Scope: "global", Path: path, Action: diff.Merge, Before: []byte(prior), After: []byte(after)}}}
	before := dp02Snapshot(t, f.home, f.root)
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("yes\n"))
	err := runDeployWith(cmd, cs, toolpath.New(os.LookupEnv, f.home, f.root), deployOptions{home: f.home, projectDir: f.root, yes: true, force: true}, &fakeRunner{})
	if err == nil || !strings.Contains(err.Error(), "duplicate normalized") {
		t.Fatalf("proposed duplicate admitted: %v", err)
	}
	if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
		t.Fatal("MCP preflight changed destination/state/lock")
	}
}

func TestPiPreflightMCPImportedCollision(t *testing.T) {
	for _, kind := range []string{"import", "package"} {
		t.Run(kind, func(t *testing.T) {
			f := dp02Setup(t)
			agent := filepath.Join(f.home, ".pi/agent")
			path := filepath.Join(agent, "mcp-adapter.json")
			prior := `{"imports":["cursor"],"mcpServers":{}}`
			server := "fixture_server"
			source := filepath.Join(f.home, ".cursor/mcp.json")
			dp02File(t, source, `{"mcpServers":{"fixture-server":{"command":"never-run-fixture"}}}`)
			if kind == "package" {
				prior = `{"mcpServers":{}}`
				server = "fixture_package__fixture_server"
				pkg := filepath.Join(f.home, "fixture-package")
				dp02File(t, filepath.Join(agent, "settings.json"), fmt.Sprintf(`{"packages":[%q]}`, pkg))
				dp02File(t, filepath.Join(pkg, "package.json"), `{"name":"fixture-package","pi":{"mcp":"defaults.json"}}`)
				source = filepath.Join(pkg, "defaults.json")
				dp02File(t, source, `{"mcpServers":{"fixture-server":{"command":"never-run-fixture"}}}`)
			}
			after := strings.Replace(prior, `"mcpServers":{}`, fmt.Sprintf(`"mcpServers":{%q:{"command":"never-run-fixture"}}`, server), 1)
			dp02File(t, path, prior)
			valid := filepath.Join(agent, "prompts/valid-fixture.md")
			cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Artifact: "valid-fixture", Type: "command", Tool: "pi", Scope: "global", Path: valid, Action: diff.Create, After: []byte("Fixture prompt\n")}, {Artifact: server, Tool: "pi", Scope: "global", Path: path, Action: diff.Merge, Before: []byte(prior), After: []byte(after)}}}
			before := dp02Snapshot(t, f.home, f.root)
			cmd := &cobra.Command{}
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetIn(strings.NewReader("yes\n"))
			runner := &fakeRunner{}
			err := runDeployWith(cmd, cs, toolpath.New(os.LookupEnv, f.home, f.root), deployOptions{home: f.home, projectDir: f.root, yes: true, force: true}, runner)
			if err == nil || !strings.Contains(err.Error(), "duplicate normalized") || !strings.Contains(err.Error(), source) {
				t.Fatalf("known imported collision admitted: %v", err)
			}
			if len(runner.ran) != 0 || !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
				t.Fatal("import conflict executed a runner or mutated destination/state/lock")
			}
		})
	}
}

func TestPiPreflightUnknownSectionOwner(t *testing.T) {
	f := dp02Setup(t)
	dp02Artifact(t, f.root, "fixture-context", "instruction", "New instruction\n")
	path := filepath.Join(f.home, ".pi/agent/AGENTS.md")
	dp02File(t, path, "<!-- patronus:start pi:fixture-context -->\nUnknown prior\n<!-- patronus:end pi:fixture-context -->\n")
	before := dp02Snapshot(t, f.home, f.root)
	if _, _, err := runInstall(t, "fixture-context", "--target", "pi", "--global", "--deploy", "--force"); err == nil || !strings.Contains(err.Error(), "unknown ownership") {
		t.Fatalf("unowned section replaced: %v", err)
	}
	if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
		t.Fatal("unowned section refusal wrote files")
	}
}

func TestPiPathsRelocatedInstall(t *testing.T) {
	for _, value := range []string{"~/kit", "relative-kit"} {
		t.Run(value, func(t *testing.T) {
			f := dp02Setup(t)
			t.Setenv("PI_CODING_AGENT_DIR", value)
			dp02Artifact(t, f.root, "fixture-skill", "skill", "---\nname: fixture-skill\ndescription: Fixture\n---\nBody\n")
			if _, _, err := runInstall(t, "fixture-skill", "--target", "pi", "--global", "--deploy"); err != nil {
				t.Fatal(err)
			}
			root := filepath.Join(f.root, "relative-kit")
			if value == "~/kit" {
				root = filepath.Join(f.home, "kit")
			}
			destination := filepath.Join(root, "skills/fixture-skill/SKILL.md")
			if _, err := os.Stat(destination); err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(mustRead(t, filepath.Join(f.home, ".patronus/state.json")), []byte(destination)) {
				t.Fatal("absolute relocated path not recorded")
			}
		})
	}
	for _, kind := range []string{"leading", "trailing", "uri"} {
		t.Run(kind, func(t *testing.T) {
			f := dp02Setup(t)
			value := filepath.Join(f.home, "unqualified")
			switch kind {
			case "leading":
				value = " " + value
			case "trailing":
				value += " "
			case "uri":
				value = "file://" + value
			}
			t.Setenv("PI_CODING_AGENT_DIR", value)
			dp02Artifact(t, f.root, "fixture-skill", "skill", "---\nname: fixture-skill\ndescription: Fixture\n---\nBody\n")
			before := dp02Snapshot(t, f.home, f.root)
			if _, _, err := runInstall(t, "fixture-skill", "--target", "pi", "--global", "--deploy"); err == nil {
				t.Fatal("mismatched root spelling admitted")
			}
			if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
				t.Fatal("root mismatch refusal wrote files")
			}
		})
	}
}

func TestPiPreflightNoConsentWithoutPi(t *testing.T) {
	root := t.TempDir()
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Path: filepath.Join(root, "CLAUDE.md"), Tool: "claude"}}}
	res := toolpath.New(os.LookupEnv, root, root)
	if reviews, err := piPreflightPlan(cs, res, root, root); err != nil || len(reviews) != 0 {
		t.Fatalf("legacy Claude discovery changed: %v", err)
	}
}
