package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/state"
)

// These are application-capability fixtures, not packaged-content acceptance.
// All acquired bytes are invented and served in memory; no extension is loaded.
func dp07Profile(t *testing.T, f directoryFixture, version string, sidecar bool) []byte {
	t.Helper()
	dp02Artifact(t, f.root, "folio", "skill", "---\nname: folio\ndescription: Invented notes\n---\nRead carefully.\n")
	manifestPath := filepath.Join(f.root, "artifacts/folio/patronus.yaml")
	body := strings.ReplaceAll(string(mustRead(t, manifestPath)), "version: 1.0.0", "version: "+version)
	if sidecar {
		body += "files: [notes.txt]\n"
		dp06Write(t, filepath.Join(f.root, "artifacts/folio/notes.txt"), "Invented sidecar\n")
	}
	dp06Write(t, manifestPath, body)
	agent := []byte("---\r\nname: scribe\r\ndescription: Invented native role\r\ntools: read, bash\r\nskills: folio\r\ninheritProjectContext: true\r\ninheritGlobalContext: true\r\nallowNestedSubagents: false\r\n---\r\nReview " + version + ".\r\n")
	dp02Artifact(t, f.root, "scribe", "agent", string(agent))
	path := filepath.Join(f.root, "artifacts/scribe/patronus.yaml")
	dp06Write(t, path, strings.ReplaceAll(string(mustRead(t, path)), "version: 1.0.0", "version: "+version)+"requires: [folio]\n")
	dp02Artifact(t, f.root, "salute", "command", "Invented prompt\n")
	dp02Artifact(t, f.root, "conduct", "instruction", "Invented shared instruction\n")
	dp06Setting(t, f.root, "palette", version, "[]")
	dp06Write(t, filepath.Join(f.root, "recipes/relay.yaml"), fmt.Sprintf("apiVersion: patronus/v2\nfamily: recipe\nname: relay\nversion: %s\nrole: tools\ndescription: Invented HTTP wiring\nwire:\n  method: merge\n  actor: patronus\n  tools: [pi]\n  mcp:\n    transport: http\n    url: https://relay.invalid/v%s\n", version, version))
	dp06Write(t, filepath.Join(f.root, "profiles/folio-kit.yaml"), "apiVersion: patronus/v2\nfamily: profile\nname: folio-kit\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  capabilities: [folio, scribe, salute, palette]\n  instructions: [conduct]\n  tools: [relay]\n")
	return agent
}

