package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

const openCodeRestartNotice = "Restart OpenCode: plugins, permissions and agents load at session start"

func openCodeLifecycleFixture(t *testing.T) (project, home string) {
	t.Helper()
	project, home = fixtureCatalog(t), t.TempDir()
	t.Setenv("HOME", home)
	for _, key := range []string{"OPENCODE_CONFIG_DIR", "XDG_CONFIG_HOME", "OPENCODE_CONFIG", "OPENCODE_CONFIG_CONTENT"} {
		t.Setenv(key, "")
	}
	t.Chdir(project)
	return project, home
}

func requireNotice(t *testing.T, out string, count int) {
	t.Helper()
	if n := strings.Count(out, openCodeRestartNotice); n != count {
		t.Fatalf("deploy notice count = %d, want %d:\n%s", n, count, out)
	}
}

func TestOpenCodeDeployNotice(t *testing.T) {
	openCodeLifecycleFixture(t)
	out, errOut, err := runInstall(t, "fix-skill", "--target", "opencode", "--local", "--deploy")
	if err != nil {
		t.Fatalf("install: %v\n%s\n%s", err, out, errOut)
	}
	requireNotice(t, out, 1)
}

func TestOpenCodeDeployNoticePreviews(t *testing.T) {
	for _, flags := range [][]string{{"--dry-run"}, {"--dry-run", "--json"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			project, _ := openCodeLifecycleFixture(t)
			args := append([]string{"fix-skill", "--target", "opencode", "--local"}, flags...)
			oldJSON := jsonOutput
			t.Cleanup(func() { jsonOutput = oldJSON })
			root := newRootCmd()
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs(append([]string{"install"}, args...))
			err := root.Execute()
			out, errOut := stdout.String(), stderr.String()
			if err != nil {
				t.Fatalf("preview: %v\n%s\n%s", err, out, errOut)
			}
			requireNotice(t, out+errOut, 0)
			mustExist(t, filepath.Join(project, ".opencode/skills/fix-skill/SKILL.md"), false)
		})
	}
}

