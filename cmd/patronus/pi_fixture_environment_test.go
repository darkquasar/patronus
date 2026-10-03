package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Both CLI fixture entry points must close inherited tool-root overrides before
// any Pi write, not just replace HOME. The sentinel simulates an ambient kit.
func TestPiFixtureEnvironmentIsolation(t *testing.T) {
	for _, helper := range []string{"withRemoteEnv", "newDirectoryFixture"} {
		t.Run(helper, func(t *testing.T) {
			sentinel := t.TempDir()
			for _, rel := range []string{"settings.json", "skills/sample-isolation/SKILL.md", "nested/keep.bin"} {
				path := filepath.Join(sentinel, rel)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("untouched sentinel: "+rel+"\x00\xff"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PI_CODING_AGENT_DIR", sentinel)
			t.Setenv("PI_PACKAGE_DIR", filepath.Join(sentinel, "unqualified-host"))
			t.Setenv("PI_SUBAGENT_EXTRA_AGENT_DIRS", filepath.Join(sentinel, "runtime-agents"))
			t.Setenv("PI_MCP_CONFIG_MODE", "exclusive")
			t.Setenv("PI_OFFLINE", "false")
			before := dp01SnapshotFiles(t, sentinel)
			var home string
			switch helper {
			case "withRemoteEnv":
				root := fixtureCatalog(t)
				dp01WriteIsolationSkill(t, root)
				t.Chdir(root)
				out := t.TempDir()
				if _, err := runBuild(t, "--out", out, "--base-url", testRegistryBase); err != nil {
					t.Fatal(err)
				}
				home = withRemoteEnv(t, serveTree(t, out))
			case "newDirectoryFixture":
				f := newDirectoryFixture(t)
				home = f.home
				dp01WriteIsolationSkill(t, f.root)
			}
			for key, want := range map[string]string{"PI_PACKAGE_DIR": "", "PI_SUBAGENT_EXTRA_AGENT_DIRS": "", "PI_MCP_CONFIG_MODE": "", "PI_OFFLINE": "true"} {
				if got := os.Getenv(key); got != want {
					t.Errorf("helper left %s=%q, want %q", key, got, want)
				}
			}
			wantRoot := filepath.Join(home, ".pi", "agent")
			if got := os.Getenv("PI_CODING_AGENT_DIR"); got != wantRoot {
				t.Fatalf("helper left Pi root %q, want %q", got, wantRoot)
			}
			if _, _, err := runInstall(t, "sample-isolation", "--target", "pi", "--global", "--deploy"); err != nil {
				t.Fatal(err)
			}
			if got := string(mustRead(t, filepath.Join(wantRoot, "skills/sample-isolation/SKILL.md"))); got != dp01IsolationSkill {
				t.Fatalf("installed bytes = %q", got)
			}
			if after := dp01SnapshotFiles(t, sentinel); !reflect.DeepEqual(after, before) {
				t.Fatalf("inherited Pi tree changed: before %v, after %v", before, after)
			}
		})
	}
}

const dp01IsolationSkill = "---\nname: sample-isolation\ndescription: Invented isolation fixture\n---\nFixture only.\n"

func dp01WriteIsolationSkill(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "artifacts", "sample-isolation")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"patronus.yaml": "apiVersion: patronus/v2\nfamily: artifact\nname: sample-isolation\nversion: 1.0.0\nrole: capability\ndescription: Invented isolation fixture\ntype: skill\nentry: SKILL.md\ntargets: [pi]\n",
		"SKILL.md":      dp01IsolationSkill,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func dp01SnapshotFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
