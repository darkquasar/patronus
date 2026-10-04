package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/nativepi"
	"github.com/darkquasar/patronus/internal/state"
)

type fixturePiRunner struct {
	t                    *testing.T
	root, project, agent string
	calls                []nativepi.Invocation
	fail                 bool
	noop                 bool
}

func (r *fixturePiRunner) Run(_ context.Context, in nativepi.Invocation) (string, error) {
	r.t.Helper()
	r.calls = append(r.calls, in)
	if in.Cwd != r.project {
		r.t.Errorf("cwd=%q want %q", in.Cwd, r.project)
	}
	if !contains(in.Env, "PI_CODING_AGENT_DIR="+r.agent) {
		r.t.Errorf("agent override missing: %v", in.Env)
	}
	if len(in.Argv) == 2 && in.Argv[1] == "--version" {
		return nativepi.RuntimeVersion, nil
	}
	if len(in.Argv) < 4 || in.Argv[0] != "pi" {
		r.t.Fatalf("invalid argv %v", in.Argv)
	}
	settings := filepath.Join(r.root, "settings.json")
	cfg := map[string]any{"operator": true, "npmCommand": []string{"operator-npm", "--some-option"}}
	if b, err := os.ReadFile(settings); err == nil {
		if err = json.Unmarshal(b, &cfg); err != nil {
			r.t.Fatal(err)
		}
	}
	if !r.noop {
		switch in.Argv[1] {
		case "install":
			name, version, err := nativepi.ParseSource(in.Argv[2])
			if err != nil {
				r.t.Fatal(err)
			}
			cfg["packages"] = []string{in.Argv[2]}
			dp06Write(r.t, filepath.Join(r.root, "npm/node_modules", name, "package.json"), fmt.Sprintf(`{"name":%q,"version":%q}`, name, version))
		case "remove":
			if strings.Contains(strings.TrimPrefix(in.Argv[2], "npm:@"), "@") {
				r.t.Errorf("remove version not stripped: %v", in.Argv)
			}
			cfg["packages"] = []string{}
			if err := os.RemoveAll(filepath.Join(r.root, "npm/node_modules", strings.TrimPrefix(in.Argv[2], "npm:"))); err != nil {
				r.t.Fatal(err)
			}
		default:
			r.t.Fatalf("unexpected operation %v", in.Argv)
		}
		b, _ := json.Marshal(cfg)
		dp06Write(r.t, settings, string(b))
	}
	if r.fail {
		return "", fmt.Errorf("invented manager failure after effects")
	}
	return "", nil
}

func nativeFixture(t *testing.T, location string) (workflowDeliveryFixture, *fixturePiRunner) {
	t.Helper()
	f := newWorkflowDeliveryFixture(t, location)
	dp06Write(t, filepath.Join(f.root, "recipes/fixture-native.yaml"), "apiVersion: patronus/v2\nfamily: recipe\nname: fixture-native\nversion: 1.0.0\nrole: tools\ndeliver:\n  via: package-manager\n  install:\n    - manager: pi\n      ref: npm:@invented/plugin@1.2.3\n")
	agent := piSelectedRoot("global", f.home, f.root)
	runner := &fixturePiRunner{t: t, root: f.agentDir, agent: agent, project: f.root}
	previous := nativeRunner
	nativeRunner = runner
	t.Cleanup(func() { nativeRunner = previous })
	return f, runner
}

