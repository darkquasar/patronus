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
	"sort"
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

// publishSecond adds a second invented skill artifact to the source tree and
// republishes the index with both artifacts, so tests can build closures that
// differ in membership.
func (f *fixture) publishSecond(t *testing.T, name string) {
	t.Helper()
	manifest := map[string]any{}
	for k, v := range f.manifest {
		manifest[k] = v
	}
	manifest["name"] = name
	manifest["files"] = []string{}
	members := map[string]string{"SKILL.md": "---\nname: " + name + "\ndescription: Invented skill\n---\nInert body.\n"}
	for member, body := range members {
		writeFixtureFile(t, filepath.Join(f.root, "artifacts/skills", name, member), []byte(body))
	}
	manifestJSON, _ := json.Marshal(manifest)
	writeFixtureFile(t, filepath.Join(f.root, "artifacts/skills", name, "patronus.yaml"), manifestJSON)
	tarball := fixtureTarball(t, manifestJSON, members)
	writeFixtureFile(t, filepath.Join(filepath.Dir(f.index), name, "1.0.0", name+"-1.0.0.tar.gz"), tarball)
	firstJSON, _ := json.Marshal(f.manifest)
	first := fixtureTarball(t, firstJSON, f.members)
	writeFixtureFile(t, filepath.Join(filepath.Dir(f.index), "sample/1.0.0/sample-1.0.0.tar.gz"), first)
	entry := func(m map[string]any, raw []byte, n string) map[string]any {
		return map[string]any{"manifest": m, "tarball": map[string]any{"url": "https://never-fetched.invalid/catalog/" + n + "/1.0.0/" + n + "-1.0.0.tar.gz", "sha256": fmt.Sprintf("sha256:%x", sha256.Sum256(raw))}}
	}
	ix := map[string]any{"schemaVersion": 1, "artifacts": []any{entry(f.manifest, first, "sample"), entry(manifest, tarball, name)}, "profiles": []any{map[string]any{"manifest": map[string]any{"apiVersion": "patronus/v2", "family": "profile", "role": "lifecycle", "name": "invented", "version": "1.0.0", "layers": map[string]any{"capabilities": []string{"sample"}}}}}}
	data, _ := json.Marshal(ix)
	writeFixtureFile(t, f.index, data)
	writeFixtureFile(t, f.index+".sha256", []byte(fmt.Sprintf("sha256:%x\n", sha256.Sum256(data))))
}

// fixtureTarball builds a deterministic tarball (sorted members) of a manifest
// and its members.
func fixtureTarball(t *testing.T, manifestJSON []byte, members map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	all := map[string]string{"patronus.yaml": string(manifestJSON)}
	for name, body := range members {
		all[name] = body
	}
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		body := all[name]
		if err := tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(body)), Mode: 0644, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
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
