package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/state"
)

func TestCodexInstructionIncrementalInstallKeepsSiblingOwnership(t *testing.T) {
	root := fixtureCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Chdir(root)
	for _, name := range []string{"fix-instruction-global", "fix-instruction-2", "fix-instruction-global"} {
		if _, _, err := runInstall(t, name, "--target", "codex", "--global", "--deploy"); err != nil {
			t.Fatalf("incremental install/reinstall %s: %v", name, err)
		}
	}
	assertCodexInstructionChecksums(t, home, 2)

	path := filepath.Join(home, ".codex/AGENTS.md")
	sp := filepath.Join(home, ".patronus/state.json")
	ownership := mustRead(t, sp)
	edited := append(mustRead(t, path), []byte("User edit\n")...)
	codexWrite(t, path, edited)
	if _, _, err := runInstall(t, "fix-instruction-2", "--target", "codex", "--global", "--deploy", "--force"); err == nil || !strings.Contains(err.Error(), "drift") {
		t.Fatalf("user edit must still block even with force: %v", err)
	}
	if !bytes.Equal(mustRead(t, path), edited) || !bytes.Equal(mustRead(t, sp), ownership) {
		t.Fatal("drift refusal changed file or ownership")
	}
}

func TestCodexCoreInstructionSequentialUpdatesKeepSiblingOwnership(t *testing.T) {
	root := fixtureCatalog(t)
	names := []string{"fix-instruction-global", "fix-instruction-2", "fix-context-a", "fix-context-b", "fix-context-c", "fix-context-d", "fix-context-e"}
	for _, name := range names[2:] {
		dir := filepath.Join(root, "artifacts/instructions", name)
		codexWrite(t, filepath.Join(dir, "patronus.yaml"), []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: instruction\nrole: instruction\nname: "+name+"\ndescription: Invented sequential ownership fixture\nversion: 1.0.0\nentry: INSTRUCTIONS.md\ntargets: [codex]\n"))
		codexWrite(t, filepath.Join(dir, "INSTRUCTIONS.md"), []byte("Invented "+name+" context\n"))
	}
	isolated := root
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Chdir(isolated)
	args := append(append([]string(nil), names...), "--target", "codex", "--global", "--deploy", "--local-registry")
	if _, _, err := runInstall(t, args...); err != nil {
		t.Fatal(err)
	}
	for _, name := range names[:2] {
		mp := filepath.Join(isolated, "artifacts/instructions", name, "patronus.yaml")
		codexWrite(t, mp, []byte(strings.ReplaceAll(string(mustRead(t, mp)), "version: 1.0.0", "version: 2.0.0")))
		body := filepath.Join(isolated, "artifacts/instructions", name, "INSTRUCTIONS.md")
		codexWrite(t, body, append(mustRead(t, body), []byte("\nInvented update regression.\n")...))
		if _, _, err := runUpdate(t, name, "--target", "codex", "--deploy", "--local-registry"); err != nil {
			t.Fatalf("sequential core instruction update %s: %v", name, err)
		}
		assertCodexInstructionChecksums(t, home, len(names))
	}
}

func assertCodexInstructionChecksums(t *testing.T, home string, count int) {
	t.Helper()
	path := filepath.Join(home, ".codex/AGENTS.md")
	s, err := state.Load(filepath.Join(home, ".patronus/state.json"))
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, item := range s.Items {
		for _, f := range item.Files {
			if f.Path == path && f.Section != "" {
				found++
				if f.Checksum != codexChecksum(mustRead(t, path)) {
					t.Fatalf("stale instruction checksum for %s: %s", item.Artifact, f.Checksum)
				}
			}
		}
	}
	if found != count {
		t.Fatalf("instruction owners=%d, want %d", found, count)
	}
}

func TestCodexLockAdmitsTargetAgnosticRecipeInstallAndUpdate(t *testing.T) {
	testCodexLockRecipeInstallAndUpdate(t, nil)
}

func TestCodexLockAdmitsTargetAgnosticRecipeUpdate(t *testing.T) {
	testCodexLockRecipeInstallAndUpdate(t, []string{"--target", "codex"})
}

func testCodexLockRecipeInstallAndUpdate(t *testing.T, targetArgs []string) {
	t.Helper()
	root := fixtureCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(root)
	lp := filepath.Join(root, "patronus.lock")
	if err := lock.Save(lp, &lock.Lock{Version: lock.TargetVersion, Target: "codex", Profile: "invented-codex-profile"}); err != nil {
		t.Fatal(err)
	}
	priorLock := mustRead(t, lp)
	oldFetcher := fetcherForDeploy
	t.Cleanup(func() { fetcherForDeploy = oldFetcher })
	fetcherForDeploy = &servingFetcher{bodies: map[string][]byte{fixRawURL: fixRawBinary}}
	installArgs := append([]string{"fix-bin", "--global", "--deploy"}, targetArgs...)
	if _, _, err := runInstall(t, installArgs...); err != nil {
		t.Fatalf("target-free fetch-only recipe install with Codex lock: %v", err)
	}
	mp := filepath.Join(root, "recipes/fix-bin.yaml")
	codexWrite(t, mp, []byte(strings.ReplaceAll(string(mustRead(t, mp)), "version: 1.0.0", "version: 2.0.0")))
	if _, _, err := runUpdate(t, "fix-bin", "--target", "codex", "--deploy"); err != nil {
		t.Fatalf("target-agnostic recipe refresh with Codex lock: %v", err)
	}
	s, err := state.Load(filepath.Join(home, ".patronus/state.json"))
	if err != nil {
		t.Fatal(err)
	}
	got := s.Find("fix-bin", recipe.TargetAgnostic, "global")
	if len(got) != 1 || got[0].ItemVersion != "2.0.0" {
		t.Fatalf("recipe did not refresh its agnostic row: %+v", got)
	}
	if !bytes.Equal(mustRead(t, lp), priorLock) {
		t.Fatal("recipe refresh overwrote Codex lock")
	}
	for _, args := range [][]string{
		{"fix-skill", "--global", "--deploy"},
		{"fix-bin", "fix-skill", "--global", "--deploy"},
		{"fix-skill", "--target", "claude", "--global", "--deploy"},
	} {
		if _, _, err := runInstall(t, args...); err == nil {
			t.Fatalf("ambiguous or foreign runtime selection admitted: %v", args)
		}
	}
	if !bytes.Equal(mustRead(t, lp), priorLock) {
		t.Fatal("negative controls altered Codex lock")
	}
}
