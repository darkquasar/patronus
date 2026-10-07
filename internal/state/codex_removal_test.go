package state_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/state"
)

func TestCodexLifecycleRefusesMissingPriorEvidence(t *testing.T) {
	for _, fields := range []string{
		``,
		`,"PriorValue":null`,
		`,"PriorPresent":false`,
		`,"PriorValue":null,"PriorPresent":null`,
		`,"PriorValue":null,"PriorPresent":"false"`,
		`,"PriorValue":"invented","PriorPresent":false`,
	} {
		t.Run(fields, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state.json")
			data := fmt.Sprintf(`{"version":2,"items":[{"artifact":"fixture-cx","tool":"codex","scope":"global","files":[{"path":"/fixture/config.toml","action":"MERGE","setting":{"Target":{"File":"config.toml","Format":"toml"},"Dotted":"mcp_servers.fixture.url","ScalarValue":"https://invented.invalid/mcp"%s}}]}]}`, fields)
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := state.Load(path); err == nil {
				t.Fatal("unsafe prior evidence accepted")
			}
			if got, err := os.ReadFile(path); err != nil || string(got) != data {
				t.Fatal("refusal mutated ownership")
			}
		})
	}
}

func TestCodexLifecycleLoadsExplicitPriorEvidence(t *testing.T) {
	for _, fields := range []string{`,"PriorValue":null,"PriorPresent":false`, `,"PriorValue":"invented-prior","PriorPresent":true`} {
		t.Run(fields, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state.json")
			data := fmt.Sprintf(`{"version":2,"items":[{"artifact":"fixture-cx","tool":"codex","files":[{"path":"/fixture/config.toml","action":"MERGE","setting":{"Target":{"File":"config.toml","Format":"toml"},"Dotted":"owned","ScalarValue":"invented"%s}}]}]}`, fields)
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := state.Load(path); err != nil {
				t.Fatal(err)
			}
		})
	}
}