func dp07JSON(t *testing.T, path string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(mustRead(t, path), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func dp07NoEffects(t *testing.T, f directoryFixture, before map[string]string) {
	t.Helper()
	if !reflect.DeepEqual(before, dp02Snapshot(t, f.home, f.root)) {
		t.Fatal("refusal changed payload, state or lock")
	}
}

func TestPiDeliveryIntegrationProfileLifecycle(t *testing.T) {
	for _, location := range []string{"global", "local", "relocated"} {
		t.Run(location, func(t *testing.T) {
			f := dp02Setup(t) // closes inherited Pi discovery overrides before any CLI call
			scope, dest := "global", os.Getenv("PI_CODING_AGENT_DIR")
			switch location {
			case "local":
				scope, dest = "local", filepath.Join(f.root, ".pi")
			case "relocated":
				dest = filepath.Join(f.home, "relocated", "agent")
				t.Setenv("PI_CODING_AGENT_DIR", dest)
			}
			agent := dp07Profile(t, f, "1.0.0", true)
			dp06Write(t, filepath.Join(dest, "settings.json"), `{"userColor":"amber"}`)
			dp06Write(t, filepath.Join(dest, "mcp-adapter.json"), `{"mcpServers":{"external":{"url":"https://external.invalid"}}}`)
			before := dp02Snapshot(t, f.home, f.root)
			if _, _, err := runInstall(t, "--profile", "folio-kit", "--target", "pi", "--"+scope); err != nil {
				t.Fatal(err)
			}
			dp07NoEffects(t, f, before)
			if _, _, err := runLock(t, "--profile", "folio-kit", "--target", "pi"); err != nil {
				t.Fatal(err)
			}
			lockPath := filepath.Join(f.root, "patronus.lock")
			locked := mustRead(t, lockPath)
			loaded, err := lock.Load(lockPath)
			if err != nil || loaded.Target != "pi" || loaded.Version != 3 {
				t.Fatalf("target-bearing lock reload: %+v %v", loaded, err)
			}
			out, _, err := runInstall(t, "--profile", "folio-kit", "--"+scope, "--deploy")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "runtime-unverified") || !strings.Contains(out, "reload") {
				t.Fatalf("missing static placement boundary: %s", out)
			}
			agentPath := filepath.Join(dest, "agents/scribe.md")
			if !bytes.Equal(agent, mustRead(t, agentPath)) {
				t.Fatal("native agent identity/path/CRLF bytes changed")
			}
			if got := string(mustRead(t, filepath.Join(dest, "skills/folio/notes.txt"))); got != "Invented sidecar\n" {
				t.Fatal(got)
			}
			if _, _, err := execRemove(t, "salute", "--target", "pi", "--"+scope, "--deploy", "--force"); err != nil {
				t.Fatal(err)
			}
			agent = dp07Profile(t, f, "2.0.0", false)
			out, _, err = runUpdate(t, "folio-kit", "--target", "pi", "--"+scope, "--deploy")
			// D-11 permits conservative retention, not silent ownership loss:
			// retire the old skill explicitly before reinstalling its smaller set.
			if err == nil || !strings.Contains(err.Error(), "obsolete owned path retained") {
				t.Fatalf("missing obsolete-sidecar conflict: %v", err)
			}
			dp06OwnedVersion(t, f.home, f.root, scope, "folio", "pi", "1.0.0")
			if string(mustRead(t, filepath.Join(dest, "skills/folio/notes.txt"))) != "Invented sidecar\n" {
				t.Fatal("obsolete sidecar was silently lost")
			}
			if _, _, err := execRemove(t, "folio", "--target", "pi", "--"+scope, "--deploy", "--force"); err != nil {
				t.Fatal(err)
			}
			if _, _, err := runInstall(t, "folio", "--target", "pi", "--"+scope, "--deploy"); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "absent pinned member") {
				t.Fatal("removed desired member not reported")
			}
			if _, _, err := runUpdate(t, "folio-kit", "--target", "pi", "--"+scope, "--deploy"); err != nil {
				t.Fatalf("reconciled profile update: %v", err)
			}
			for _, absent := range []string{"skills/folio/notes.txt", "prompts/salute.md"} {
				if _, err := os.Stat(filepath.Join(dest, absent)); !os.IsNotExist(err) {
					t.Fatalf("obsolete/removed file survived or resurrected: %s: %v", absent, err)
				}
			}
			if !bytes.Equal(agent, mustRead(t, agentPath)) || !bytes.Equal(locked, mustRead(t, lockPath)) {
				t.Fatal("native update changed bytes or desired lock")
			}
			settings := dp07JSON(t, filepath.Join(dest, "settings.json"))
			if settings["palette"] != "version-2.0.0" || settings["userColor"] != "amber" {
				t.Fatalf("scalar update lost value/sibling: %v", settings)
			}
			servers, ok := dp07JSON(t, filepath.Join(dest, "mcp-adapter.json"))["mcpServers"].(map[string]any)
			if !ok || !reflect.DeepEqual(servers["relay"], map[string]any{"url": "https://relay.invalid/v2.0.0"}) || servers["external"] == nil {
				t.Fatalf("MCP update lost value/sibling: %v", servers)
			}
			if _, _, err := runInstall(t, "salute", "--target", "pi", "--"+scope, "--deploy"); err != nil {
				t.Fatal(err)
			}
			if got := string(mustRead(t, filepath.Join(dest, "prompts/salute.md"))); got != "Invented prompt\n" {
				t.Fatal("explicit reinstall failed")
			}
			// Removal is selected-root qualified; an alternate root cannot select old ownership.
			sentinel := filepath.Join(f.home, "new-root")
			dp06Write(t, filepath.Join(sentinel, "agents/scribe.md"), "unowned sentinel")
			t.Setenv("PI_CODING_AGENT_DIR", sentinel)
			if scope == "global" {
				if _, _, err := execRemove(t, "scribe", "--target", "pi", "--global", "--deploy", "--force"); err == nil {
					t.Fatal("alternate root selected old ownership")
				}
				t.Setenv("PI_CODING_AGENT_DIR", dest)
			}
			for _, name := range []string{"scribe", "folio", "salute", "conduct", "palette", "relay"} {
				if _, _, err := execRemove(t, name, "--target", "pi", "--"+scope, "--deploy", "--force"); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := os.Stat(agentPath); !os.IsNotExist(err) {
				t.Fatalf("owned native agent not removed: %v", err)
			}
			if string(mustRead(t, filepath.Join(sentinel, "agents/scribe.md"))) != "unowned sentinel" {
				t.Fatal("environment change redirected inverse")
			}
			if !reflect.DeepEqual(dp07JSON(t, filepath.Join(dest, "settings.json")), map[string]any{"userColor": "amber"}) {
				t.Fatal("scalar inverse did not preserve original absent baseline and sibling")
			}
			servers, ok = dp07JSON(t, filepath.Join(dest, "mcp-adapter.json"))["mcpServers"].(map[string]any)
			if !ok || len(servers) != 1 || servers["external"] == nil {
				t.Fatal("MCP inverse lost external sibling or retained managed leaf")
			}
			runner, ok := runnerForCommands.(*fakeRunner)
			if !ok || len(runner.ran) != 0 || f.fetcher.calls != 0 {
				t.Fatal("static profile executed or acquired a payload")
			}
		})
	}
}

func TestPiDeliveryIntegrationNativeRefusalAndDrift(t *testing.T) {
	for _, invalid := range []struct{ name, source, manifest string }{
		{"unsupported-field", "permission: read-only\n", ""},
		{"files", "", "files: [notes.md]\n"},
		{"overrides", "", "overrides:\n  claude:\n    model: invented\n"},
		{"targets", "", "targets: [pi, claude]\n"},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			f := dp02Setup(t)
			dp07Profile(t, f, "1.0.0", true)
			path := filepath.Join(f.root, "artifacts/scribe/entry.md")
			dp06Write(t, path, strings.Replace(string(mustRead(t, path)), "tools: read, bash\r\n", invalid.source, 1))
			path = filepath.Join(f.root, "artifacts/scribe/patronus.yaml")
			body := string(mustRead(t, path))
			if invalid.name == "targets" {
				body = strings.Replace(body, "targets: [pi]\n", "", 1)
			}
			dp06Write(t, path, body+invalid.manifest)
			dp06Write(t, filepath.Join(f.root, "artifacts/scribe/notes.md"), "not an agent sidecar")
			before := dp02Snapshot(t, f.home, f.root)
			if _, _, err := runInstall(t, "--profile", "folio-kit", "--target", "pi", "--global", "--deploy"); err == nil {
				t.Fatal("invalid agent admitted")
			}
			dp07NoEffects(t, f, before)
		})
	}
	t.Run("drift-retained", func(t *testing.T) {
		f := dp02Setup(t)
		dp07Profile(t, f, "1.0.0", true)
		if _, _, err := runInstall(t, "--profile", "folio-kit", "--target", "pi", "--global", "--deploy"); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(os.Getenv("PI_CODING_AGENT_DIR"), "agents/scribe.md")
		dp06Write(t, path, "Operator edited native role\n")
		if _, _, err := execRemove(t, "scribe", "--target", "pi", "--global", "--deploy"); err == nil || !strings.Contains(err.Error(), "conflict") {
			t.Fatalf("drift removal did not refuse: %v", err)
		}
		if string(mustRead(t, path)) != "Operator edited native role\n" {
			t.Fatal("native drift destroyed")
		}
		dp06OwnedVersion(t, f.home, f.root, "global", "scribe", "pi", "1.0.0")
	})
}