func TestPiNativeConsentPreviewInstallRemove(t *testing.T) {
	for _, location := range []string{"global", "local", "custom"} {
		t.Run(location, func(t *testing.T) {
			f, r := nativeFixture(t, location)
			args := f.args("fixture-native")
			for previewIndex, extra := range [][]string{nil, {"--force"}, {"--deploy", "--force"}} {
				before := dp01SnapshotFiles(t, f.home)
				jsonOutput = previewIndex == 2
				out, _, err := runInstall(t, append(append([]string{}, args...), extra...)...)
				jsonOutput = false
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(out, "does not check those files") {
					t.Errorf("missing honest warning: %s", out)
				}
				if len(r.calls) != 0 {
					t.Fatal("preview invoked native runner")
				}
				dp06SameSnapshot(t, f.home, before)
			}
			if _, _, err := runInstall(t, append(args, "--deploy", "--yes")...); err == nil {
				t.Fatal("yes authorized install")
			}
			if len(r.calls) != 0 {
				t.Fatal("decline invoked runner")
			}
			apply := append(args, "--deploy", "--allow-package-installs")
			if location == "local" {
				if _, _, err := runInstall(t, apply...); err == nil {
					t.Fatal("missing project trust allowed")
				}
				if len(r.calls) != 0 {
					t.Fatal("untrusted local invoked runner")
				}
				apply = append(apply, "--allow-pi-project-config")
			}
			if _, _, err := runInstall(t, apply...); err != nil {
				t.Fatal(err)
			}
			if len(r.calls) != 2 {
				t.Fatalf("calls=%v", r.calls)
			}
			want := []string{"pi", "install", "npm:@invented/plugin@1.2.3", "--no-approve"}
			if location == "local" {
				want = []string{"pi", "install", "npm:@invented/plugin@1.2.3", "-l", "--approve"}
			}
			if strings.Join(r.calls[1].Argv, "|") != strings.Join(want, "|") {
				t.Errorf("argv=%v", r.calls[1].Argv)
			}
			installed, err := state.Load(removeStatePath(f.scope, f.home, f.root))
			if err != nil {
				t.Fatal(err)
			}
			if len(installed.Items) != 1 || installed.Items[0].Native == nil || installed.Items[0].Native.Pending != nil || installed.Items[0].Native.InstalledAt == "" {
				t.Fatalf("state=%+v", installed)
			}
			remove := f.args("fixture-native", "--deploy")
			if location == "local" {
				remove = append(remove, "--allow-pi-project-config")
			}
			if _, _, err := execRemove(t, remove...); err != nil {
				t.Fatal(err)
			}
			workflowDeliveryWantAbsent(t, filepath.Join(f.agentDir, "npm/node_modules/@invented/plugin"))
			cfg := string(mustRead(t, filepath.Join(f.agentDir, "settings.json")))
			if !strings.Contains(cfg, "operator-npm") {
				t.Fatal("npmCommand lost")
			}
		})
	}
}

func nativePendingRemovalFixture(t *testing.T, marker, missing bool) (workflowDeliveryFixture, *fixturePiRunner) {
	t.Helper()
	f, r := nativeFixture(t, "custom")
	if _, _, err := runInstall(t, f.args("fixture-native", "--deploy", "--allow-package-installs")...); err != nil {
		t.Fatal(err)
	}

	statePath := removeStatePath(f.scope, f.home, f.root)
	s, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Items) != 1 || s.Items[0].Native == nil {
		t.Fatalf("missing native state: %+v", s.Items)
	}
	n := s.Items[0].Native
	if marker {
		n.Pending = &nativepi.Pending{Operation: n.Operation, Before: n.Observed}
	} else {
		n.Provenance = "pending"
	}
	if err := state.Save(statePath, s); err != nil {
		t.Fatal(err)
	}
	if missing {
		dp06Write(t, filepath.Join(f.agentDir, "settings.json"), `{"operator":true,"npmCommand":["operator-npm","--some-option"],"packages":[]}`)
		if err := os.RemoveAll(filepath.Join(f.agentDir, "npm/node_modules/@invented/plugin")); err != nil {
			t.Fatal(err)
		}
	}
	r.calls = nil
	return f, r
}

