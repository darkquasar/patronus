package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Persisted-bytes goldens (HP-05): exact patronus.lock and state.json bytes for
// invented Codex and Pi installs. Timestamps are normalized to <ts>; $H and $P
// stand for the temporary home and catalog root. These cover the persistence
// paths HS-T10 re-threads (state.LoadWithPolicy, ReconcileInput.Policy,
// FromResolvedWithPolicy, RefreshSectionChecksumsWithPolicy).

func harnessNormalizeBytes(raw []byte, norm func(string) string) string {
	timestamp := regexp.MustCompile(`"(generated|installedAt)": "[0-9TZ:-]+"`)
	return timestamp.ReplaceAllString(norm(string(raw)), `"$1": "<ts>"`)
}

const harnessLockEntry = "  \"entries\": [\n" +
	"    {\n" +
	"      \"name\": \"fixture-any\",\n" +
	"      \"source\": \"registry\",\n" +
	"      \"version\": \"1.0.0\",\n" +
	"      \"sha256\": \"sha256:ee99e7eddaa8c8423c26f3a422958548499f0468b322226a397644502284867b\",\n" +
	"      \"slot\": \"capabilities\",\n" +
	"      \"kind\": \"artifact\"\n" +
	"    }\n" +
	"  ]\n" +
	"}\n"

func TestHarnessPersistedGoldenLock(t *testing.T) {
	for _, tc := range []struct {
		target string
		want   string
	}{
		{"codex", "{\n  \"version\": 3,\n  \"target\": \"codex\",\n  \"profile\": \"fixture-any-profile\",\n  \"generated\": \"<ts>\",\n" + harnessLockEntry},
		{"pi", "{\n  \"version\": 3,\n  \"target\": \"pi\",\n  \"profile\": \"fixture-any-profile\",\n  \"generated\": \"<ts>\",\n" + harnessLockEntry},
		{"all", "{\n  \"version\": 2,\n  \"profile\": \"fixture-any-profile\",\n  \"generated\": \"<ts>\",\n" + harnessLockEntry},
	} {
		t.Run(tc.target, func(t *testing.T) {
			_, root, norm := harnessOutputFixture(t)
			if _, _, err := runLock(t, "--profile", "fixture-any-profile", "--target", tc.target); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(root, "patronus.lock"))
			if err != nil {
				t.Fatal(err)
			}
			if got := harnessNormalizeBytes(raw, norm); got != tc.want {
				t.Fatalf("lock bytes (%s)\n got: %s\nwant: %s", tc.target, got, tc.want)
			}
		})
	}
}

func TestHarnessPersistedGoldenCodexState(t *testing.T) {
	home, _, norm := harnessOutputFixture(t)
	if out, errOut, err := runInstall(t, "fixture-ctx", "fix-mcp-two", "--target", "codex", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("%v\n%s\n%s", err, out, errOut)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".patronus/state.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"{",
		"  \"version\": 2,",
		"  \"items\": [",
		"    {",
		"      \"artifact\": \"fixture-ctx\",",
		"      \"itemVersion\": \"1.0.0\",",
		"      \"type\": \"instruction\",",
		"      \"tool\": \"codex\",",
		"      \"scope\": \"global\",",
		"      \"installedAt\": \"<ts>\",",
		"      \"files\": [",
		"        {",
		"          \"path\": \"$H/.codex/AGENTS.md\",",
		"          \"action\": \"APPEND\",",
		"          \"checksum\": \"sha256:eb1045541c108931f85db7547ca4ba26d5f0246f44aa2134516900626e214363\",",
		"          \"section\": \"fixture-ctx\"",
		"        }",
		"      ]",
		"    },",
		"    {",
		"      \"artifact\": \"fix-mcp-two\",",
		"      \"itemVersion\": \"1.0.0\",",
		"      \"type\": \"wire-only\",",
		"      \"tool\": \"codex\",",
		"      \"scope\": \"global\",",
		"      \"installedAt\": \"<ts>\",",
		"      \"files\": [",
		"        {",
		"          \"path\": \"$H/.codex/config.toml\",",
		"          \"action\": \"MERGE\",",
		"          \"checksum\": \"sha256:2269fa15c0c840589813599de7fc4579a07f9899c953bd23e2307abdb59d53de\",",
		"          \"setting\": {",
		"            \"Target\": {",
		"              \"File\": \"~/.codex/config.toml\",",
		"              \"Format\": \"toml\"",
		"            },",
		"            \"Dotted\": \"mcp_servers.fix-mcp-two.url\",",
		"            \"IdentityKey\": \"\",",
		"            \"Identity\": \"\",",
		"            \"Elem\": null,",
		"            \"ScalarValue\": \"https://fixture.invalid/mcp/\",",
		"            \"PriorValue\": null,",
		"            \"PriorPresent\": false",
		"          }",
		"        }",
		"      ]",
		"    }",
		"  ]",
		"}",
		"",
	}, "\n")
	if got := harnessNormalizeBytes(raw, norm); got != want {
		t.Fatalf("codex state bytes\n got: %s\nwant: %s", got, want)
	}
}

func TestHarnessPersistedGoldenPiState(t *testing.T) {
	home, _, norm := harnessOutputFixture(t)
	if out, errOut, err := runInstall(t, "fixture-ctx", "--target", "pi", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("%v\n%s\n%s", err, out, errOut)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".patronus/state.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"{",
		"  \"version\": 2,",
		"  \"items\": [",
		"    {",
		"      \"root\": \"$H/.pi/agent\",",
		"      \"artifact\": \"fixture-ctx\",",
		"      \"itemVersion\": \"1.0.0\",",
		"      \"type\": \"instruction\",",
		"      \"tool\": \"pi\",",
		"      \"scope\": \"global\",",
		"      \"installedAt\": \"<ts>\",",
		"      \"files\": [",
		"        {",
		"          \"path\": \"$H/.pi/agent/AGENTS.md\",",
		"          \"action\": \"APPEND\",",
		"          \"checksum\": \"sha256:bf69e73b5f334d377b26ecac78c6d8031a63d0b1f15b5cf8808c6d506e29d8e3\",",
		"          \"section\": \"pi:fixture-ctx\"",
		"        }",
		"      ]",
		"    }",
		"  ]",
		"}",
		"",
	}, "\n")
	if got := harnessNormalizeBytes(raw, norm); got != want {
		t.Fatalf("pi state bytes\n got: %s\nwant: %s", got, want)
	}
}
