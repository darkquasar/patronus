package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func maskFixture(t *testing.T) (planInputs, string) {
	t.Helper()
	home, project := t.TempDir(), t.TempDir()
	shared := filepath.Join(project, "shared")
	roots := func(harness.RootInputs) ([]scan.SkillRoot, error) {
		return []scan.SkillRoot{{Path: shared, Name: func(_ []byte, path string) (string, error) { return filepath.Base(filepath.Dir(path)), nil }}}, nil
	}
	remote := filepath.Join(project, "not-read")
	foreign := func(harness.RootInputs) ([]scan.SkillRoot, error) {
		return []scan.SkillRoot{{Path: remote, SafePath: func(string) error { t.Fatal("nonintersecting root discovered"); return nil }}}, nil
	}
	reg, err := harness.NewRegistry(
		harness.Descriptor{Name: "host", ItemSuffix: "hh", OwnsEffects: true, AdmitPlan: true, AdmitRemoval: true, ContextFile: "CONTEXT.md", SkillRoots: roots, ResourcePhase: harness.ResourceBeforeOwnership, SkillMask: harness.SkillMask{File: "toy.json", Format: "json", Dotted: "permission.skill", Value: "deny"}},
		harness.Descriptor{Name: "alpha", ItemSuffix: "aa", SkillRoots: roots, ResourcePhase: harness.ResourceBeforeOwnership},
		harness.Descriptor{Name: "beta", ItemSuffix: "bb", SkillRoots: roots, ResourcePhase: harness.ResourceBeforeOwnership},
		harness.Descriptor{Name: "empty", SkillRoots: func(harness.RootInputs) ([]scan.SkillRoot, error) {
			t.Fatal("empty-suffix descriptor should not be resolved for masking")
			return nil, nil
		}, ResourcePhase: harness.ResourceBeforeOwnership},
		harness.Descriptor{Name: "far", ItemSuffix: "cc", SkillRoots: foreign, ResourcePhase: harness.ResourceBeforeOwnership})
	if err != nil {
		t.Fatal(err)
	}
	ad, err := manifest.DecodeAdapter([]byte("apiVersion: patronus/v2\nfamily: adapter\ntool: host\nlayout:\n  setting:\n    project: {file: toy.json, format: json}\n    redact: [provider.*]\n"))
	if err != nil {
		t.Fatal(err)
	}
	art, err := manifest.DecodeArtifact([]byte("apiVersion: patronus/v2\nfamily: artifact\nname: invented-mask\nversion: 1.0.0\ndescription: fixture\ntype: setting\ntargets: [host]\ndefaults: {scope: project}\nsetting: {path: permission.skill, computed: skill-mask}\n"))
	if err != nil {
		t.Fatal(err)
	}
	return planInputs{cat: &registry.Catalog{Artifacts: []registry.ArtifactEntry{{Manifest: art}}}, adapters: map[string]*manifest.Adapter{"host": ad}, res: toolpath.New(func(k string) (string, bool) {
		if k == "HOME" {
			return home, true
		}
		return "", false
	}, home, project), names: []string{art.Name}, tool: "host", scope: "local", harness: reg}, shared
}