// These entries are invented. The production adapter/descriptor are the target
// contract; no source catalog artifact or profile is loaded by this test.
func TestHarnessOpenCodeLifecycle(t *testing.T) {
	project, _ := openCodeLifecycleFixture(t)
	write := func(rel, body string) { codexWrite(t, filepath.Join(project, rel), []byte(body)) }
	for _, name := range []string{"lifecycle-rule", "lifecycle-sibling"} {
		write("artifacts/instructions/"+name+"/patronus.yaml", "apiVersion: patronus/v2\nfamily: artifact\ntype: instruction\nrole: instruction\nname: "+name+"\ndescription: invented\nversion: 1.0.0\nentry: INSTRUCTIONS.md\ntargets: [opencode]\ndefaults: {scope: project}\n")
		write("artifacts/instructions/"+name+"/INSTRUCTIONS.md", "invented "+name+" body\n")
	}
	write("artifacts/settings/lifecycle-permission/patronus.yaml", `apiVersion: patronus/v2
family: artifact
type: setting
role: guardrail
name: lifecycle-permission
description: invented
version: 1.0.0
targets: [opencode]
defaults: {scope: project}
setting:
  path: permission.edit
  value: {"*": allow, "*.env": deny, ".env.*": deny}
`)
	// Two script hooks contribute the same adapter runner. A later acquisition and
	// selective removal must agree with the registration list and owner rows.
	for _, name := range []string{"lifecycle-hook-a", "lifecycle-hook-b"} {
		write("artifacts/hooks/"+name+"/patronus.yaml", "apiVersion: patronus/v2\nfamily: artifact\ntype: hook\nrole: guardrail\nname: "+name+"\ndescription: invented\nversion: 1.0.0\nfiles: [guard.sh]\ntargets: [opencode]\ndefaults: {scope: project}\nhook:\n  event: PreToolUse\n  matcher: Write\n  command: '{script}'\n  script: guard.sh\n  intent: gate\n")
		write("artifacts/hooks/"+name+"/guard.sh", "invented inert guard bytes\n")
	}
	write("profiles/lifecycle-bundle.yaml", `apiVersion: patronus/v2
family: profile
role: lifecycle
name: lifecycle-bundle
version: 1.0.0
targets: [opencode]
layers:
  capabilities: [fix-skill]
  instructions: [lifecycle-rule, lifecycle-sibling]
  guardrails: [lifecycle-hook-a, lifecycle-permission]
  tools: [fix-mcp-two]
`)
	const prose = "# User prose\n\nKeep spacing exactly.\n"
	write("AGENTS.md", prose)
	const packageBytes = "{\"name\":\"owned-by-harness\", \"private\":true}\n"
	write(".opencode/package.json", packageBytes)
	write("opencode.json", `{"permission":{"edit":{"*.secret":"ask"},"bash":{"*":"ask"}},"mcp":{"fix-mcp-two":{"headers":{"x-invented":"fixture-not-a-secret"}}}}`)
	install := func(args ...string) string {
		t.Helper()
		out, errOut, err := runInstall(t, args...)
		if err != nil {
			t.Fatalf("install: %v\n%s\n%s", err, out, errOut)
		}
		requireNotice(t, out, 1)
		if got := string(mustRead(t, filepath.Join(project, ".opencode/package.json"))); got != packageBytes {
			t.Fatalf("package changed: %q", got)
		}
		return out
	}
	args := []string{"--profile", "lifecycle-bundle", "--target", "opencode", "--local", "--deploy"}
	install(args...)
	skill := filepath.Join(project, ".opencode/skills/fix-skill/SKILL.md")
	if got := string(mustRead(t, skill)); got != "---\nname: fix-skill\ndescription: fixture skill\n---\nDo the fixture thing.\n" {
		t.Fatalf("skill bytes: %q", got)
	}
	contextPath := filepath.Join(project, "AGENTS.md")
	beforeContext := mustRead(t, contextPath)
	if !bytes.Equal(beforeContext, adapter.AppendSection(adapter.AppendSection([]byte(prose), "oc:lifecycle-rule", []byte("invented lifecycle-rule body\n")), "oc:lifecycle-sibling", []byte("invented lifecycle-sibling body\n"))) {
		t.Fatalf("context bytes:\n%s", beforeContext)
	}
	runner := filepath.Join(project, ".opencode/plugins/patronus-hook-runner.ts")
	runnerBytes := mustRead(t, runner)
	if len(runnerBytes) == 0 {
		t.Fatal("empty runner")
	}
	s := lifecycleState(t, project)
	if len(s.Items) != 6 {
		t.Fatalf("installed items: %+v", s.Items)
	}
	permissionLeaves := map[string]bool{}
	for _, it := range s.Items {
		if it.Tool != "opencode" || it.Scope != "local" {
			t.Fatalf("wrong ownership: %+v", it)
		}
		for _, f := range it.Files {
			if f.Path == filepath.Join(project, ".opencode/package.json") {
				t.Fatal("harness package claimed")
			}
			if it.Artifact == "lifecycle-permission" && f.Setting != nil {
				permissionLeaves[f.Setting.Dotted] = true
			}
		}
	}
	if !reflect.DeepEqual(permissionLeaves, map[string]bool{"permission.edit.\"*\"": true, "permission.edit.\"*.env\"": true, "permission.edit.\".env.*\"": true}) {
		t.Fatalf("split leaves: %v", permissionLeaves)
	}
	config := lifecycleJSON(t, filepath.Join(project, "opencode.json"))
	wantConfig := map[string]any{
		"permission": map[string]any{"edit": map[string]any{"*": "allow", "*.env": "deny", ".env.*": "deny", "*.secret": "ask"}, "bash": map[string]any{"*": "ask"}},
		"mcp":        map[string]any{"fix-mcp-two": map[string]any{"type": "remote", "url": "https://fixture.invalid/mcp/", "headers": map[string]any{"x-invented": "fixture-not-a-secret"}}},
	}
	if !reflect.DeepEqual(config, wantConfig) {
		t.Fatalf("config: %#v", config)
	}
	out := install(args...)
	if !strings.Contains(out, "Applied: 0 written") {
		t.Fatalf("reinstall not all SKIP:\n%s", out)
	}
	// Lock independently exercises the ordinary shared target policy and exact closure.
	if out, errOut, err := runLock(t, "--profile", "lifecycle-bundle", "--target", "opencode", "--local-registry"); err != nil {
		t.Fatalf("lock: %v\n%s\n%s", err, out, errOut)
	}
	l, err := lock.Load(filepath.Join(project, "patronus.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if l.Version != lock.TargetVersion || l.Target != "opencode" || len(l.Entries) != 6 {
		t.Fatalf("lock: %+v", l)
	}
	names := map[string]bool{}
	for _, e := range l.Entries {
		names[e.Name] = true
	}
	if !reflect.DeepEqual(names, map[string]bool{"fix-skill": true, "lifecycle-rule": true, "lifecycle-sibling": true, "lifecycle-hook-a": true, "lifecycle-permission": true, "fix-mcp-two": true}) {
		t.Fatalf("lock closure: %v", names)
	}
	// Bump only the skill. Update replaces its bytes and leaves every other row/file intact.
	manifestPath := filepath.Join(project, "artifacts/skills/fix-skill/patronus.yaml")
	codexWrite(t, manifestPath, bytes.ReplaceAll(mustRead(t, manifestPath), []byte("version: 1.0.0"), []byte("version: 1.1.0")))
	const bumped = "---\nname: fix-skill\ndescription: fixture skill\n---\nBumped invented body.\n"
	write("artifacts/skills/fix-skill/SKILL.md", bumped)
	out, errOut, err := runUpdate(t, "fix-skill", "--target", "opencode", "--deploy")
	if err != nil {
		t.Fatalf("update: %v\n%s\n%s", err, out, errOut)
	}
	requireNotice(t, out, 1)
	if string(mustRead(t, skill)) != bumped || !bytes.Equal(mustRead(t, contextPath), beforeContext) || !bytes.Equal(mustRead(t, runner), runnerBytes) {
		t.Fatal("update changed unrelated bytes or failed to replace skill")
	}
	if len(lifecycleState(t, project).Items) != 6 {
		t.Fatal("update retired unrelated rows")
	}
	for _, it := range lifecycleState(t, project).Items {
		if it.Artifact == "fix-skill" && it.ItemVersion != "1.1.0" {
			t.Fatalf("version not bumped: %+v", it)
		}
	}
	install("lifecycle-hook-b", "--target", "opencode", "--local", "--deploy")
	lifecycleHookOwners(t, project, runner, []string{"lifecycle-hook-a", "lifecycle-hook-b"})
	remove := func(name string) {
		t.Helper()
		out, errOut, err := execRemove(t, name, "--target", "opencode", "--local", "--deploy")
		if err != nil {
			t.Fatalf("remove %s: %v\n%s\n%s", name, err, out, errOut)
		}
		if string(mustRead(t, filepath.Join(project, ".opencode/package.json"))) != packageBytes {
			t.Fatal("remove changed harness package")
		}
	}
	remove("lifecycle-rule")
	expectedContext := adapter.AppendSection([]byte(prose), "oc:lifecycle-sibling", []byte("invented lifecycle-sibling body\n"))
	if !bytes.Equal(mustRead(t, contextPath), expectedContext) {
		t.Fatalf("selective section removal:\n%s", mustRead(t, contextPath))
	}
	remove("lifecycle-hook-a")
	if !bytes.Equal(mustRead(t, runner), runnerBytes) {
		t.Fatal("shared runner changed while sibling owns it")
	}
	lifecycleHookOwners(t, project, runner, []string{"lifecycle-hook-b"})
	remove("lifecycle-hook-b")
	mustExist(t, runner, false)
	lifecycleHookOwners(t, project, runner, nil)
	install("lifecycle-hook-a", "--target", "opencode", "--local", "--deploy")
	lifecycleHookOwners(t, project, runner, []string{"lifecycle-hook-a"})
	if !bytes.Equal(mustRead(t, runner), runnerBytes) {
		t.Fatal("runner reinstall changed bytes")
	}
	remove("lifecycle-hook-a")
	remove("lifecycle-permission")
	remove("fix-mcp-two")
	wantUser := map[string]any{"permission": map[string]any{"edit": map[string]any{"*.secret": "ask"}, "bash": map[string]any{"*": "ask"}}, "mcp": map[string]any{"fix-mcp-two": map[string]any{"headers": map[string]any{"x-invented": "fixture-not-a-secret"}}}}
	if got := lifecycleJSON(t, filepath.Join(project, "opencode.json")); !reflect.DeepEqual(got, wantUser) {
		t.Fatalf("remove changed user config: %#v", got)
	}
	remove("fix-skill")
	mustExist(t, skill, false)
	remove("lifecycle-sibling")
	if got := string(mustRead(t, contextPath)); got != prose {
		t.Fatalf("user prose changed: %q", got)
	}
	if len(lifecycleState(t, project).Items) != 0 {
		t.Fatal("ownership remains after removal")
	}
}

func lifecycleState(t *testing.T, project string) *state.State {
	t.Helper()
	s, err := state.LoadWithPolicy(filepath.Join(project, ".patronus/state.json"), testRegistry(t).OwnershipPolicy())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func lifecycleJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(mustRead(t, path), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func lifecycleHookOwners(t *testing.T, project, runner string, want []string) {
	t.Helper()
	var owners []string
	for _, it := range lifecycleState(t, project).Items {
		for _, f := range it.Files {
			if f.Path == runner {
				owners = append(owners, it.Artifact)
				if f.Action != "CREATE" || f.Checksum != state.Checksum(mustRead(t, runner)) {
					t.Fatalf("runner ownership: %+v", f)
				}
			}
		}
	}
	if !reflect.DeepEqual(owners, want) {
		t.Fatalf("runner owners = %v, want %v", owners, want)
	}
	v := lifecycleJSON(t, filepath.Join(project, ".opencode/patronus/hooks.json"))
	hooks, _ := v["hooks"].(map[string]any)
	rows, _ := hooks["PreToolUse"].([]any)
	var ids []string
	for _, row := range rows {
		m, ok := row.(map[string]any)
		if !ok {
			t.Fatalf("hook row: %#v", row)
		}
		id, ok := m["patronusId"].(string)
		if !ok {
			t.Fatalf("hook id: %#v", m)
		}
		parts := strings.Split(id, ":")
		if len(parts) != 3 || parts[0] != "patronus" || parts[2] == "" {
			t.Fatalf("invalid registration identity %q", id)
		}
		ids = append(ids, parts[1])
	}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("registrations = %v, want %v", ids, want)
	}
}

func TestHarnessOpenCodeLifecycleMixedContextRefusesForce(t *testing.T) {
	for _, flags := range [][]string{nil, {"--force", "--yes"}} {
		t.Run(fmt.Sprint(flags), func(t *testing.T) {
			project, _ := openCodeLifecycleFixture(t)
			path := filepath.Join(project, "AGENTS.md")
			prior := adapter.AppendSection([]byte("User prose\n"), "foreign-toy", []byte("foreign body\n"))
			codexWrite(t, path, prior)
			seed := &state.State{Version: state.Version, Items: []state.Item{{Artifact: "foreign-toy", Type: "instruction", Tool: "codex", Scope: "local", Files: []state.FileState{{Path: path, Action: "APPEND", Checksum: state.Checksum(prior), Section: "foreign-toy", Prior: []byte("User prose\n")}}}}}
			if err := state.Save(filepath.Join(project, ".patronus/state.json"), seed); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"fix-instruction-2", "--target", "opencode", "--local", "--deploy"}, flags...)
			out, errOut, err := runInstall(t, args...)
			if err == nil || !strings.Contains(err.Error(), "mixed-context") || !strings.Contains(err.Error(), path) {
				t.Fatalf("mixed refusal: %v\n%s\n%s", err, out, errOut)
			}
			requireNotice(t, out+errOut, 0)
			if !bytes.Equal(mustRead(t, path), prior) {
				t.Fatal("refused context was changed")
			}
		})
	}
}

