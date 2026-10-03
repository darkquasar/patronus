package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/recipe"
)

func TestPiLifecycleArchiveUpdateRevalidatesSelectedPin(t *testing.T) {
	for _, outcome := range []string{"changed-member", "same-version-pin", "unavailable", "corrupt"} {
		t.Run(outcome, func(t *testing.T) {
			f := dp06Dependencies(t)
			fetch := &servingFetcher{bodies: map[string][]byte{fixArchiveURL: fixArchiveTarGz(t)}}
			fetcherForDeploy = fetch
			if _, _, err := runInstall(t, "fix-archive-bin", "fixture-optional", "--target", "pi", "--global", "--deploy"); err != nil {
				t.Fatal(err)
			}
			member := []byte("invented replacement member\n")
			archive := mustTarGz(t, map[string][]byte{"fix-archive-bin": member})
			path := filepath.Join(f.root, "recipes/fix-archive-bin.yaml")
			manifest := strings.ReplaceAll(string(mustRead(t, path)), shaHex(fixArchiveTarGz(t)), shaHex(archive))
			version := "2.0.0"
			if outcome == "same-version-pin" {
				version = "1.0.0"
			}
			dp06Write(t, path, strings.ReplaceAll(manifest, "version: 1.0.0", "version: "+version))
			if outcome != "same-version-pin" {
				dp06Setting(t, f.root, "fixture-optional", "2.0.0", "[]")
			}
			fetch.bodies[fixArchiveURL] = archive
			switch outcome {
			case "unavailable":
				delete(fetch.bodies, fixArchiveURL)
			case "corrupt":
				fetch.bodies[fixArchiveURL] = []byte("corrupt archive")
			}
			before, localBefore := dp01SnapshotFiles(t, f.home), dp01SnapshotFiles(t, f.root)
			_, _, err := runUpdate(t, "fix-archive-bin", "fixture-optional", "--target", "pi", "--global", "--deploy")
			if outcome == "unavailable" || outcome == "corrupt" {
				if err == nil {
					t.Fatal("invalid selected archive admitted without acquisition")
				}
				dp06SameSnapshot(t, f.home, before)
				dp06SameSnapshot(t, f.root, localBefore)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(mustRead(t, filepath.Join(f.home, ".patronus/bin/fix-archive-bin")), member) {
				t.Fatal("selected archive member not delivered")
			}
			dp06OwnedVersion(t, f.home, f.root, "global", "fix-archive-bin", recipe.TargetAgnostic, version)
		})
	}
}

func TestPiLifecycleGlobalFetchDriftRefusesWholeSelection(t *testing.T) {
	for _, operation := range []string{"install", "update", "force-update", "clean-update"} {
		t.Run(operation, func(t *testing.T) {
			f := dp06Dependencies(t)
			if _, _, err := runInstall(t, "fix-bin", "fixture-optional", "--target", "pi", "--global", "--deploy"); err != nil {
				t.Fatal(err)
			}
			if operation != "clean-update" {
				dp06Write(t, filepath.Join(f.home, ".patronus/bin/fix-bin"), "external edited bytes")
			}
			newBytes := []byte("invented raw v2\n")
			path := filepath.Join(f.root, "recipes/fix-bin.yaml")
			manifest := strings.ReplaceAll(string(mustRead(t, path)), shaHex(fixRawBinary), shaHex(newBytes))
			dp06Write(t, path, strings.ReplaceAll(manifest, "version: 1.0.0", "version: 2.0.0"))
			dp06Setting(t, f.root, "fixture-optional", "2.0.0", "[]")
			fetcherForDeploy = &servingFetcher{bodies: map[string][]byte{fixRawURL: newBytes}}
			before, localBefore := dp01SnapshotFiles(t, f.home), dp01SnapshotFiles(t, f.root)
			args := []string{"fix-bin", "fixture-optional", "--target", "pi", "--global", "--deploy"}
			if operation == "force-update" {
				args = append(args, "--force")
			}
			var err error
			if operation == "install" {
				_, _, err = runInstall(t, args...)
			} else {
				_, _, err = runUpdate(t, args...)
			}
			if operation == "clean-update" {
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(mustRead(t, filepath.Join(f.home, ".patronus/bin/fix-bin")), newBytes) {
					t.Fatal("unchanged owned raw delivery did not update")
				}
				dp06OwnedVersion(t, f.home, f.root, "global", "fix-bin", recipe.TargetAgnostic, "2.0.0")
				return
			}
			if err == nil || !strings.Contains(err.Error(), "drift") {
				t.Fatalf("edited global dependency admitted: %v", err)
			}
			dp06SameSnapshot(t, f.home, before)
			dp06SameSnapshot(t, f.root, localBefore)
		})
	}
}

func TestPiLifecycleDirectoryRemovalConflictRefusesWholeSelection(t *testing.T) {
	for _, kind := range []string{"owned-drift", "unknown-leftover"} {
		t.Run(kind, func(t *testing.T) {
			f := dp06Dependencies(t)
			dp06InstallProfile(t, "global")
			path := f.readme("fixture-tree")
			if kind == "unknown-leftover" {
				path = filepath.Join(filepath.Dir(path), "user-extra.txt")
			}
			dp06Write(t, path, "external bytes to preserve")
			before, localBefore := dp01SnapshotFiles(t, f.home), dp01SnapshotFiles(t, f.root)
			_, _, err := execRemove(t, "fixture-tree", "fixture-optional", "--target", "pi", "--global", "--deploy")
			if err == nil || !strings.Contains(err.Error(), "conflict") {
				t.Fatalf("directory conflict did not block whole selection: %v", err)
			}
			dp06SameSnapshot(t, f.home, before)
			dp06SameSnapshot(t, f.root, localBefore)
		})
	}
}

func TestPiLockRegenerationRetainsProvenTarget(t *testing.T) {
	for _, target := range []string{"", "all", "claude", "codex", "opencode"} {
		t.Run("target="+target, func(t *testing.T) {
			f := dp06Dependencies(t)
			if _, _, err := runLock(t, "--profile", "fixture-pi", "--target", "pi"); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(f.root, "patronus.lock")
			before := mustRead(t, path)
			args := []string{"--profile", "fixture-pi"}
			if target != "" {
				args = append(args, "--target", target)
			}
			out, _, err := runLock(t, args...)
			if target != "" {
				if err == nil || !bytes.Equal(before, mustRead(t, path)) {
					t.Fatalf("wrong-target lock overwrite allowed: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			l, err := lock.Load(path)
			if err != nil || l.Version != 3 || l.Target != "pi" || !strings.Contains(out, "closure preview") {
				t.Fatalf("proven Pi target lost: %+v %v output=%s", l, err, out)
			}
			found := false
			for _, e := range l.Entries {
				found = found || e.Name == "fixture-feature"
			}
			if !found {
				t.Fatal("Pi flavored member silently dropped")
			}
		})
	}
}

func TestPiLifecycleDefaultScopeJSONIsMachineReadable(t *testing.T) {
	dp06Fixture(t)
	old := jsonOutput
	jsonOutput = true
	t.Cleanup(func() { jsonOutput = old })
	out, _, err := runInstall(t, "sample-isolation", "--target", "pi")
	if err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid advertised JSON: %v; output=%s", err, out)
	}
}
