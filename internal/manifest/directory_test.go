package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func directoryYAML() string {
	return `apiVersion: patronus/v3
family: recipe
role: sandbox
name: kit
version: 1.0.0
deliver:
  via: fetch
  unpack: directory
  package: {name: payload, version: 2.0.0}
  assets:
    - os: darwin
      arch: arm64
      url: https://example.test/kit.tar.gz
      sha256: "` + strings.Repeat("0", 64) + `"
      archive: tar.gz
`
}

func TestDirectoryValid(t *testing.T) {
	for _, data := range []string{directoryYAML(), strings.Replace(directoryYAML(), "os: darwin", "os: linux", 1), strings.Replace(directoryYAML(), strings.Repeat("0", 64), "sha256:"+strings.Repeat("a", 64), 1)} {
		r, err := DecodeRecipe([]byte(data))
		if err != nil {
			t.Fatal(err)
		}
		if r.Shape() != ShapeInstall {
			t.Fatalf("shape = %s", r.Shape())
		}
	}
}

func TestDirectoryRejectsInvalidFields(t *testing.T) {
	cases := []struct{ name, old, replacement string }{
		{"v2", "patronus/v3", "patronus/v2"},
		{"future", "patronus/v3", "patronus/v4"},
		{"no directory", "  unpack: directory\n", ""},
		{"unknown unpack", "unpack: directory", "unpack: zip"},
		{"wrong role", "role: sandbox", "role: memory"},
		{"binary path", "archive: tar.gz", "archive: tar.gz\n      binaryPath: bin/kit"},
		{"binary", "  unpack: directory", "  binary: kit\n  unpack: directory"},
		{"install target", "  unpack: directory", "  installTo: /tmp/kit\n  unpack: directory"},
		{"direct URL", "  unpack: directory", "  url: https://example.test/kit.tar.gz\n  unpack: directory"},
		{"direct digest", "  unpack: directory", "  sha256: abc\n  unpack: directory"},
		{"install candidate", "  unpack: directory", "  install: [{manager: npm}]\n  unpack: directory"},
		{"platform gate", "  unpack: directory", "  platforms: [darwin]\n  unpack: directory"},
		{"script", "via: fetch", "via: script"},
		{"identity", "  package: {name: payload, version: 2.0.0}\n", ""},
		{"assets", "  assets:", "  unused:"},
		{"local scope", "family: recipe", "family: recipe\nscope: {marker: .kit}"},
		{"exec", "family: recipe", "family: recipe\nwire: {method: exec, actor: patronus, run: [echo]}"},
		{"merge", "family: recipe", "family: recipe\nwire: {method: merge, actor: patronus, mcp: {transport: http}}"},
		{"tools", "family: recipe", "family: recipe\nwire: {tools: [claude]}"},
		{"run without method", "family: recipe", "family: recipe\nwire: {run: [echo]}"},
		{"short digest", strings.Repeat("0", 64), "abc"},
		{"nonhex digest", strings.Repeat("0", 64), strings.Repeat("g", 64)},
		{"http", "https://", "http://"},
		{"empty host", "https://example.test/kit.tar.gz", "https:///kit.tar.gz"},
		{"zip", "archive: tar.gz", "archive: zip"},
		{"tgz", "archive: tar.gz", "archive: tgz"},
		{"no archive", "      archive: tar.gz\n", ""},
		{"unknown OS", "os: darwin", "os: plan9"},
		{"unknown arch", "arch: arm64", "arch: mips"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodeRecipe([]byte(strings.Replace(directoryYAML(), tc.old, tc.replacement, 1))); err == nil {
				t.Fatal("invalid directory recipe accepted")
			}
		})
	}
}

