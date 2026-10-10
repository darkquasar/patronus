package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

const (
	fixtureProviderValue = "fixture-not-a-secret"
	fixtureOpenCodeJSON  = `{"provider":{"invented":{"options":{"apiKey":"` + fixtureProviderValue + `"}}},"user":{"keep":true}}` + "\n"
)

// credentialFixture writes an invented OpenCode setting artifact into the
// fixture catalog and a project opencode.json carrying a provider option
// value no rendered view may show.
func credentialFixture(t *testing.T, version, value string) (root, configPath string) {
	t.Helper()
	root = fixtureCatalog(t)
	codexWrite(t, filepath.Join(root, "artifacts/settings/fix-setting-oc/patronus.yaml"), []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: setting\nrole: guardrail\nname: fix-setting-oc\ndescription: \"OpenCode only: invented setting.\"\nversion: "+version+"\ntargets: [opencode]\ndefaults:\n  scope: project\nsetting:\n  path: invented.flag\n  value: "+value+"\n"))
	configPath = filepath.Join(root, "opencode.json")
	codexWrite(t, configPath, []byte(fixtureOpenCodeJSON))
	return root, configPath
}

// assertRedacted: the provider value never appears; where a view renders the
// file's bytes (placeholder true), the placeholder stands in its place.
func assertRedacted(t *testing.T, stage, out string, placeholder bool) {
	t.Helper()
	if strings.Contains(out, fixtureProviderValue) {
		t.Fatalf("%s shows the provider value:\n%s", stage, out)
	}
	if placeholder && !strings.Contains(out, "[redacted]") {
		t.Fatalf("%s shows no redaction placeholder:\n%s", stage, out)
	}
}

// TestRenderRedactsProviderOptions: every rendered view of opencode.json bytes
// (install --verbose, update preview, remove --verbose) shows the placeholder
// and never the provider value, while the applied bytes, the state checksum
// and the file on disk keep the real value.
func TestRenderRedactsProviderOptions(t *testing.T) {
	root, configPath := credentialFixture(t, "1.0.0", "true")
	t.Setenv("HOME", t.TempDir())
	t.Chdir(root)

	out, _, err := runInstall(t, "fix-setting-oc", "--target", "opencode", "--local", "--verbose")
	if err != nil {
		t.Fatalf("install preview: %v\n%s", err, out)
	}
	assertRedacted(t, "install --verbose", out, true)

	if out, errOut, err := runInstall(t, "fix-setting-oc", "--target", "opencode", "--local", "--deploy"); err != nil {
		t.Fatalf("install deploy: %v\n%s\n%s", err, out, errOut)
	}
	applied, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(applied, []byte(fixtureProviderValue)) || bytes.Contains(applied, []byte("[redacted]")) {
		t.Fatalf("applied bytes altered by redaction:\n%s", applied)
	}
	s, err := state.Load(filepath.Join(root, ".patronus/state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !harnessOwnsPath(s, "fix-setting-oc", configPath) {
		t.Fatalf("setting row missing: %+v", s.Items)
	}
	for _, it := range s.Items {
		for _, f := range it.Files {
			if f.Path == configPath && f.Checksum != state.Checksum(applied) {
				t.Fatalf("state checksum does not match the real bytes: %s", f.Checksum)
			}
		}
	}

	codexWrite(t, filepath.Join(root, "artifacts/settings/fix-setting-oc/patronus.yaml"), []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: setting\nrole: guardrail\nname: fix-setting-oc\ndescription: \"OpenCode only: invented setting.\"\nversion: 1.1.0\ntargets: [opencode]\ndefaults:\n  scope: project\nsetting:\n  path: invented.flag\n  value: false\n"))
	out, _, err = runUpdate(t, "fix-setting-oc", "--target", "opencode")
	if err != nil {
		t.Fatalf("update preview: %v\n%s", err, out)
	}
	assertRedacted(t, "update preview", out, false)

	out, _, err = execRemove(t, "fix-setting-oc", "--target", "opencode", "--local", "--verbose")
	if err != nil {
		t.Fatalf("remove preview: %v\n%s", err, out)
	}
	assertRedacted(t, "remove --verbose", out, true)
	if out, errOut, err := execRemove(t, "fix-setting-oc", "--target", "opencode", "--local", "--deploy"); err != nil {
		t.Fatalf("remove deploy: %v\n%s\n%s", err, out, errOut)
	}
	restored, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(restored, []byte(fixtureProviderValue)) || bytes.Contains(restored, []byte("invented")) && bytes.Contains(restored, []byte(`"flag"`)) {
		t.Fatalf("removal left the wrong bytes:\n%s", restored)
	}
}

// TestConflictPromptRedactsProviderOptions: the interactive conflict prompt's
// [d]iff view renders the redacted copy of a diff carrying Redact.
func TestConflictPromptRedactsProviderOptions(t *testing.T) {
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader("d\ns\n"))
	home := t.TempDir()
	res := toolpath.New(os.LookupEnv, home, t.TempDir())
	d := diff.FileDiff{
		Path:   filepath.Join(home, "opencode.json"),
		Action: diff.Conflict,
		Before: []byte(fixtureOpenCodeJSON),
		After:  []byte(`{"provider":{"invented":{"options":{"apiKey":"second-fixture-value"}}},"user":{"keep":false}}` + "\n"),
		Redact: []string{"provider.*.options"},
	}
	how, err := conflictPrompt(cmd, res, false)(d)
	if err != nil || how != install.Skip {
		t.Fatalf("prompt = %v, %v", how, err)
	}
	assertRedacted(t, "conflict prompt diff", out.String(), false)
	if strings.Contains(out.String(), "second-fixture-value") {
		t.Fatalf("conflict prompt shows the incoming provider value:\n%s", out.String())
	}
}

// TestScanOutputNeverCarriesProviderValue: `patronus scan` over a home and
// project holding opencode.json never prints the provider value.
func TestScanOutputNeverCarriesProviderValue(t *testing.T) {
	root, _ := credentialFixture(t, "1.0.0", "true")
	home := t.TempDir()
	codexWrite(t, filepath.Join(home, ".config/opencode/opencode.json"), []byte(fixtureOpenCodeJSON))
	t.Setenv("HOME", home)
	t.Chdir(root)
	out, errOut, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v\n%s\n%s", err, out, errOut)
	}
	if strings.Contains(out+errOut, fixtureProviderValue) {
		t.Fatalf("scan output carries the provider value:\n%s\n%s", out, errOut)
	}
}