func maskSkill(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("invented skill\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeMaskCatalog(t *testing.T, in planInputs) string {
	t.Helper()
	project := in.res.ResolveMarker(".", "host", "local")
	for path, object := range map[string]any{"adapters/host.yaml": in.adapters["host"], "artifacts/settings/fixture/patronus.yaml": in.cat.Artifacts[0].Manifest} {
		data, err := yaml.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		full := filepath.Join(project, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return project
}

func TestPermissionDefaultsPreservesSiblingLeaves(t *testing.T) {
	in, _ := maskFixture(t)
	art := in.cat.Artifacts[0].Manifest
	art.Name = "invented-defaults"
	art.Setting = &manifest.SettingSpec{Path: "permission.edit", Value: map[string]any{"*": "allow", "*.env": "deny", ".env.*": "deny"}}
	in.adapters["host"].Layout.Setting.SplitLeaves = []string{"permission.edit"}
	project := writeMaskCatalog(t, in)
	t.Setenv("HOME", in.res.ExpandHome("~"))
	t.Chdir(project)
	path := filepath.Join(project, "toy.json")
	if err := os.WriteFile(path, []byte(`{"permission":{"bash":{"*":"ask"},"edit":{"*.secret":"ask"}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		out, err := runWith(t, newInstallCmd(in.harness), art.Name, "--target", "host", "--local", "--local-registry", "--deploy")
		if err != nil {
			t.Fatalf("install/reinstall: %v\n%s", err, out)
		}
	}
	out, err := runWith(t, newUpdateCmd(in.harness), art.Name, "--local-registry", "--deploy")
	if err != nil || !strings.Contains(out, "up to date") || strings.Contains(out, "computed setting") {
		t.Fatalf("ordinary unchanged setting refreshed: %v\n%s", err, out)
	}
	art.Version = "1.0.1"
	writeMaskCatalog(t, in)
	out, err = runWith(t, newUpdateCmd(in.harness), art.Name, "--local-registry", "--deploy")
	if err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	ft := manifest.FileTarget{File: "toy.json", Format: "json"}
	for _, tc := range []struct {
		dotted string
		value  any
	}{{`permission.edit."*"`, "allow"}, {`permission.edit."*.env"`, "deny"}, {`permission.edit.".env.*"`, "deny"}, {`permission.edit."*.secret"`, "ask"}, {`permission.bash."*"`, "ask"}} {
		value, present, err := adapter.ReadDotted(mustRead(t, path), ft, tc.dotted)
		if err != nil || !present || value != tc.value {
			t.Fatalf("wrong leaf %s: %s", tc.dotted, mustRead(t, path))
		}
	}
	out, err = runWith(t, newRemoveCmd([]string{"revert"}, in.harness), art.Name, "--target", "host", "--local", "--deploy")
	if err != nil {
		t.Fatalf("remove: %v\n%s", err, out)
	}
	want := []byte(`{"permission":{"bash":{"*":"ask"},"edit":{"*.secret":"ask"}}}`)
	var gotTree, wantTree any
	if err := json.Unmarshal(mustRead(t, path), &gotTree); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &wantTree); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotTree, wantTree) {
		t.Fatalf("siblings or removal changed: %s", mustRead(t, path))
	}
}

func TestComputedFailedDeployPrintsNoSuccessMessage(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires ordinary-user directory permission enforcement")
	}
	in, shared := maskFixture(t)
	project := writeMaskCatalog(t, in)
	t.Setenv("HOME", in.res.ExpandHome("~"))
	t.Chdir(project)
	maskSkill(t, shared, "x-aa")
	if err := os.MkdirAll(filepath.Join(project, ".patronus"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(project, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(project, 0755); err != nil {
			t.Error(err)
		}
	})
	out, err := runWith(t, newInstallCmd(in.harness), "invented-mask", "--target", "host", "--local", "--deploy", "--local-registry")
	if err == nil || !strings.Contains(err.Error(), "permission denied") || strings.Count(out, "Masked 1 co-visible alpha") != 1 {
		t.Fatalf("failed deployment claimed success or missed preview: %v\n%s", err, out)
	}
}

func TestComputedRemovalDeletesOnlyOwnedLeaves(t *testing.T) {
	in, shared := maskFixture(t)
	project := writeMaskCatalog(t, in)
	t.Setenv("HOME", in.res.ExpandHome("~"))
	t.Chdir(project)
	path := filepath.Join(project, "toy.json")
	if err := os.WriteFile(path, []byte(`{"permission":{"skill":{"*":"ask","*-cc":"deny"}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	maskSkill(t, shared, "x-aa")
	out, err := runWith(t, newInstallCmd(in.harness), "invented-mask", "--target", "host", "--local", "--deploy", "--local-registry")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	out, err = runWith(t, newRemoveCmd([]string{"revert"}, in.harness), "invented-mask", "--target", "host", "--local", "--deploy")
	if err != nil {
		t.Fatalf("remove: %v\n%s", err, out)
	}
	ft := manifest.FileTarget{File: "toy.json", Format: "json"}
	if _, present, err := adapter.ReadDotted(mustRead(t, path), ft, `permission.skill."*-aa"`); err != nil || present {
		t.Fatalf("owned mask retained: %s", mustRead(t, path))
	}
	for _, tc := range []struct{ dotted, value string }{{`permission.skill."*"`, "ask"}, {`permission.skill."*-cc"`, "deny"}} {
		value, present, err := adapter.ReadDotted(mustRead(t, path), ft, tc.dotted)
		if err != nil || !present || value != tc.value {
			t.Fatalf("sibling changed: %s", mustRead(t, path))
		}
	}
}

func TestComputedEqualUnmanagedLeafNotAdopted(t *testing.T) {
	in, shared := maskFixture(t)
	maskSkill(t, shared, "x-aa")
	path := in.res.ResolveMarker("toy.json", "host", "local")
	before := []byte(`{"permission":{"skill":{"*-aa":"deny","*":"ask"}}}`)
	if err := os.WriteFile(path, before, 0644); err != nil {
		t.Fatal(err)
	}
	cs, err := computePlan(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 1 || cs.Diffs[0].Action != diff.Skip {
		t.Fatalf("equal user leaf writable: %+v", cs.Diffs)
	}
	result, err := (&install.Applier{}).Apply(cs)
	if err != nil {
		t.Fatal(err)
	}
	got := state.Reconcile(state.ReconcileInput{Desired: cs.Diffs, Result: *result, Now: "first"})
	if len(got.Items) != 0 || len(got.Unresolved) != 0 {
		t.Fatalf("equal user key was adopted: %+v", got)
	}
}

func TestComputedRetirementPreservesEditedAndUnmanagedLeaves(t *testing.T) {
	in, shared := maskFixture(t)
	maskSkill(t, shared, "x-aa")
	maskSkill(t, shared, "x-bb")
	cs, err := computePlan(in)
	if err != nil {
		t.Fatal(err)
	}
	result, err := (&install.Applier{}).Apply(cs)
	if err != nil {
		t.Fatal(err)
	}
	owned := state.Reconcile(state.ReconcileInput{Desired: cs.Diffs, Result: *result, Now: "first"}).Items
	project := in.res.ResolveMarker(".", "host", "local")
	if err := state.Save(filepath.Join(project, ".patronus/state.json"), &state.State{Version: state.Version, Items: owned}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(project, "toy.json")
	if err := os.WriteFile(path, []byte(`{"permission":{"skill":{"*-aa":"deny","*-bb":"ask","*-cc":"deny","*":"ask"}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(shared, "x-bb")); err != nil {
		t.Fatal(err)
	}
	var warning string
	in.warnf = func(f string, a ...any) { warning = fmt.Sprintf(f, a...) }
	cs, err = computePlan(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warning, `computed leaf permission.skill."*-bb" was edited; not retired`) {
		t.Fatalf("missing edited warning: %q", warning)
	}
	result, err = (&install.Applier{}).Apply(cs)
	if err != nil {
		t.Fatal(err)
	}
	got := state.Reconcile(state.ReconcileInput{Old: owned, Desired: cs.Diffs, Result: *result, Now: "next"})
	if len(got.Unresolved) != 0 || len(got.Items) != 1 || len(got.Items[0].Files) != 2 {
		t.Fatalf("edited owner lost or blocks refresh: %+v", got)
	}
	for _, d := range cs.Diffs {
		if d.Setting != nil && d.Setting.Dotted == `permission.skill."*-bb"` && d.Action != diff.Skip {
			t.Fatalf("edited leaf unexpectedly writable: %+v", d)
		}
	}
	value, present, err := adapter.ReadDotted(mustRead(t, path), manifest.FileTarget{File: "toy.json", Format: "json"}, `permission.skill."*-bb"`)
	if err != nil || !present || value != "ask" {
		t.Fatalf("edited key changed: %s", mustRead(t, path))
	}
}

func TestComputedRefreshAndRetire(t *testing.T) {
	in, shared := maskFixture(t)
	project := in.res.ResolveMarker(".", "host", "local")
	t.Setenv("HOME", in.res.ExpandHome("~"))
	t.Chdir(project)
	for path, object := range map[string]any{"adapters/host.yaml": in.adapters["host"], "artifacts/settings/invented-mask/patronus.yaml": in.cat.Artifacts[0].Manifest} {
		data, err := yaml.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		full := filepath.Join(project, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(project, "toy.json")
	if err := os.WriteFile(config, []byte(`{"permission":{"skill":{"*":"ask","*-cc":"deny"}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	maskSkill(t, shared, "x-aa")
	out, err := runWith(t, newInstallCmd(in.harness), "invented-mask", "--target", "host", "--local", "--deploy", "--local-registry")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if strings.Count(out, "Masked 1 co-visible alpha") != 2 {
		t.Fatalf("want plan and successful-deploy messages: %s", out)
	}
	preview, err := runWith(t, newInstallCmd(in.harness), "invented-mask", "--target", "host", "--local", "--local-registry")
	if err != nil || strings.Count(preview, "Masked 1 co-visible alpha") != 1 {
		t.Fatalf("preview message: %v\n%s", err, preview)
	}
	assertLeaves := func(want int) {
		t.Helper()
		s, err := state.Load(filepath.Join(project, ".patronus/state.json"))
		if err != nil {
			t.Fatal(err)
		}
		rows := s.Find("invented-mask", "host", "local")
		if len(rows) != 1 || len(rows[0].Files) != want || rows[0].ItemVersion != "1.0.0" {
			t.Fatalf("wrong member/leaves: %+v", rows)
		}
	}
	assertLeaves(1)
	maskSkill(t, shared, "x-bb")
	update := func() {
		t.Helper()
		out, err := runWith(t, newUpdateCmd(in.harness), "invented-mask", "--deploy", "--local-registry")
		if err != nil {
			t.Fatalf("update: %v\n%s", err, out)
		}
		if !strings.Contains(out, "computed setting, recomputing at 1.0.0") {
			t.Fatalf("did not refresh unchanged version: %s", out)
		}
	}
	update()
	assertLeaves(2)
	if err := os.RemoveAll(filepath.Join(shared, "x-bb")); err != nil {
		t.Fatal(err)
	}
	update()
	assertLeaves(1)
	if err := os.RemoveAll(shared); err != nil {
		t.Fatal(err)
	}
	update()
	assertLeaves(0)
	out, err = runWith(t, newRemoveCmd([]string{"revert"}, in.harness), "invented-mask", "--target", "host", "--local", "--deploy")
	if err != nil {
		t.Fatalf("remove zero member: %v\n%s", err, out)
	}
	for _, dotted := range []string{`permission.skill."*"`, `permission.skill."*-cc"`} {
		_, present, err := adapter.ReadDotted(mustRead(t, config), manifest.FileTarget{File: "toy.json", Format: "json"}, dotted)
		if err != nil || !present {
			t.Fatalf("user sibling removed: %s", mustRead(t, config))
		}
	}
}

func TestSkillMaskComputedLeaves(t *testing.T) {
	in, shared := maskFixture(t)
	var messages []string
	in.message = func(s string) { messages = append(messages, s) }
	maskSkill(t, shared, "x-aa")
	cs, err := computePlan(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 1 {
		t.Fatalf("want one mask leaf, got %+v", cs.Diffs)
	}
	d := cs.Diffs[0]
	if d.Setting == nil || d.Setting.Dotted != `permission.skill."*-aa"` || d.Artifact != "invented-mask" || d.Version != "1.0.0" || len(d.Redact) != 1 {
		t.Fatalf("wrong mask evidence: %+v", d)
	}
	value, present, err := adapter.ReadDotted(d.After, manifest.FileTarget{File: "toy.json", Format: "json"}, `permission.skill."*-aa"`)
	if err != nil || !present || value != "deny" {
		t.Fatalf("wrong written mask: %s %v", d.After, err)
	}
	if len(messages) != 1 || !strings.Contains(messages[0], "Masked 1 co-visible alpha skills (*-aa)") || !strings.Contains(messages[0], "toy.json") {
		t.Fatalf("wrong messages: %v", messages)
	}
	if err := os.RemoveAll(shared); err != nil {
		t.Fatal(err)
	}
	messages = nil
	cs, err = computePlan(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Diffs) != 0 || len(messages) != 1 || !strings.Contains(messages[0], "Nothing masked") {
		t.Fatalf("empty roots: %+v %v", cs.Diffs, messages)
	}
}
