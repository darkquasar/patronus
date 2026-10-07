package main

import (
	"os"
	"strings"
	"testing"
)

// TestCodexMCPReportsMissingAuthPrerequisite proves that explicitly configured
// Codex MCP auth/environment references are checked for presence only and a
// missing one is reported as an actionable prerequisite on install and update.
// Unauthenticated servers are not assumed to need credentials. Every name and
// value is invented; no real auth file, value or request is involved.
func TestCodexMCPReportsMissingAuthPrerequisite(t *testing.T) {
	const headerEnv = "INVENTED_HEADER_ENV"
	const secretValue = "invented-secret-value-not-for-output"
	unset := func(t *testing.T, names ...string) {
		for _, n := range names {
			t.Setenv(n, "")
			os.Unsetenv(n)
		}
	}
	root, _, config := codexMCPFixture(t)
	unset(t, codexMCPTokenEnv, headerEnv)

	_, errOut, err := runInstall(t, codexMCPServer, "--target", "codex", "--global", "--deploy")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(errOut, "prerequisite") {
		t.Fatalf("unauthenticated server assumed to need credentials:\n%s", errOut)
	}

	codexAddUserAuth(t, config)
	for _, run := range []struct {
		name string
		fn   func() (string, string, error)
	}{
		{"install dry-run", func() (string, string, error) {
			return runInstall(t, codexMCPServer, "--target", "codex", "--global")
		}},
		{"install deploy", func() (string, string, error) {
			return runInstall(t, codexMCPServer, "--target", "codex", "--global", "--deploy")
		}},
		{"update", func() (string, string, error) {
			codexBumpMCPRecipe(t, root)
			return runUpdate(t, codexMCPServer, "--deploy")
		}},
	} {
		_, errOut, err := run.fn()
		if err != nil {
			t.Fatalf("%s: %v", run.name, err)
		}
		for _, want := range []string{"Codex MCP server \"" + codexMCPServer + "\"", "bearer_token_env_var", codexMCPTokenEnv, "env_http_headers", headerEnv, "not set", "export"} {
			if !strings.Contains(errOut, want) {
				t.Fatalf("%s: prerequisite report missing %q:\n%s", run.name, want, errOut)
			}
		}
	}

	t.Setenv(codexMCPTokenEnv, secretValue)
	t.Setenv(headerEnv, secretValue)
	out, errOut, err := runInstall(t, codexMCPServer, "--target", "codex", "--global")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(errOut, "prerequisite") {
		t.Fatalf("present references still reported missing:\n%s", errOut)
	}
	if strings.Contains(out+errOut, secretValue) {
		t.Fatal("environment value leaked into output")
	}
}
