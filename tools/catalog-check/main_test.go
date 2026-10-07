package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixture struct {
	root, index string
	manifest    map[string]any
	members     map[string]string
}

func writeFixtureFile(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, data, 0644); err != nil {
		t.Fatal(err)
	}
}
func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	f := &fixture{root: root, index: filepath.Join(root, "out/catalog/index.json"), manifest: map[string]any{"apiVersion": "patronus/v2", "family": "artifact", "type": "skill", "role": "capability", "name": "sample", "version": "1.0.0", "description": "Invented skill", "entry": "SKILL.md", "files": []string{"notes.txt"}, "targets": []string{"toy"}, "defaults": map[string]any{"scope": "project"}}, members: map[string]string{"SKILL.md": "---\nname: sample\ndescription: Invented skill\n---\nInert body.\n", "notes.txt": "inert sidecar\n"}}
	for name, body := range f.members {
		writeFixtureFile(t, filepath.Join(root, "artifacts/skills/sample", name), []byte(body))
	}
	manifestJSON, _ := json.Marshal(f.manifest)
	writeFixtureFile(t, filepath.Join(root, "artifacts/skills/sample/patronus.yaml"), manifestJSON)
	writeFixtureFile(t, filepath.Join(root, "profiles/invented.yaml"), []byte("apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: invented\nversion: 1.0.0\nlayers:\n  capabilities: [sample]\n"))
	writeFixtureFile(t, filepath.Join(root, "adapters/toy.yaml"), []byte("tool: toy\nfamily: adapter\n"))
	writeFixtureFile(t, filepath.Join(root, "docs/compatibility/profile-targets.yaml"), []byte("schema_version: 1\nprofiles:\n  invented:\n    targets: [toy]\n"))
	f.publish(t, nil)
	return f
}
func (f *fixture) publish(t *testing.T, extra *tar.Header) {
	t.Helper()
	manifestJSON, _ := json.Marshal(f.manifest)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	members := map[string]string{"patronus.yaml": string(manifestJSON)}
	for name, body := range f.members {
		members[name] = body
	}
	for name, body := range members {
		if err := tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(body)), Mode: 0644, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if extra != nil {
		if err := tw.WriteHeader(extra); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(filepath.Dir(f.index), "sample/1.0.0/sample-1.0.0.tar.gz"), buf.Bytes())
	ix := map[string]any{"schemaVersion": 1, "artifacts": []any{map[string]any{"manifest": f.manifest, "tarball": map[string]any{"url": "https://never-fetched.invalid/catalog/sample/1.0.0/sample-1.0.0.tar.gz", "sha256": fmt.Sprintf("sha256:%x", sha256.Sum256(buf.Bytes()))}}}, "profiles": []any{map[string]any{"manifest": map[string]any{"apiVersion": "patronus/v2", "family": "profile", "role": "lifecycle", "name": "invented", "version": "1.0.0", "layers": map[string]any{"capabilities": []string{"sample"}}}}}}
	data, _ := json.Marshal(ix)
	writeFixtureFile(t, f.index, data)
	writeFixtureFile(t, f.index+".sha256", []byte(fmt.Sprintf("sha256:%x\n", sha256.Sum256(data))))
}
func (f *fixture) check(t *testing.T, extra ...string) (int, string, string) {
	t.Helper()
	var out, err bytes.Buffer
	code := run(append([]string{"--source", f.root, "--index", f.index}, extra...), &out, &err)
	return code, out.String(), err.String()
}
func TestRejectsUnsafeArchiveMembers(t *testing.T) {
	for _, header := range []*tar.Header{{Name: "../escape", Typeflag: tar.TypeReg}, {Name: "notes.txt", Typeflag: tar.TypeReg}, {Name: "link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"}} {
		t.Run(header.Name, func(t *testing.T) {
			f := newFixture(t)
			f.publish(t, header)
			code, _, diagnostic := f.check(t)
			if code == 0 || !strings.Contains(diagnostic, "archive") {
				t.Fatalf("unsafe archive accepted: code=%d diagnostic=%s", code, diagnostic)
			}
		})
	}
}