func TestHarnessOpenCodeLifecycleLegacyMergeRefuses(t *testing.T) {
	project, _ := openCodeLifecycleFixture(t)
	path := filepath.Join(project, "opencode.json")
	prior := []byte("{\"invented\":true}\n")
	codexWrite(t, path, prior)
	seed := fmt.Sprintf(`{"version":2,"items":[{"artifact":"legacy-toy","type":"setting","tool":"opencode","scope":"local","files":[{"path":%q,"action":"MERGE","checksum":%q,"setting":{"Target":{"File":"opencode.json","Format":"jsonc"},"Dotted":"invented","ScalarValue":true}}]}]}`, path, state.Checksum(prior))
	codexWrite(t, filepath.Join(project, ".patronus/state.json"), []byte(seed))
	out, errOut, err := runInstall(t, "fix-skill", "--target", "opencode", "--local", "--deploy")
	if err == nil || !strings.Contains(err.Error(), "state legacy-toy "+path+": missing or invalid pre-management prior evidence") {
		t.Fatalf("legacy refusal: %v\n%s\n%s", err, out, errOut)
	}
	requireNotice(t, out+errOut, 0)
	if !bytes.Equal(mustRead(t, path), prior) {
		t.Fatal("legacy refusal changed config")
	}
}

// An existing download seam creates an obstructing directory after admission,
// immediately before the second physical write. This works even for privileged
// users and does not introduce production IO hooks or rely on permission bits.
type lifecycleObstructingFetcher struct {
	path    string
	payload []byte
}

