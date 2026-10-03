package nativepi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSourceExactOnly(t *testing.T) {
	for _, s := range []string{"npm:invented@1.2.3", "npm:@team/invented@0.0.1-beta.2+build"} {
		if _, _, err := ParseSource(s); err != nil {
			t.Error(err)
		}
	}
	for _, s := range []string{"npm:invented", "npm:invented@latest", "npm:invented@^1.2.3", "npm:invented@1.2", "npm:invented@1.2.3-01", "git:invented@1.2.3", "npm:../escape@1.2.3", "npm:@team/../escape@1.2.3", "npm:invented@1.2.3;touch /tmp/x"} {
		if _, _, err := ParseSource(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
}

func TestIdentityValidateRejectsUnsafePaths(t *testing.T) {
	validRoot, project := t.TempDir(), t.TempDir()
	fileRoot := filepath.Join(t.TempDir(), "invented-root-file")
	if err := os.WriteFile(fileRoot, []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name string
		root string
	}{
		{name: "filesystem root", root: string(filepath.Separator)},
		{name: "noncanonical", root: filepath.Join(validRoot, "missing") + string(filepath.Separator) + ".."},
		{name: "existing non-directory", root: fileRoot},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id := Identity{Name: "invented-boundary", Scope: "global", Root: tt.root, AgentRoot: tt.root, Project: project}
			if err := id.Validate(); err == nil {
				t.Fatalf("accepted unsafe root %q", tt.root)
			}
		})
	}
}

func TestObserveRejectsMalformedTopLevelSettings(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "truncated object", body: `{`},
		{name: "null", body: `null`},
		{name: "array", body: `[]`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root, project := t.TempDir(), t.TempDir()
			op := Operation{Kind: "install", Source: "npm:invented-settings@1.2.3", Identity: Identity{Name: "invented-settings", Scope: "global", Root: root, AgentRoot: root, Project: project}}
			if err := os.WriteFile(op.Identity.SettingsPath(), []byte(tt.body), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := Observe(op); err == nil {
				t.Fatalf("accepted malformed top-level settings %s", tt.body)
			}
		})
	}
}

func TestObserveRejectsMismatchedSelectedMetadata(t *testing.T) {
	root, project := t.TempDir(), t.TempDir()
	op := Operation{Kind: "install", Source: "npm:invented-selected@1.2.3", Identity: Identity{Name: "invented-selected", Scope: "global", Root: root, AgentRoot: root, Project: project}}
	if err := os.MkdirAll(filepath.Dir(op.Identity.MetadataPath()), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(op.Identity.MetadataPath(), []byte(`{"name":"invented-other","version":"1.2.3"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Observe(op); err == nil {
		t.Fatal("accepted selected metadata for a different package")
	}
}

func TestObserveScopedMetadata(t *testing.T) {
	root, project := t.TempDir(), t.TempDir()
	op := Operation{Kind: "install", Source: "npm:@invented/plugin@1.2.3", Identity: Identity{Name: "@invented/plugin", Scope: "global", Root: root, AgentRoot: root, Project: project}}
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	check := func(want string) {
		t.Helper()
		got, err := Observe(op)
		if err != nil || got.Status != want {
			t.Fatalf("observation=%+v error=%v want=%s", got, err, want)
		}
	}
	check("missing")
	write(op.Identity.SettingsPath(), `{"packages":["npm:@invented/plugin@1.2.3"],"npmCommand":["operator-manager"]}`)
	check("declaration-only")
	write(op.Identity.MetadataPath(), `{"name":"@invented/plugin","version":"1.2.3"}`)
	check("present")
	write(op.Identity.MetadataPath(), `{"name":"@invented/plugin","version":"2.0.0"}`)
	check("drifted")
	write(op.Identity.SettingsPath(), `{"packages":[]}`)
	check("payload-only")
	for _, body := range []string{`{"packages":null}`, `{"packages":[],"packages":[]}`, `{"packages":[{},"npm:@invented/plugin@1.2.3"]}`, `{"packages":["npm:@invented/plugin@1.2.3","npm:@invented/plugin@2.0.0"]}`, `{"packages":[{"source":"npm:@invented/plugin@1.2.3","extensions":[]}]}`} {
		write(op.Identity.SettingsPath(), body)
		if _, err := Observe(op); err == nil {
			t.Errorf("accepted malformed/ambiguous settings %s", body)
		}
	}
	write(op.Identity.SettingsPath(), `{"packages":[]}`)
	if err := os.Remove(op.Identity.MetadataPath()); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "metadata.json")
	write(outside, `{"name":"@invented/plugin","version":"1.2.3"}`)
	if err := os.Symlink(outside, op.Identity.MetadataPath()); err != nil {
		t.Fatal(err)
	}
	if _, err := Observe(op); err == nil {
		t.Fatal("followed outside-root metadata symlink")
	}
}