func TestPiNativePendingRemovalPreviewBlocked(t *testing.T) {
	for _, tc := range []struct {
		name            string
		marker, missing bool
	}{
		{name: "marker present", marker: true},
		{name: "marker missing", marker: true, missing: true},
		{name: "provenance present"},
		{name: "provenance missing", missing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, r := nativePendingRemovalFixture(t, tc.marker, tc.missing)
			beforeHome := dp01SnapshotFiles(t, f.home)
			beforeProject := dp01SnapshotFiles(t, f.root)
			const decision = "blocked; unresolved pending intent requires inspection/reconciliation"
			for _, preview := range []struct {
				name string
				args []string
				json bool
			}{
				{name: "normal-human"},
				{name: "force-human", args: []string{"--force"}},
				{name: "force-json", args: []string{"--force"}, json: true},
			} {
				t.Run(preview.name, func(t *testing.T) {
					oldJSON := jsonOutput
					jsonOutput = preview.json
					t.Cleanup(func() { jsonOutput = oldJSON })
					out, _, err := execRemove(t, f.args("fixture-native", preview.args...)...)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(out, decision) {
						t.Fatalf("pending preview did not report blocked decision: %s", out)
					}
					if strings.Contains(out, "force-remove selected identity") || strings.Contains(out, "retain; use --force") || strings.Contains(out, "settle verified absence") {
						t.Fatalf("pending preview promised a removal path: %s", out)
					}
				})
			}
			if len(r.calls) != 0 {
				t.Fatalf("pending preview invoked native runner: %v", r.calls)
			}
			dp06SameSnapshot(t, f.home, beforeHome)
			dp06SameSnapshot(t, f.root, beforeProject)
		})
	}
}

func TestPiNativePendingRemovalApplyBlocked(t *testing.T) {
	for _, tc := range []struct {
		name            string
		marker, missing bool
	}{
		{name: "marker present", marker: true},
		{name: "marker missing", marker: true, missing: true},
		{name: "provenance present"},
		{name: "provenance missing", missing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, r := nativePendingRemovalFixture(t, tc.marker, tc.missing)
			beforeHome := dp01SnapshotFiles(t, f.home)
			beforeProject := dp01SnapshotFiles(t, f.root)
			if _, _, err := execRemove(t, f.args("fixture-native", "--deploy", "--force")...); err == nil || !strings.Contains(err.Error(), "pending intent alone grants no native deletion authority") {
				t.Fatalf("force apply did not fail closed: %v", err)
			}
			if len(r.calls) != 0 {
				t.Fatalf("blocked force apply invoked native runner: %v", r.calls)
			}
			dp06SameSnapshot(t, f.home, beforeHome)
			dp06SameSnapshot(t, f.root, beforeProject)
		})
	}
}