func (f lifecycleObstructingFetcher) Fetch(_ context.Context, _ string) (io.ReadCloser, error) {
	if err := os.Mkdir(f.path, 0o755); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(f.payload)), nil
}

func TestHarnessOpenCodeLifecyclePartialApplyDeployNotice(t *testing.T) {
	project, home := openCodeLifecycleFixture(t)
	skill := filepath.Join(project, ".opencode/skills/fix-skill/SKILL.md")
	fetched := filepath.Join(home, ".patronus/bin/fix-bin")
	codexWrite(t, filepath.Join(home, ".patronus/bin/.keep"), []byte("invented\n"))
	payload := []byte("invented physical write\n")
	old := fetcherForDeploy
	fetcherForDeploy = lifecycleObstructingFetcher{path: fetched, payload: payload}
	t.Cleanup(func() { fetcherForDeploy = old })
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{
		{Path: skill, Action: diff.Create, Artifact: "fix-skill", Type: "skill", Tool: "opencode", Scope: "local", After: mustRead(t, filepath.Join(project, "artifacts/skills/fix-skill/SKILL.md"))},
		{Path: fetched, Action: diff.Fetch, Artifact: "fix-bin", Type: "binary", Tool: "opencode", Scope: "local", Fetch: &diff.FetchSpec{URL: "https://invented.invalid/payload", SHA256: state.Checksum(payload), Dest: fetched}},
	}}
	cmd := newInstallCmd(testRegistry(t))
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := runDeployWith(cmd, cs, toolpath.New(os.LookupEnv, home, project), deployOptions{Harness: testRegistry(t), home: home, projectDir: project}, &fakeRunner{})
	if err == nil || !strings.Contains(err.Error(), "re-preview current bytes") || !strings.Contains(err.Error(), fetched) {
		t.Fatalf("physical apply failure: %v\n%s", err, out.String())
	}
	requireNotice(t, out.String(), 0)
	if !bytes.Equal(mustRead(t, skill), cs.Diffs[0].After) {
		t.Fatal("first write not retained")
	}
	s := lifecycleState(t, project)
	if len(s.Items) != 1 || s.Items[0].Artifact != "fix-skill" || len(s.Items[0].Files) != 1 || s.Items[0].Files[0].Path != skill {
		t.Fatalf("partial state: %+v", s)
	}
	mustExist(t, filepath.Join(home, ".patronus/state.json"), false)
	// Repair only the invented obstruction, then preview current reality. The
	// first owned file is a SKIP, while the failed/unowned download remains FETCH.
	if err := os.Remove(fetched); err != nil {
		t.Fatal(err)
	}
	preview, errOut, err := runInstall(t, "fix-skill", "fix-bin", "--target", "opencode", "--local", "--dry-run")
	if err != nil || !strings.Contains(preview, "SKIP") || !strings.Contains(preview, "FETCH") {
		t.Fatalf("re-preview: %v\n%s\n%s", err, preview, errOut)
	}
	requireNotice(t, preview+errOut, 0)
}

