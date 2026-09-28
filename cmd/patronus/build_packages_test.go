package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/archive"
	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/registry"
)

const testPackageDescriptor = "schemaVersion: 1\nname: kit\nversion: 1.0.0\nplatforms:\n  - os: darwin\n    arch: arm64\npayload:\n  - path: spec.yaml\n    executable: false\n"

func packageFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"packages/kit", "artifacts", "adapters"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	packageWrite(t, root, "packages/kit/package.yaml", testPackageDescriptor)
	packageWrite(t, root, "packages/kit/spec.yaml", "name: demo\n")

	packageGit(t, root, "init", "-b", "main")
	packageGit(t, root, "config", "user.email", "test@example.invalid")
	packageGit(t, root, "config", "user.name", "Test")
	packageGit(t, root, "add", ".")
	packageGit(t, root, "commit", "-m", "fixture")
	return root
}
func packageWrite(t *testing.T, root, name, data string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}
func TestBuildPackageBootstrap(t *testing.T) {
	root := packageFixture(t)
	t.Chdir(root)
	out := t.TempDir()
	output, err := runBuild(t, "--package", "kit", "--out", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "sha256:") || !strings.Contains(output, "kit-1.0.0-darwin-arm64.tar.gz") {
		t.Fatal(output)
	}
	if _, err := os.Stat(filepath.Join(out, "catalog/index.json")); !os.IsNotExist(err) {
		t.Fatalf("selected build wrote index: %v", err)
	}
}

func TestBuildPackageSixteenComponentPayload(t *testing.T) {
	root := packageFixture(t)
	path := strings.Repeat("dir/", 15) + "spec.yaml"
	packageWrite(t, root, "packages/kit/package.yaml", strings.ReplaceAll(testPackageDescriptor, "spec.yaml", path))
	packageWrite(t, root, "packages/kit/"+path, "deep payload")
	t.Chdir(root)
	out := t.TempDir()
	if _, err := runBuild(t, "--package", "kit", "--out", out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "packages/kit/1.0.0/kit-1.0.0-darwin-arm64.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := packagebundle.Decode(bytes.NewReader(data), packagebundle.Identity{Name: "kit", Version: "1.0.0", OS: "darwin", Arch: "arm64"}, packagebundle.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Files) != 2 || bundle.Files[0].Path != "package.json" || bundle.Files[1].Path != path || string(bundle.Files[1].Data) != "deep payload" {
		t.Fatalf("unexpected payload: %+v", bundle.Files)
	}
}