func TestPiDeliveryIntegrationMixedContextAndStateFault(t *testing.T) {
	for _, scenario := range []string{"consent-and-remove", "state-save-failure"} {
		t.Run(scenario, func(t *testing.T) {
			f := dp02Setup(t)
			dp07Profile(t, f, "1.0.0", true)
			prepared := "Operator-prepared combined instructions\nClaude-specific section\nPi-specific section\n"
			path := filepath.Join(f.root, "CLAUDE.md")
			dp06Write(t, path, prepared)
			before := dp02Snapshot(t, f.home, f.root)
			if _, _, err := runInstall(t, "--profile", "folio-kit", "--target", "pi", "--local", "--deploy", "--yes"); err == nil || !strings.Contains(err.Error(), "mixed-context") {
				t.Fatalf("headless combined-context write admitted: %v", err)
			}
			dp07NoEffects(t, f, before)
			cmd := newInstallCmd()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			planned, err := planInstall(cmd, installPlanRequest{Profile: "folio-kit", Tool: "pi", Scope: "local", Home: f.home, ProjectDir: f.root})
			if err != nil {
				t.Fatal(err)
			}
			reviews, err := piPreflightPlan(planned.Changes, planned.Resolver, f.home, f.root)
			if err != nil {
				t.Fatal(err)
			}
			consents, err := confirmPiContexts(reviews, true, false, false, strings.NewReader("prepared combined file\n"), &output)
			if err != nil || len(consents) != 1 {
				t.Fatalf("separate consent: %v", err)
			}
			opts := deployOptions{target: "pi", home: f.home, projectDir: f.root, piConsents: consents}
			if scenario == "state-save-failure" {
				opts.saveState = func(string, *state.State) error { return errors.New("invented persistence fault") }
			}
			// Exercise the production consent/check/apply/persistence seams with
			// invented interactive input, not a fake claim of a real terminal.
			m, err := beginMutation(f.home, f.root)
			if err != nil {
				t.Fatal(err)
			}
			opts.mutation = m
			if err := m.checkPlan(planned.Changes, consents); err != nil {
				m.release()
				t.Fatal(err)
			}
			app := install.Applier{BeforeWrite: func(d diff.FileDiff) error { return m.checkFile(d, consents) }}
			result, err := app.Apply(planned.Changes)
			if err == nil {
				err = recordStateLocked(planned.Changes, result, opts)
			}
			m.close(&err)
			if scenario == "state-save-failure" {
				if err == nil || !strings.Contains(err.Error(), "ownership uncertain") || !strings.Contains(err.Error(), path) {
					t.Fatalf("missing partial-effect diagnostic: %v", err)
				}
				if _, err := os.Stat(removeStatePath("local", f.home, f.root)); !os.IsNotExist(err) {
					t.Fatalf("fault advanced durable state: %v", err)
				}
				if !strings.Contains(string(mustRead(t, path)), "pi:conduct") {
					t.Fatal("fixture did not reach committed bytes before state fault")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := execRemove(t, "conduct", "--target", "pi", "--local", "--deploy"); err != nil {
				t.Fatal(err)
			}
			if string(mustRead(t, path)) != prepared {
				t.Fatal("Pi inverse undid the separate operator migration")
			}
		})
	}
}

func TestPiDeliveryIntegrationGlobalDependencies(t *testing.T) {
	f := dp06Dependencies(t)
	rec := f.recipe(t, "fixture-tree", "1.0.0", "Invented orchestration payload")
	rec.Role = manifest.RoleOrchestration
	f.saveRecipe(t, rec)
	before := dp02Snapshot(t, f.home, f.root)
	if _, _, err := runInstall(t, "--profile", "fixture-pi", "--target", "pi", "--local", "--deploy"); err == nil {
		t.Fatal("local install without global ownership admitted")
	}
	dp07NoEffects(t, f, before)
	dp06InstallProfile(t, "global")
	dp06OwnedVersion(t, f.home, f.root, "global", "fix-bin", recipe.TargetAgnostic, "1.0.0")
	dp06OwnedVersion(t, f.home, f.root, "global", "fixture-tree", recipe.TargetAgnostic, "1.0.0")
	dp06InstallProfile(t, "local")
	globalBefore := dp01SnapshotFiles(t, f.home)
	dp06Setting(t, f.root, "fixture-feature", "2.0.0", "[fix-bin, fixture-tree]")
	statePath := removeStatePath("global", f.home, f.root)
	owned := mustRead(t, statePath)
	if err := os.Remove(statePath); err != nil {
		t.Fatal(err)
	}
	before = dp02Snapshot(t, f.home, f.root)
	if _, _, err := runUpdate(t, "fixture-pi", "--target", "pi", "--local", "--deploy"); err == nil || !strings.Contains(err.Error(), "separate global") {
		t.Fatalf("local update accepted missing global ownership: %v", err)
	}
	dp07NoEffects(t, f, before)
	dp06Write(t, statePath, string(owned))
	if _, _, err := runUpdate(t, "fixture-pi", "--target", "pi", "--local", "--deploy"); err != nil {
		t.Fatal(err)
	}
	dp06SameSnapshot(t, f.home, globalBefore)
	if _, _, err := execRemove(t, "fixture-feature", "fixture-optional", "--target", "pi", "--local", "--deploy", "--force"); err != nil {
		t.Fatal(err)
	}
	dp06SameSnapshot(t, f.home, globalBefore)
	localBefore := dp01SnapshotFiles(t, f.root)
	out, _, err := runUpdate(t, "fixture-pi", "--target", "pi", "--global", "--deploy")
	if err != nil {
		t.Fatal(err)
	}
	dp06SameSnapshot(t, f.root, localBefore)
	if !strings.Contains(out, "consumers") {
		t.Fatalf("shared dependency consumers not disclosed: %s", out)
	}
	dp06OwnedVersion(t, f.home, f.root, "global", "fix-bin", recipe.TargetAgnostic, "1.0.0")
	dp06OwnedVersion(t, f.home, f.root, "global", "fixture-tree", recipe.TargetAgnostic, "1.0.0")
	runner, ok := runnerForCommands.(*fakeRunner)
	if !ok || len(runner.ran) != 0 || f.fetcher.calls == 0 {
		t.Fatal("expected acquisition through inert seam and no runtime execution")
	}
}

func TestPiDeliveryIntegrationDependencyAdmission(t *testing.T) {
	for _, kind := range []string{"corrupt-pin", "unsupported-role", "unsupported-platform", "acquisition-failure", "unresolved-required"} {
		t.Run(kind, func(t *testing.T) {
			f := dp06Dependencies(t)
			rec := f.recipe(t, "fixture-tree", "1.0.0", "Invented payload")
			rec.Role = manifest.RoleOrchestration
			switch kind {
			case "corrupt-pin":
				rec.Delivery.Assets[0].SHA256 = strings.Repeat("0", 64)
			case "unsupported-role":
				rec.Role = manifest.RoleMemory
			case "unsupported-platform":
				rec.Delivery.Assets[0].OS = "unsupported-os"
			case "acquisition-failure":
				f.fetcher.bodies = map[string][]byte{}
			case "unresolved-required":
				dp06Setting(t, f.root, "fixture-feature", "1.0.0", "[unresolved-fixture]")
			}
			f.saveRecipe(t, rec)
			before := dp02Snapshot(t, f.home, f.root)
			if _, _, err := runInstall(t, "--profile", "fixture-pi", "--target", "pi", "--global", "--deploy"); err == nil {
				t.Fatal("invalid required delivery admitted")
			}
			dp07NoEffects(t, f, before)
		})
	}
}

func TestPiDeliveryIntegrationSandboxReceiptRecovery(t *testing.T) {
	f := dp02Setup(t)
	f.recipe(t, "shelter", "1.0.0", "Invented sandbox payload")
	if _, _, err := runInstall(t, "shelter", "--target", "pi", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
	receipt, err := packagestate.Load(f.home, "shelter")
	if err != nil || receipt == nil || receipt.Root != filepath.Dir(f.readme("shelter")) {
		t.Fatalf("sandbox receipt missing/wrong root: %+v %v", receipt, err)
	}
	// A receipt, not matching payload bytes, permits recovery of the lost
	// discovery reference. This is the unchanged directory lifecycle.
	statePath := removeStatePath("global", f.home, f.root)
	if err := os.Remove(statePath); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runInstall(t, "shelter", "--target", "pi", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
	recovered, err := packagestate.Load(f.home, "shelter")
	if err != nil || !reflect.DeepEqual(receipt, recovered) {
		t.Fatalf("receipt recovery changed payload pins: %v", err)
	}
	st, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	rows := st.Find("shelter", recipe.TargetAgnostic, "global")
	if len(rows) != 1 || rows[0].PackageReceipt != "shelter" || len(rows[0].Files) != 0 {
		t.Fatalf("receipt reference not repaired: %+v", rows)
	}
	if string(mustRead(t, f.readme("shelter"))) != "Invented sandbox payload" {
		t.Fatal("receipt recovery changed payload")
	}
}