func TestSharedDeployNoticeDataAndFailurePaths(t *testing.T) {
	for _, notice := range []string{"", "Invented target: reload now"} {
		t.Run("notice="+notice, func(t *testing.T) {
			home, project := t.TempDir(), t.TempDir()
			reg, err := harness.NewRegistry(harness.Descriptor{Name: "toy", DeployNotice: notice})
			if err != nil {
				t.Fatal(err)
			}
			paths := []string{filepath.Join(project, "one.md"), filepath.Join(project, "two.md")}
			cs := &diff.ChangeSet{}
			for _, p := range paths {
				cs.Diffs = append(cs.Diffs, diff.FileDiff{Path: p, Action: diff.Create, Artifact: filepath.Base(p), Type: "instruction", Tool: "toy", Scope: "local", After: []byte("invented\n")})
			}
			opts := deployOptions{Harness: reg, home: home, projectDir: project}
			deploy := func(opts deployOptions) (string, error) {
				cmd := &cobra.Command{}
				var out bytes.Buffer
				cmd.SetOut(&out)
				err := runDeployWith(cmd, cs, toolpath.New(os.LookupEnv, home, project), opts, &fakeRunner{})
				return out.String(), err
			}
			opts.saveState = func(string, *state.State) error { return errors.New("invented state failure") }
			out, err := deploy(opts)
			if err == nil || !strings.Contains(err.Error(), "invented state failure") {
				t.Fatalf("state failure: %v\n%s", err, out)
			}
			if notice != "" && strings.Contains(out, notice) {
				t.Fatalf("success notice on failed state:\n%s", out)
			}
			// Separate roots avoid adopting uncertain writes after a persistence error.
			opts.home, opts.projectDir = t.TempDir(), t.TempDir()
			opts.saveState = nil
			for i := range cs.Diffs {
				cs.Diffs[i].Path = filepath.Join(opts.projectDir, filepath.Base(paths[i]))
			}
			out, err = deploy(opts)
			if err != nil {
				t.Fatalf("shared deploy: %v\n%s", err, out)
			}
			if notice == "" {
				if out != "\nApplied: 2 written, 0 skipped\n" {
					t.Fatalf("zero notice altered output: %q", out)
				}
			} else if strings.Count(out, notice) != 1 {
				t.Fatalf("data notice not emitted once: %q", out)
			}
		})
	}
}

