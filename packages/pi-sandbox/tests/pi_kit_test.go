package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"gopkg.in/yaml.v3"
)

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The upstream publishes a normative Markdown specification, not JSON Schema.
// This deliberately narrow validator accepts only the reviewed upstream kit's
// fields and values, with an immutable image and one pinned extension installation.
// SPEC-v2 sections 3.2, 5.1, 5.4, 5.6 and 5.8 distinguish bundled inputs from
// runtime output paths. New fields or file references require a new review.
func TestPiPackageStaticKit(t *testing.T) {
	var provenance struct {
		Commit       string `json:"commit"`
		Image        string `json:"image"`
		PiVersion    string `json:"piVersion"`
		License      string `json:"license"`
		SchemaSHA256 string `json:"schemaSHA256"`
		SpecSHA256   string `json:"specSHA256"`
	}
	if err := json.Unmarshal(readFile(t, "testdata/upstream.json"), &provenance); err != nil {
		t.Fatal(err)
	}
	if provenance.Commit != "869c83997680a252ed2b35671b3fd0d9adc2d487" || provenance.PiVersion != "0.87.1" || provenance.License != "Apache-2.0" {
		t.Fatalf("unexpected release provenance: %+v", provenance)
	}
	for _, tt := range []struct{ path, digest string }{
		{"testdata/SPEC-v2.md", provenance.SchemaSHA256},
		{"testdata/upstream-spec.yaml", provenance.SpecSHA256},
	} {
		if got := fmt.Sprintf("sha256:%x", sha256.Sum256(readFile(t, tt.path))); got != tt.digest {
			t.Errorf("%s digest = %s, want %s", tt.path, got, tt.digest)
		}
	}
	var upstream, kit map[string]any
	if err := yaml.Unmarshal(readFile(t, "testdata/upstream-spec.yaml"), &upstream); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(readFile(t, "../spec.yaml"), &kit); err != nil {
		t.Fatal(err)
	}
	sandbox, ok := upstream["sandbox"].(map[string]any)
	if !ok {
		t.Fatal("upstream sandbox block missing")
	}
	if !regexp.MustCompile(`^docker\.io/sbx/pi-image@sha256:[a-f0-9]{64}$`).MatchString(provenance.Image) {
		t.Fatal("image is not immutable")
	}
	sandbox["image"] = provenance.Image
	setup := upstream["setup"].(map[string]any)
	setup["install"] = append(setup["install"].([]any), map[string]any{
		"command":     "pi install npm:pi-subagents@0.71.0",
		"user":        "1000",
		"description": "Install and register pinned pi-subagents inside the sandbox using Pi's package manager",
	})
	if !reflect.DeepEqual(kit, upstream) {
		t.Fatal("kit differs from reviewed upstream semantics beyond image pin and pinned extension installation")
	}
}

func TestPiPackageInventory(t *testing.T) {
	var descriptor struct {
		SchemaVersion int    `yaml:"schemaVersion"`
		Name          string `yaml:"name"`
		Version       string `yaml:"version"`
		Platforms     []struct {
			OS   string `yaml:"os"`
			Arch string `yaml:"arch"`
		} `yaml:"platforms"`
		Payload []struct {
			Path       string `yaml:"path"`
			Executable bool   `yaml:"executable"`
		} `yaml:"payload"`
	}
	d := yaml.NewDecoder(bytes.NewReader(readFile(t, "../package.yaml")))
	d.KnownFields(true)
	if err := d.Decode(&descriptor); err != nil {
		t.Fatal(err)
	}
	if descriptor.SchemaVersion != 1 || descriptor.Name != "pi-sandbox" || descriptor.Version != "1.1.0" {
		t.Fatal("unexpected package identity")
	}
	if len(descriptor.Platforms) != 1 || descriptor.Platforms[0].OS != "darwin" || descriptor.Platforms[0].Arch != "arm64" {
		t.Fatal("unexpected platform")
	}
	// The reviewed kit has no source-file references, extends, build context,
	// mixins or files/ tree. Exact semantic comparison above enforces that fact.
	// These are the complete allowed files: no credentials, account identifiers,
	// tokens, sessions, generated runtime files or test fixtures are payloads.
	want := map[string]bool{"spec.yaml": true, "README.md": true, "LICENSE": true, "NOTICE": true}
	for _, entry := range descriptor.Payload {
		if !want[entry.Path] || entry.Executable {
			t.Fatalf("unexpected payload: %+v", entry)
		}
		delete(want, entry.Path)
		info, err := os.Lstat(filepath.Join("..", entry.Path))
		if err != nil {
			t.Fatal(err)
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("nonregular payload %s", entry.Path)
		}
		if len(readFile(t, filepath.Join("..", entry.Path))) == 0 {
			t.Fatalf("empty payload %s", entry.Path)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing payloads: %v", want)
	}
}