func TestBuildPackagePinsAndIndex(t *testing.T) {
	root := packageFixture(t)
	t.Chdir(root)
	out := t.TempDir()
	built, err := buildPackage(root, "kit", out)
	if err != nil {
		t.Fatal(err)
	}
	recipe := func(sum, version, url string) string {
		return "apiVersion: patronus/v3\nkind: Recipe\nname: kit-recipe\nversion: 9.0.0\nfamily: recipe\nrole: sandbox\ndeliver:\n  via: fetch\n  unpack: directory\n  package:\n    name: kit\n    version: " + version + "\n  assets:\n    - os: darwin\n      arch: arm64\n      url: " + url + "\n      sha256: " + sum + "\n      archive: tar.gz\n"
	}
	url := "https://registry.test/" + built[0].Key
	packageWrite(t, root, "recipes/kit.yaml", recipe(built[0].SHA256, "1.0.0", url))
	if _, err := runBuild(t, "--out", out, "--base-url", "https://registry.test"); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(out, "catalog/index.json"))
	if err != nil {
		t.Fatal(err)
	}
	ix, err := registry.LoadIndex(original)
	if err != nil {
		t.Fatal(err)
	}
	if ix.SchemaVersion != 2 {
		t.Fatalf("schema=%d", ix.SchemaVersion)
	}
	for _, tc := range []struct{ name, sum, version, url string }{
		{"checksum", "sha256:" + strings.Repeat("0", 64), "1.0.0", url},
		{"version", built[0].SHA256, "1.0.1", url},
		{"url", built[0].SHA256, "1.0.0", "https://other.test/" + built[0].Key},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := recipe(tc.sum, tc.version, tc.url)
			packageWrite(t, root, "recipes/kit.yaml", content)
			if _, err := runBuild(t, "--out", out, "--base-url", "https://registry.test"); err == nil {
				t.Fatal("bad pin accepted")
			}
			after, _ := os.ReadFile(filepath.Join(out, "catalog/index.json"))
			if !bytes.Equal(after, original) {
				t.Fatal("failed build changed index")
			}
			after, _ = os.ReadFile(filepath.Join(root, "recipes/kit.yaml"))
			if string(after) != content {
				t.Fatal("build rewrote recipe")
			}
		})
	}
}
func TestBuildPackageSourceConfinement(t *testing.T) {
	for _, tc := range []struct{ name, path string }{
		{"traversal", "../secret"}, {"absolute", "/secret"}, {"metadata", "package.json"}, {"directory", "tests"}, {"glob", "*.yaml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := packageFixture(t)
			os.Mkdir(filepath.Join(root, "packages/kit/tests"), 0755)
			packageWrite(t, root, "packages/kit/package.yaml", strings.ReplaceAll(testPackageDescriptor, "spec.yaml", tc.path))
			if _, err := buildPackage(root, "kit", t.TempDir()); err == nil {
				t.Fatal("unsafe payload accepted")
			}
		})
	}
	for _, path := range []string{"packages/kit/spec.yaml", "packages/kit", "packages"} {
		t.Run("link "+path, func(t *testing.T) {
			root := packageFixture(t)
			target := filepath.Join(root, path)
			backup := target + "-real"
			if err := os.Rename(target, backup); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(backup, target); err != nil {
				t.Fatal(err)
			}
			if _, err := buildPackage(root, "kit", t.TempDir()); err == nil {
				t.Fatal("symlink accepted")
			}
		})
	}
}
func TestBuildPackageStrictDescriptor(t *testing.T) {
	for _, data := range []string{testPackageDescriptor + "unknown: true\n", testPackageDescriptor + "---\n{}\n", strings.ReplaceAll(testPackageDescriptor, "schemaVersion: 1", "schemaVersion: 2"), strings.ReplaceAll(testPackageDescriptor, "name: kit", "name: other")} {
		root := packageFixture(t)
		packageWrite(t, root, "packages/kit/package.yaml", data)
		if _, err := buildPackage(root, "kit", t.TempDir()); err == nil {
			t.Fatal("invalid descriptor accepted")
		}
	}
}
func TestBuildPackageSourceDigestAndProvenance(t *testing.T) {
	root := packageFixture(t)
	out := t.TempDir()
	a, err := buildPackage(root, "kit", out)
	if err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(out, a[0].Key))
	if err != nil {
		t.Fatal(err)
	}
	packageWrite(t, root, "packages/kit/tests/unshipped", "unshipped change")
	t.Setenv("GITHUB_RUN_ID", "different-run")
	if _, err := buildPackage(root, "kit", out); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(out, a[0].Key))
	if !bytes.Equal(first, second) {
		t.Fatal("unshipped input or CI run changed archive")
	}
	sidecar, _ := os.ReadFile(filepath.Join(out, a[0].Key+".provenance.json"))
	if !bytes.Contains(sidecar, []byte("different-run")) {
		t.Fatal("missing provenance")
	}
	packageWrite(t, root, "packages/kit/spec.yaml", "changed")
	b, err := buildPackage(root, "kit", out)
	if err != nil {
		t.Fatal(err)
	}
	if a[0].SHA256 == b[0].SHA256 {
		t.Fatal("shipped change did not affect package")
	}
}
func TestBuildPackageLegacyArchiveBytes(t *testing.T) {
	data, err := archive.CreateTarGz(map[string][]byte{"README.md": []byte("legacy\n"), "bin/tool": []byte("payload\n")})
	if err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprintf("%x", sha256.Sum256(data))
	const want = "864bca6a55925c1d13d4e6a15769ccabb8d6d9a35806231671a6144d2e8471ae"
	if got != want {
		t.Fatalf("legacy archive bytes changed: %s", got)
	}
}

func TestBuildPackageCanonicalSourceDigest(t *testing.T) {
	root := packageFixture(t)
	d, files, err := loadPackageSource(root, "kit")
	if err != nil {
		t.Fatal(err)
	}
	a, err := packageSourceDigest(d, files)
	if err != nil {
		t.Fatal(err)
	}
	// YAML presentation and undeclared filesystem permissions do not change shipped inputs.
	packageWrite(t, root, "packages/kit/package.yaml", "# comment\n"+testPackageDescriptor)
	if err := os.Chmod(filepath.Join(root, "packages/kit/spec.yaml"), 0755); err != nil {
		t.Fatal(err)
	}
	d, files, err = loadPackageSource(root, "kit")
	if err != nil {
		t.Fatal(err)
	}
	b, err := packageSourceDigest(d, files)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("non-shipped presentation/mode changed source digest")
	}
	d.Payload[0].Executable = true
	files[0].Mode = 0755
	b, err = packageSourceDigest(d, files)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("declared executable mode did not change source digest")
	}
}