// OC-CD5 keeps lenient sections. This observed acceptance is not a claim that
// arbitrary unrecorded foreign context is refused: the broader spec wording
// remains a coordinator review/disposition gap.
func TestHarnessOpenCodeLifecycleUnrecordedUnprefixedSection(t *testing.T) {
	project, _ := openCodeLifecycleFixture(t)
	path := filepath.Join(project, "AGENTS.md")
	prior := adapter.AppendSection([]byte("User prose\n"), "unrecorded-toy", []byte("unrecorded body\n"))
	codexWrite(t, path, prior)
	out, errOut, err := runInstall(t, "fix-instruction-2", "--target", "opencode", "--local", "--deploy")
	if err != nil {
		t.Fatalf("lenient install: %v\n%s\n%s", err, out, errOut)
	}
	requireNotice(t, out, 1)
	if got, found := adapter.RemoveSection(mustRead(t, path), "oc:fix-instruction-2"); !found || !bytes.Equal(got, prior) {
		t.Fatalf("unrecorded section changed:\n%s", got)
	}
}

func TestOpenCodeDeployNoticeStateFailure(t *testing.T) {
	project, home := openCodeLifecycleFixture(t)
	path := filepath.Join(project, ".opencode/skills/fix-skill/SKILL.md")
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Path: path, Action: diff.Create, Artifact: "fix-skill", Type: "skill", Tool: "opencode", Scope: "local", After: mustRead(t, filepath.Join(project, "artifacts/skills/fix-skill/SKILL.md"))}}}
	cmd := newInstallCmd(testRegistry(t))
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := runDeployWith(cmd, cs, toolpath.New(os.LookupEnv, home, project), deployOptions{Harness: testRegistry(t), home: home, projectDir: project, saveState: func(string, *state.State) error { return errors.New("invented state failure") }}, &fakeRunner{})
	if err == nil || !strings.Contains(err.Error(), "invented state failure") {
		t.Fatalf("state failure: %v\n%s", err, out.String())
	}
	requireNotice(t, out.String(), 0)
}

type lifecycleFailingRunner struct{}

func (lifecycleFailingRunner) Run([]string) error { return errors.New("invented exec failure") }

func TestSharedDeployNoticeExecFailure(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	reg, err := harness.NewRegistry(harness.Descriptor{Name: "toy", DeployNotice: "Invented reload notice"})
	if err != nil {
		t.Fatal(err)
	}
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Path: "invented-tool configure", Action: diff.Exec, Artifact: "invented-wire", Type: "fetch+run", Tool: "toy", Scope: "local", Exec: &diff.ExecSpec{Command: []string{"invented-tool", "configure"}, Display: "invented-tool configure", SelfManaged: true}}}}
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	err = runDeployWith(cmd, cs, toolpath.New(os.LookupEnv, home, project), deployOptions{Harness: reg, home: home, projectDir: project}, lifecycleFailingRunner{})
	if err == nil || !strings.Contains(err.Error(), "invented exec failure") {
		t.Fatalf("exec failure: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "Invented reload notice") {
		t.Fatalf("success notice after failed exec: %s", out.String())
	}
}
