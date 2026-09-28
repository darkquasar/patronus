package packagebundle

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildDeterministic(t *testing.T) {
	id := Identity{Name: "kit", Version: "1.0.0", OS: "darwin", Arch: "arm64"}
	files := []File{{Path: "spec.yaml", Mode: 0644, Data: []byte("name: demo\n")}, {Path: "bin/run", Mode: 0755, Data: []byte("hello")}}
	source := "sha256:" + strings.Repeat("1", 64)
	a, err := Build(id, source, files)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build(id, source, []File{files[1], files[0]})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("nondeterministic archive")
	}
	bundle, err := Decode(bytes.NewReader(a), id, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Files) != 3 || bundle.Metadata.Files[0].Mode != 0755 {
		t.Fatalf("wrong inventory: %+v", bundle)
	}
	if files[0].Path != "spec.yaml" {
		t.Fatal("input mutated")
	}
}
func TestBuildUSTARPaths(t *testing.T) {
	id := Identity{Name: "kit", Version: "1.0.0", OS: "linux", Arch: "amd64"}
	source := "sha256:" + strings.Repeat("1", 64)
	for _, name := range []string{strings.Repeat("a", 101), strings.Repeat("a", 156) + "/" + strings.Repeat("b", 100)} {
		if _, err := Build(id, source, []File{{Path: name, Mode: 0644}}); err == nil {
			t.Fatalf("accepted %d-byte path", len(name))
		}
	}
	name := strings.Repeat("a", 155) + "/" + strings.Repeat("b", 100)
	if _, err := Build(id, source, []File{{Path: name, Mode: 0644}}); err != nil {
		t.Fatal(err)
	}
}

func TestBuildRejectsInvalidPayloads(t *testing.T) {
	id := Identity{Name: "kit", Version: "1.0.0", OS: "linux", Arch: "amd64"}
	source := "sha256:" + strings.Repeat("1", 64)
	for _, tc := range []struct {
		name  string
		files []File
	}{
		{"metadata", []File{{Path: "package.json", Mode: 0644}}},
		{"duplicate", []File{{Path: "a", Mode: 0644}, {Path: "a", Mode: 0644}}},
		{"case collision", []File{{Path: "Dir/a", Mode: 0644}, {Path: "dir/b", Mode: 0644}}},
		{"ancestor file", []File{{Path: "a", Mode: 0644}, {Path: "a/b", Mode: 0644}}},
		{"privileged mode", []File{{Path: "a", Mode: 04755}}},
		{"noncanonical mode", []File{{Path: "a", Mode: 0700}}},
		{"traversal", []File{{Path: "../a", Mode: 0644}}},
		{"large file", []File{{Path: "a", Mode: 0644, Data: make([]byte, (16<<20)+1)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Build(id, source, tc.files); err == nil {
				t.Fatal("invalid payload accepted")
			}
		})
	}
}