func TestPiNativeFailureAndTracking(t *testing.T) {
	for _, mode := range []string{"noop", "failure", "tracking"} {
		t.Run(mode, func(t *testing.T) {
			f, r := nativeFixture(t, "custom")
			args := f.args("fixture-native", "--deploy", "--allow-package-installs")
			switch mode {
			case "noop":
				r.noop = true
			case "failure":
				r.fail = true
			case "tracking":
				dp06Write(t, filepath.Join(f.agentDir, "settings.json"), `{"packages":["npm:@invented/plugin@1.2.3"]}`)
				dp06Write(t, filepath.Join(f.agentDir, "npm/node_modules/@invented/plugin/package.json"), `{"name":"@invented/plugin","version":"1.2.3"}`)
				if _, _, err := runInstall(t, args...); err == nil {
					t.Fatal("silently enrolled external")
				}
				args = f.args("fixture-native", "--deploy", "--track-existing")
			}
			_, _, err := runInstall(t, args...)
			if mode != "tracking" && err == nil {
				t.Fatal("uncertain native install reported success")
			}
			s, err := state.Load(removeStatePath(f.scope, f.home, f.root))
			if err != nil {
				t.Fatal(err)
			}
			if len(s.Items) != 1 || s.Items[0].Native == nil {
				t.Fatalf("missing evidence: %+v", s)
			}
			n := s.Items[0].Native
			if mode == "noop" && (n.Pending == nil || n.Provenance != "pending") {
				t.Fatalf("noop blessed: %+v", n)
			}
			if mode == "tracking" {
				if len(r.calls) != 0 || n.TrackedAt == "" || n.InstalledAt != "" {
					t.Fatalf("tracking fabricated install: %+v calls=%v", n, r.calls)
				}
				if _, _, err := execRemove(t, f.args("fixture-native", "--deploy")...); err == nil {
					t.Fatal("tracked normal removal allowed")
				}
				if len(r.calls) != 0 {
					t.Fatal("retained tracking invoked manager")
				}
				if _, _, err := execRemove(t, f.args("fixture-native", "--deploy", "--force")...); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPiNativeUpdateReplansSettingsAndDetectsDrift(t *testing.T) {
	f, r := nativeFixture(t, "custom")
	dp06Setting(t, f.root, "fixture-toggle", "1.0.0", "[fixture-native]")
	if _, _, err := runInstall(t, f.args("fixture-toggle", "--deploy", "--allow-package-installs")...); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(f.agentDir, "settings.json")
	if got := string(mustRead(t, settings)); !strings.Contains(got, "fixture-toggle") || !strings.Contains(got, "npm:@invented/plugin@1.2.3") {
		t.Fatalf("manager state resurrected/lost: %s", got)
	}
	recipePath := filepath.Join(f.root, "recipes/fixture-native.yaml")
	dp06Write(t, recipePath, strings.ReplaceAll(strings.ReplaceAll(string(mustRead(t, recipePath)), "version: 1.0.0", "version: 2.0.0"), "@1.2.3", "@2.0.0"))
	before := len(r.calls)
	if _, _, err := runUpdate(t, f.args("fixture-native", "--deploy")...); err == nil {
		t.Fatal("update bypassed package consent")
	}
	if len(r.calls) != before {
		t.Fatal("declined update invoked manager")
	}
	if _, _, err := runUpdate(t, f.args("fixture-native", "--deploy", "--allow-package-installs")...); err != nil {
		t.Fatal(err)
	}
	if got := r.calls[len(r.calls)-1].Argv; got[1] != "install" || got[2] != "npm:@invented/plugin@2.0.0" {
		t.Fatal(got)
	}
	dp06Write(t, filepath.Join(f.agentDir, "npm/node_modules/@invented/plugin/package.json"), `{"name":"@invented/plugin","version":"9.0.0"}`)
	before = len(r.calls)
	out, _, err := execRemove(t, f.args("fixture-native")...)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "declaration/version changed") || !strings.Contains(out, "installed ") {
		t.Fatalf("missing drift evidence/date: %s", out)
	}
	if _, _, err := execRemove(t, f.args("fixture-native", "--deploy")...); err == nil {
		t.Fatal("drift removal authorized")
	}
	if len(r.calls) != before {
		t.Fatal("drift retain invoked manager")
	}
	if _, _, err := execRemove(t, f.args("fixture-native", "--deploy", "--force")...); err != nil {
		t.Fatal(err)
	}
	if got := string(mustRead(t, settings)); !strings.Contains(got, "fixture-toggle") {
		t.Fatal("unrelated setting removed")
	}
}

func TestPiNativeStoredProfilesAndIndependentRoots(t *testing.T) {
	f, r := nativeFixture(t, "custom")
	for _, name := range []string{"native-team-a", "native-team-b"} {
		dp06Write(t, filepath.Join(f.root, "profiles", name+".yaml"), fmt.Sprintf("apiVersion: patronus/v2\nfamily: profile\nname: %s\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  capabilities: [fixture-native@pi, fixture-workflow@pi]\n", name))
		if _, _, err := runInstall(t, "--profile", name, "--target", "pi", "--global", "--deploy", "--allow-package-installs"); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(filepath.Join(f.root, "profiles")); err != nil {
		t.Fatal(err)
	}
	calls := len(r.calls)
	if _, _, err := execRemove(t, f.args("native-team-a", "--deploy")...); err == nil {
		t.Fatal("shared profile removed normally")
	}
	if len(r.calls) != calls {
		t.Fatal("shared retention invoked manager")
	}
	if _, _, err := execRemove(t, f.args("native-team-a", "--deploy", "--force")...); err != nil {
		t.Fatal(err)
	}
	s, err := state.Load(removeStatePath(f.scope, f.home, f.root))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Profiles) != 1 || s.Profiles[0].Name != "native-team-b" || len(s.Profiles[0].Members) != 2 {
		t.Fatalf("other profile references lost: %+v", s.Profiles)
	}
	workflowDeliveryWantAbsent(t, f.skillDir())
	// Fresh native installs in two roots remain independently controlled.
	if _, _, err := runInstall(t, f.args("fixture-native", "--deploy", "--allow-package-installs")...); err != nil {
		t.Fatal(err)
	}
	first := f.agentDir
	other := t.TempDir()
	t.Setenv("PI_CODING_AGENT_DIR", other)
	r.root, r.agent = other, other
	if _, _, err := runInstall(t, f.args("fixture-native", "--deploy", "--allow-package-installs")...); err != nil {
		t.Fatal(err)
	}
	if _, _, err := execRemove(t, f.args("fixture-native", "--deploy", "--force")...); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(first, "npm/node_modules/@invented/plugin/package.json")); err != nil {
		t.Fatal("other root lost", err)
	}
}

func TestPiNativeRemoveFailureAfterMutationRetainsEvidence(t *testing.T) {
	f, r := nativeFixture(t, "custom")
	if _, _, err := runInstall(t, f.args("fixture-native", "--deploy", "--allow-package-installs")...); err != nil {
		t.Fatal(err)
	}
	r.fail = true
	if _, _, err := execRemove(t, f.args("fixture-native", "--deploy")...); err == nil {
		t.Fatal("manager failure reported success")
	}
	s, err := state.Load(removeStatePath(f.scope, f.home, f.root))
	if err != nil {
		t.Fatal(err)
	}
	native := s.Items[0].Native
	if len(s.Items) != 1 || native.Observed.Status != "missing" || native.Pending == nil || native.Pending.Operation.Kind != "remove" || !strings.Contains(native.Outcome, "manager-error") {
		t.Fatalf("lost unresolved manager failure evidence: %+v", s)
	}
	r.fail = false
	calls := len(r.calls)
	for _, extra := range [][]string{nil, {"--force"}} {
		args := append(f.args("fixture-native", "--deploy"), extra...)
		if _, _, err := execRemove(t, args...); err == nil || !strings.Contains(err.Error(), "pending intent alone grants no native deletion authority") {
			t.Fatalf("unreconciled manager failure did not block retry %v: %v", extra, err)
		}
	}
	if len(r.calls) != calls {
		t.Fatal("blocked retry invoked native manager")
	}
}

func TestPiNativeExternalMalformedNoProbe(t *testing.T) {
	f, r := nativeFixture(t, "custom")
	dp06Write(t, filepath.Join(f.agentDir, "settings.json"), `{"packages":["npm:@invented/plugin@1.2.3"],"packages":[]}`)
	if _, _, err := runInstall(t, f.args("fixture-native", "--deploy", "--allow-package-installs", "--force")...); err == nil {
		t.Fatal("force bypassed malformed settings")
	}
	if len(r.calls) > 0 {
		t.Fatal("malformed observation invoked manager")
	}
}

func TestPiNativeRecipeVersionOnlyUpdateAdvancesState(t *testing.T) {
	f, r := nativeFixture(t, "custom")
	if _, _, err := runInstall(t, f.args("fixture-native", "--deploy", "--allow-package-installs")...); err != nil {
		t.Fatal(err)
	}
	calls := len(r.calls)
	recipePath := filepath.Join(f.root, "recipes/fixture-native.yaml")
	dp06Write(t, recipePath, strings.Replace(string(mustRead(t, recipePath)), "version: 1.0.0", "version: 2.0.0", 1))
	if _, _, err := runUpdate(t, f.args("fixture-native", "--deploy")...); err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != calls {
		t.Fatalf("recipe-only update invoked Pi: before=%d after=%d", calls, len(r.calls))
	}
	s, err := state.Load(removeStatePath(f.scope, f.home, f.root))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Items) != 1 || s.Items[0].ItemVersion != "2.0.0" || s.Items[0].Native == nil || s.Items[0].Native.Operation.Source != "npm:@invented/plugin@1.2.3" {
		t.Fatalf("recipe-only native update did not advance controlled state: %+v", s.Items)
	}
}

func TestPiNativeLocalRemoveRefreshesAgentDir(t *testing.T) {
	f, r := nativeFixture(t, "local")
	if _, _, err := runInstall(t, f.args("fixture-native", "--deploy", "--allow-package-installs", "--allow-pi-project-config")...); err != nil {
		t.Fatal(err)
	}
	currentAgentRoot := t.TempDir()
	t.Setenv("PI_CODING_AGENT_DIR", currentAgentRoot)
	r.agent = currentAgentRoot
	if _, _, err := execRemove(t, f.args("fixture-native", "--deploy", "--allow-pi-project-config")...); err != nil {
		t.Fatal(err)
	}
	if len(r.calls) < 4 {
		t.Fatalf("missing remove probe/operation: %v", r.calls)
	}
	for _, call := range r.calls[len(r.calls)-2:] {
		if !contains(call.Env, "PI_CODING_AGENT_DIR="+currentAgentRoot) {
			t.Fatalf("local removal reused stale agent root: %v", call.Env)
		}
		if call.Cwd != f.root {
			t.Fatalf("local removal cwd=%q want current project %q", call.Cwd, f.root)
		}
	}
}

type failingSecondProbeRunner struct {
	t                    *testing.T
	root, project, agent string
	probes, operations   int
}

func (r *failingSecondProbeRunner) Run(_ context.Context, in nativepi.Invocation) (string, error) {
	r.t.Helper()
	if in.Cwd != r.project || !contains(in.Env, "PI_CODING_AGENT_DIR="+r.agent) {
		r.t.Fatalf("unexpected native context: cwd=%q env=%v", in.Cwd, in.Env)
	}
	if len(in.Argv) == 2 && in.Argv[1] == "--version" {
		r.probes++
		if r.probes == 2 {
			return "", fmt.Errorf("invented second probe failure")
		}
		return nativepi.RuntimeVersion, nil
	}
	r.operations++
	name := strings.TrimPrefix(in.Argv[2], "npm:")
	if err := os.RemoveAll(filepath.Join(r.root, "npm/node_modules", filepath.FromSlash(name))); err != nil {
		r.t.Fatal(err)
	}
	var settings struct {
		Packages []string `json:"packages"`
	}
	settingsPath := filepath.Join(r.root, "settings.json")
	if err := json.Unmarshal(mustRead(r.t, settingsPath), &settings); err != nil {
		r.t.Fatal(err)
	}
	kept := settings.Packages[:0]
	for _, source := range settings.Packages {
		if source != in.Argv[2] && source != "npm:"+name+"@1.0.0" {
			kept = append(kept, source)
		}
	}
	settings.Packages = kept
	body, err := json.Marshal(settings)
	if err != nil {
		r.t.Fatal(err)
	}
	dp06Write(r.t, settingsPath, string(body))
	return "", nil
}

func TestPiNativeRemoveBatchProbeFailureMutatesNothing(t *testing.T) {
	home, project, root := t.TempDir(), t.TempDir(), t.TempDir()
	sources := []string{"npm:@invented/one@1.0.0", "npm:@invented/two@1.0.0"}
	dp06Write(t, filepath.Join(root, "settings.json"), `{"packages":["npm:@invented/one@1.0.0","npm:@invented/two@1.0.0"]}`)
	s := &state.State{Version: state.Version}
	cs := &diff.ChangeSet{}
	for i, source := range sources {
		name, version, err := nativepi.ParseSource(source)
		if err != nil {
			t.Fatal(err)
		}
		dp06Write(t, filepath.Join(root, "npm/node_modules", filepath.FromSlash(name), "package.json"), fmt.Sprintf(`{"name":%q,"version":%q}`, name, version))
		identity := nativepi.Identity{Name: name, Scope: "global", Root: root, Project: project, AgentRoot: root}
		installed := nativepi.Operation{Identity: identity, Kind: "install", Source: source}
		observation, err := nativepi.Observe(installed)
		if err != nil {
			t.Fatal(err)
		}
		s.Items = append(s.Items, state.Item{Artifact: fmt.Sprintf("native-%d", i), ItemVersion: "1.0.0", Type: "recipe", Tool: "pi", Scope: "global", Root: root, Native: &nativepi.Record{Operation: installed, Provenance: "installed", Observed: observation}})
		removeOp := installed
		removeOp.Kind = "remove"
		cs.Diffs = append(cs.Diffs, diff.FileDiff{Action: diff.Native, Artifact: fmt.Sprintf("native-%d", i), Version: "1.0.0", Type: "recipe", Tool: "pi", Scope: "global", Root: root, Path: identity.MetadataPath(), Native: &removeOp})
	}
	statePath := removeStatePath("global", home, project)
	if err := state.Save(statePath, s); err != nil {
		t.Fatal(err)
	}
	authoredPath := filepath.Join(project, "authored.txt")
	authored := []byte("authored bytes must survive\n")
	dp06Write(t, authoredPath, string(authored))
	cs.Diffs = append(cs.Diffs, diff.FileDiff{Action: diff.Delete, Artifact: "authored", Version: "1.0.0", Type: "instruction", Tool: "claude", Scope: "local", Path: authoredPath, Before: authored})

	beforeState := append([]byte(nil), mustRead(t, statePath)...)
	beforeSettings := append([]byte(nil), mustRead(t, filepath.Join(root, "settings.json"))...)
	runner := &failingSecondProbeRunner{t: t, root: root, project: project, agent: root}
	previous := nativeRunner
	nativeRunner = runner
	t.Cleanup(func() { nativeRunner = previous })
	cmd := newRemoveCmd(nil)
	if err := runRemove(cmd, cs, nil, nil, map[string]*state.State{"global": s}, removeStateOpts{home: home, projectDir: project}); err == nil || !strings.Contains(err.Error(), "second probe failure") {
		t.Fatalf("missing batch probe failure: %v", err)
	}
	if runner.operations != 0 {
		t.Fatalf("native operation ran before whole-batch probe admission: %d", runner.operations)
	}
	if got := mustRead(t, authoredPath); string(got) != string(authored) {
		t.Fatalf("authored bytes changed after native admission failure: %q", got)
	}
	if got := mustRead(t, statePath); string(got) != string(beforeState) {
		t.Fatalf("state changed after native admission failure:\n%s", got)
	}
	if got := mustRead(t, filepath.Join(root, "settings.json")); string(got) != string(beforeSettings) {
		t.Fatalf("native settings changed after batch probe failure: %s", got)
	}
	for _, source := range sources {
		name, _, _ := nativepi.ParseSource(source)
		if _, err := os.Stat(filepath.Join(root, "npm/node_modules", filepath.FromSlash(name), "package.json")); err != nil {
			t.Fatalf("native package %s changed after batch probe failure: %v", name, err)
		}
	}
}

func TestScanNativePackagesDeduplicatesIdenticalStatePath(t *testing.T) {
	f, _ := nativeFixture(t, "custom")
	if _, _, err := runInstall(t, f.args("fixture-native", "--deploy", "--allow-package-installs")...); err != nil {
		t.Fatal(err)
	}
	statuses, err := scanNativePackages(f.home, f.home)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 {
		t.Fatalf("identical home/project state path produced %d native statuses: %+v", len(statuses), statuses)
	}
}