func TestDirectoryNamesAndVersions(t *testing.T) {
	for _, field := range []string{"name: kit", "name: payload"} {
		for _, name := range []string{"Kit", ".", "..", ".txn", ".kit", "a/b", `a\b`, "a b", "é"} {
			t.Run(field+"/"+name, func(t *testing.T) {
				data := strings.Replace(directoryYAML(), field, "name: "+name, 1)
				if _, err := DecodeRecipe([]byte(data)); err == nil {
					t.Fatal("unsafe name accepted")
				}
			})
		}
	}
	for _, field := range []string{"version: 1.0.0", "version: 2.0.0"} {
		for _, version := range []string{"1", "v1.2.3", "01.2.3", "1.02.3", "1.2.03", "1.2.3-01", "1.2.3-a..b", "1.2.3+", "1.2.3-", "1.2.3+é"} {
			t.Run(field+"/"+version, func(t *testing.T) {
				if _, err := DecodeRecipe([]byte(strings.Replace(directoryYAML(), field, "version: "+version, 1))); err == nil {
					t.Fatal("invalid SemVer accepted")
				}
			})
		}
	}
}

func TestDirectoryDuplicatePlatform(t *testing.T) {
	data := directoryYAML()
	asset := data[strings.Index(data, "    - os:"):]
	if _, err := DecodeRecipe([]byte(data + asset)); err == nil {
		t.Fatal("duplicate platform accepted")
	}
}

func TestDirectoryMetadataRejectedOnV2File(t *testing.T) {
	data := strings.Replace(directoryYAML(), "patronus/v3", "patronus/v2", 1)
	data = strings.Replace(data, "  unpack: directory\n", "", 1)
	if _, err := DecodeRecipe([]byte(data)); err == nil {
		t.Fatal("file delivery accepted package identity")
	}
}

func TestDirectoryPackageVersionVocabulary(t *testing.T) {
	for _, v := range []string{"0.0.0", "1.2.3-alpha.1", "1.2.3-0a.0+build.01", "123456789012345678901234567890.0.0", "1.2.3+001"} {
		if !ValidPackageVersion(v) {
			t.Errorf("valid SemVer %q rejected", v)
		}
	}
	for _, v := range []string{"", " 1.2.3", "1.2.3\n", "1.2.3-00", "1.2.3-α", "1.2.3+a_b", "1.2.3.4", "1.2.3+build..1"} {
		if ValidPackageVersion(v) {
			t.Errorf("invalid SemVer %q accepted", v)
		}
	}
}

func TestAPIVersionVocabulary(t *testing.T) {
	for _, v := range []string{"patronus/v2", "patronus/v3"} {
		if !SupportsAPIVersion(v) {
			t.Errorf("supported version %q rejected", v)
		}
	}
	for _, v := range []string{"", "patronus/v1", "patronus/v4"} {
		if SupportsAPIVersion(v) {
			t.Errorf("unsupported version %q accepted", v)
		}
	}
}

func TestDirectoryLegacyFileDeliveriesUnchanged(t *testing.T) {
	for _, delivery := range []string{
		"deliver: {via: fetch, url: 'https://example.test/script', sha256: legacy-pin}\n",
		"deliver:\n  via: fetch\n  assets: [{os: darwin, arch: arm64, url: 'https://example.test/binary', sha256: legacy-pin, archive: zip, binaryPath: bin/kit}]\n",
	} {
		data := "apiVersion: patronus/v2\nfamily: recipe\nrole: sandbox\nname: Legacy\nversion: legacy-version\n" + delivery
		if _, err := DecodeRecipe([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDirectoryLoadRecipe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recipe.yaml")
	if err := os.WriteFile(path, []byte(directoryYAML()+"scope: {global: '~/.patronus/packages/kit'}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := LoadRecipe(path)
	if err != nil {
		t.Fatal(err)
	}
	if r.Delivery.Package.Name != "payload" {
		t.Fatal("package identity lost")
	}
	if err := os.WriteFile(path, []byte(strings.Replace(directoryYAML(), "https://", "http://", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRecipe(path); err == nil {
		t.Fatal("local loader accepted malformed directory delivery")
	}
}
