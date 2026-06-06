package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultRole(t *testing.T) {
	cases := map[Kind]Role{
		KindSkill:       RoleCapability,
		KindAgent:       RoleCapability,
		KindCommand:     RoleCapability,
		KindHook:        RoleGuardrail,
		KindInstruction: RoleInstruction,
	}
	for k, want := range cases {
		if got := DefaultRole(k); got != want {
			t.Errorf("DefaultRole(%s) = %s, want %s", k, got, want)
		}
	}
}

func TestLoadArtifactDefaultsRole(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "patronus.yaml")
	content := `apiVersion: patronus/v1
kind: Skill
name: demo
description: A demo skill.
version: 1.0.0
entry: SKILL.md
targets: [claude]
defaults:
  scope: project
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := LoadArtifact(path)
	if err != nil {
		t.Fatalf("LoadArtifact: %v", err)
	}
	if a.Role != RoleCapability {
		t.Errorf("role = %s, want capability (defaulted)", a.Role)
	}
}

func TestLoadArtifactRejectsBadKind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "patronus.yaml")
	content := `apiVersion: patronus/v1
kind: Recipe
name: x
description: y
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadArtifact(path); err == nil {
		t.Fatal("expected error for non-artifact kind, got nil")
	}
}

func TestStringListScalarOrSequence(t *testing.T) {
	type wrap struct {
		Items StringList `yaml:"items"`
	}
	var scalar wrap
	if err := yaml.Unmarshal([]byte("items: solo\n"), &scalar); err != nil {
		t.Fatal(err)
	}
	if len(scalar.Items) != 1 || scalar.Items[0] != "solo" {
		t.Errorf("scalar => %v, want [solo]", scalar.Items)
	}

	var seq wrap
	if err := yaml.Unmarshal([]byte("items: [a, b]\n"), &seq); err != nil {
		t.Fatal(err)
	}
	if len(seq.Items) != 2 || seq.Items[0] != "a" || seq.Items[1] != "b" {
		t.Errorf("seq => %v, want [a b]", seq.Items)
	}
}
