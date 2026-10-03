package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/state"
)

func dp06Fixture(t *testing.T) (string, string) {
	t.Helper()
	root := fixtureCatalog(t)
	home := withRemoteEnv(t, &servingFetcher{bodies: map[string][]byte{}})
	t.Chdir(root)
	dp01WriteIsolationSkill(t, root)
	return root, home
}

func TestPiLifecycleUpdateScopeAdmission(t *testing.T) {
	for _, flags := range [][]string{nil, {"--local", "--global"}} {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			root, home := dp06Fixture(t)
			if _, _, err := runInstall(t, "sample-isolation", "--target", "pi", "--local", "--deploy"); err != nil {
				t.Fatal(err)
			}
			before := dp01SnapshotFiles(t, root)
			beforeHome := dp01SnapshotFiles(t, home)
			args := append([]string{"sample-isolation", "--target", "pi", "--deploy"}, flags...)
			_, _, err := runUpdate(t, args...)
			if err == nil || !strings.Contains(err.Error(), "scope") {
				t.Fatalf("want scope error, got %v", err)
			}
			dp06SameSnapshot(t, root, before)
			dp06SameSnapshot(t, home, beforeHome)
		})
	}
}

func dp06SameSnapshot(t *testing.T, root string, before map[string]string) {
	t.Helper()
	after := dp01SnapshotFiles(t, root)
	if len(after) != len(before) {
		t.Fatalf("file set changed in %s", root)
	}
	for p, b := range before {
		if after[p] != b {
			t.Fatalf("changed %s", filepath.Join(root, p))
		}
	}
}

func dp06Write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func dp06UpdateArgs(tool, name string) []string {
	args := []string{name, "--deploy"}
	if tool == "pi" {
		args = append(args, "--global")
	}
	return args
}

// dp06Dependencies supplies invented global script and directory declarations;
// ordinary CLI fixtures never fetch or execute upstream packages.
func dp06Dependencies(t *testing.T) directoryFixture {
	t.Helper()
	f := newDirectoryFixture(t)
	raw := &servingFetcher{bodies: map[string][]byte{fixRawURL: fixRawBinary}}
	old := fetcherForDeploy
	fetcherForDeploy = raw
	t.Cleanup(func() { fetcherForDeploy = old })
	rec := f.recipe(t, "fixture-tree", "1.0.0", "inert tree v1")
	rec.Role = manifest.RoleTools
	f.saveRecipe(t, rec)
	dp06Setting(t, f.root, "fixture-feature", "1.0.0", "[fix-bin, fixture-tree]")
	dp06Setting(t, f.root, "fixture-optional", "1.0.0", "[]")
	dp06Write(t, filepath.Join(f.root, "profiles/fixture-pi.yaml"), "apiVersion: patronus/v2\nfamily: profile\nname: fixture-pi\nversion: 1.0.0\nrole: lifecycle\ndescription: Invented lifecycle\nlayers:\n  capabilities: [fixture-feature@pi, fixture-optional@pi]\n")
	return f
}
func dp06Setting(t *testing.T, root, name, version, deps string) {
	t.Helper()
	dp06Write(t, filepath.Join(root, "artifacts", name, "patronus.yaml"), fmt.Sprintf("apiVersion: patronus/v2\nfamily: artifact\nname: %s\nversion: %s\nrole: capability\ndescription: Invented lifecycle\ntype: setting\ntargets: [pi]\nrequires: %s\nsetting:\n  path: %s\n  value: version-%s\n", name, version, deps, name, version))
}
func dp06InstallProfile(t *testing.T, scope string) {
	t.Helper()
	if _, _, err := runInstall(t, "--profile", "fixture-pi", "--target", "pi", "--"+scope, "--deploy"); err != nil {
		t.Fatal(err)
	}
}
func dp06OwnedVersion(t *testing.T, home, root, scope, name, tool, want string) {
	t.Helper()
	st, err := state.Load(removeStatePath(scope, home, root))
	if err != nil {
		t.Fatal(err)
	}
	rows := st.Find(name, tool, scope)
	if len(rows) != 1 || rows[0].ItemVersion != want {
		t.Fatalf("%s/%s version want %s: %+v", scope, name, want, rows)
	}
}

func TestPiLifecycleScopedUpdates(t *testing.T) {
	for _, scope := range []string{"local", "global"} {
		t.Run(scope, func(t *testing.T) {
			f := dp06Dependencies(t)
			dp06InstallProfile(t, "global")
			dp06InstallProfile(t, "local")
			if _, _, err := runLock(t, "--profile", "fixture-pi", "--target", "pi"); err != nil {
				t.Fatal(err)
			}
			dp06Write(t, filepath.Join(f.home, "patronus.lock"), "untouched global lock sentinel")
			dp06Setting(t, f.root, "fixture-feature", "2.0.0", "[fix-bin, fixture-tree]")
			if scope == "global" {
				rec := f.recipe(t, "fixture-tree", "2.0.0", "inert tree v2")
				rec.Role = manifest.RoleTools
				f.saveRecipe(t, rec)
			}
			preservedRoot := f.home
			if scope == "global" {
				preservedRoot = f.root
			}
			before := dp01SnapshotFiles(t, preservedRoot)
			lockBefore := mustRead(t, filepath.Join(f.root, "patronus.lock"))
			out, _, err := runUpdate(t, "fixture-pi", "--target", "pi", "--"+scope, "--deploy")
			if err != nil {
				t.Fatal(err)
			}
			dp06SameSnapshot(t, preservedRoot, before)
			if !bytes.Equal(lockBefore, mustRead(t, filepath.Join(f.root, "patronus.lock"))) {
				t.Fatal("update changed desired lock")
			}
			dp06OwnedVersion(t, f.home, f.root, scope, "fixture-feature", "pi", "2.0.0")
			if !strings.Contains(out, "runtime-unverified") || !strings.Contains(out, "reload") {
				t.Fatalf("missing placement/reload boundary: %s", out)
			}
			if scope == "global" {
				dp06OwnedVersion(t, f.home, f.root, "global", "fixture-tree", recipe.TargetAgnostic, "2.0.0")
			}
			release, err := packagestate.Acquire(f.home)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = runUpdate(t, "fixture-pi", "--target", "pi", "--"+scope, "--deploy")
			closeErr := release()
			if closeErr != nil {
				t.Fatal(closeErr)
			}
			if err == nil || !strings.Contains(err.Error(), "busy") {
				t.Fatalf("host lock bypass: %v", err)
			}
		})
	}
}

func TestPiLifecycleLocalPrerequisiteRefusal(t *testing.T) {
	for _, kind := range []string{"missing", "stale-state", "unowned-equal", "directory-drift", "directory-pin"} {
		t.Run(kind, func(t *testing.T) {
			f := dp06Dependencies(t)
			if kind != "missing" {
				dp06InstallProfile(t, "global")
			}
			switch kind {
			case "stale-state":
				path := removeStatePath("global", f.home, f.root)
				st, err := state.Load(path)
				if err != nil {
					t.Fatal(err)
				}
				for i := range st.Items {
					if st.Items[i].Artifact == "fix-bin" {
						st.Items[i].ItemVersion = "stale"
					}
				}
				if err := state.Save(path, st); err != nil {
					t.Fatal(err)
				}
			case "unowned-equal":
				path := removeStatePath("global", f.home, f.root)
				st, err := state.Load(path)
				if err != nil {
					t.Fatal(err)
				}
				var keep []state.Item
				for _, it := range st.Items {
					if it.Artifact != "fix-bin" {
						keep = append(keep, it)
					}
				}
				st.Items = keep
				if err := state.Save(path, st); err != nil {
					t.Fatal(err)
				}
			case "directory-drift":
				dp06Write(t, f.readme("fixture-tree"), "external edit")
			case "directory-pin":
				f.recipe(t, "fixture-tree", "2.0.0", "new pin")
			}
			// Normalize only the host coordination file, which D-09 intentionally creates.
			release, err := packagestate.Acquire(f.home)
			if err != nil {
				t.Fatal(err)
			}
			if err := release(); err != nil {
				t.Fatal(err)
			}
			before := dp01SnapshotFiles(t, f.root)
			beforeHome := dp01SnapshotFiles(t, f.home)
			_, _, err = runInstall(t, "--profile", "fixture-pi", "--target", "pi", "--local", "--deploy")
			if err == nil || !strings.Contains(err.Error(), "separate global") {
				t.Fatalf("want prerequisite refusal: %v", err)
			}
			dp06SameSnapshot(t, f.root, before)
			dp06SameSnapshot(t, f.home, beforeHome)
		})
	}
}

func TestPiLifecycleAcquisitionBarrier(t *testing.T) {
	for _, kind := range []string{"missing-script", "corrupt-directory", "missing-member"} {
		t.Run(kind, func(t *testing.T) {
			f := dp06Dependencies(t)
			names := []string{"fixture-feature"}
			switch kind {
			case "missing-script":
				fetcherForDeploy = &servingFetcher{bodies: map[string][]byte{}}
			case "corrupt-directory":
				for u := range f.fetcher.bodies {
					f.fetcher.bodies[u] = []byte("corrupt bytes")
				}
			case "missing-member":
				data := mustTarGz(t, map[string][]byte{"other": []byte("not requested")})
				fetcherForDeploy = &servingFetcher{bodies: map[string][]byte{fixRawURL: fixRawBinary, fixArchiveURL: data}}
				path := filepath.Join(f.root, "recipes/fix-archive-bin.yaml")
				text := string(mustRead(t, path))
				text = strings.ReplaceAll(text, shaHex(fixArchiveTarGz(t)), shaHex(data))
				dp06Write(t, path, text)
				names = append(names, "fix-archive-bin")
			}
			release, err := packagestate.Acquire(f.home)
			if err != nil {
				t.Fatal(err)
			}
			if err := release(); err != nil {
				t.Fatal(err)
			}
			before := dp01SnapshotFiles(t, f.root)
			beforeHome := dp01SnapshotFiles(t, f.home)
			args := append(names, "--target", "pi", "--global", "--deploy")
			_, _, err = runInstall(t, args...)
			if err == nil || !strings.Contains(err.Error(), "acquisition") {
				t.Fatalf("want acquisition refusal: %v", err)
			}
			dp06SameSnapshot(t, f.root, before)
			dp06SameSnapshot(t, f.home, beforeHome)
		})
	}
}

func TestPiProfileUpdateDoesNotResurrectAndRefusesAbsentDependency(t *testing.T) {
	f := dp06Dependencies(t)
	dp06InstallProfile(t, "global")
	if _, _, err := runLock(t, "--profile", "fixture-pi", "--target", "pi"); err != nil {
		t.Fatal(err)
	}
	locked := mustRead(t, filepath.Join(f.root, "patronus.lock"))
	if _, _, err := execRemove(t, "fixture-optional", "--target", "pi", "--global", "--deploy", "--force"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(locked, mustRead(t, filepath.Join(f.root, "patronus.lock"))) {
		t.Fatal("remove changed desired pins")
	}
	dp06Setting(t, f.root, "fixture-feature", "2.0.0", "[fix-bin, fixture-tree]")
	out, _, err := runUpdate(t, "fixture-pi", "--global", "--deploy")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "absent pinned member") {
		t.Fatal(out)
	}
	st, err := state.Load(removeStatePath("global", f.home, f.root))
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Find("fixture-optional", "pi", "global")) != 0 {
		t.Fatal("removed optional resurrected")
	}
	dp06Setting(t, f.root, "fixture-feature", "3.0.0", "[fix-bin, fixture-tree, fixture-optional]")
	before := dp01SnapshotFiles(t, f.home)
	_, _, err = runUpdate(t, "fixture-pi", "--target", "pi", "--global", "--deploy")
	if err == nil || !strings.Contains(err.Error(), "dependency-incomplete") {
		t.Fatalf("absent prerequisite accepted: %v", err)
	}
	dp06SameSnapshot(t, f.home, before)
}

func TestPiLockTargetProvenance(t *testing.T) {
	f := dp06Dependencies(t)
	path := filepath.Join(f.root, "patronus.lock")
	if _, _, err := runLock(t, "--profile", "fixture-pi", "--target", "pi"); err != nil {
		t.Fatal(err)
	}
	before := mustRead(t, path)
	for _, target := range []string{"claude", "codex", "opencode", "all"} {
		_, _, err := runInstall(t, "--profile", "fixture-pi", "--target", target, "--global", "--deploy")
		if err == nil || !strings.Contains(err.Error(), "target") {
			t.Fatalf("wrong target %s: %v", target, err)
		}
		if !bytes.Equal(before, mustRead(t, path)) {
			t.Fatal("wrong target changed lock")
		}
	}
	// A loaded v3 profile itself establishes the target; no all-flavor re-expansion.
	if _, _, err := runInstall(t, "--profile", "fixture-pi", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
	dp06OwnedVersion(t, f.home, f.root, "global", "fixture-feature", "pi", "1.0.0")
	l, err := lock.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	l.Version = 2
	l.Target = ""
	if err := lock.Save(path, l); err != nil {
		t.Fatal(err)
	}
	legacy := mustRead(t, path)
	// Consistent installed Pi provenance admits old desired pins without conversion.
	if _, _, err := runInstall(t, "--profile", "fixture-pi", "--target", "pi", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(legacy, mustRead(t, path)) {
		t.Fatal("silently converted legacy lock")
	}
	// No installed local provenance: preview closure difference then require deliberate regeneration.
	out, _, err := runInstall(t, "--profile", "fixture-pi", "--target", "pi", "--local", "--deploy")
	if err == nil || !strings.Contains(err.Error(), "regenerate") || !strings.Contains(out, "closure preview") {
		t.Fatalf("old lock not gated: %s %v", out, err)
	}
	if !bytes.Equal(legacy, mustRead(t, path)) {
		t.Fatal("refusal changed old lock")
	}
	if _, _, err := runLock(t, "--profile", "fixture-pi", "--target", "pi"); err != nil {
		t.Fatal(err)
	}
	dp06InstallProfile(t, "local")
}

func TestPiLifecycleRemovalScopesAndRecordedRoots(t *testing.T) {
	f := dp06Dependencies(t)
	dp06InstallProfile(t, "global")
	dp06InstallProfile(t, "local")
	homeBefore := dp01SnapshotFiles(t, f.home)
	if _, _, err := execRemove(t, "fixture-feature", "--target", "pi", "--local", "--deploy", "--force"); err != nil {
		t.Fatal(err)
	}
	dp06SameSnapshot(t, f.home, homeBefore)
	localBefore := dp01SnapshotFiles(t, f.root)
	if _, _, err := execRemove(t, "fix-bin", "fixture-tree", "--target", "pi", "--global", "--deploy", "--force"); err != nil {
		t.Fatal(err)
	}
	dp06SameSnapshot(t, f.root, localBefore)
	for _, p := range []string{filepath.Join(f.home, ".patronus/bin/fix-bin"), f.readme("fixture-tree")} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("global agnostic removal left %s: %v", p, err)
		}
	}
	oldRoot := os.Getenv("PI_CODING_AGENT_DIR")
	sentinel := t.TempDir()
	dp06Write(t, filepath.Join(sentinel, "settings.json"), `{"do-not-touch":true}`)
	t.Setenv("PI_CODING_AGENT_DIR", sentinel)
	before := dp01SnapshotFiles(t, sentinel)
	if _, _, err := execRemove(t, "fixture-optional", "--target", "pi", "--global", "--deploy", "--force"); err == nil {
		t.Fatal("alternate root selected old ownership")
	}
	dp06SameSnapshot(t, sentinel, before)
	if !strings.Contains(string(mustRead(t, filepath.Join(oldRoot, "settings.json"))), "fixture-optional") {
		t.Fatal("alternate root removed original effect")
	}
	t.Setenv("PI_CODING_AGENT_DIR", oldRoot)
	if _, _, err := execRemove(t, "fixture-optional", "--target", "pi", "--global", "--deploy", "--force"); err != nil {
		t.Fatal(err)
	}
}

func TestPiLifecycleLegacyTargetsAndScopeRefusal(t *testing.T) {
	for _, tool := range []string{"claude", "codex", "opencode"} {
		t.Run(tool, func(t *testing.T) {
			root, home := dp06Fixture(t)
			path := filepath.Join(root, "artifacts/fixture-legacy/patronus.yaml")
			definition := func(version string) string {
				return fmt.Sprintf("apiVersion: patronus/v2\nfamily: artifact\nname: fixture-legacy\nversion: %s\nrole: capability\ndescription: Invented legacy\ntype: skill\nentry: SKILL.md\ntargets: [%s]\n", version, tool)
			}
			dp06Write(t, path, definition("1.0.0"))
			dp06Write(t, filepath.Join(filepath.Dir(path), "SKILL.md"), "# Invented legacy\n")
			for _, scope := range []string{"global", "local"} {
				if _, _, err := runInstall(t, "fixture-legacy", "--target", tool, "--"+scope, "--deploy"); err != nil {
					t.Fatal(err)
				}
			}
			before := dp01SnapshotFiles(t, home)
			_, _, err := runUpdate(t, "fixture-legacy", "--local", "--deploy")
			if err == nil || !strings.Contains(err.Error(), "scope flags") {
				t.Fatalf("non-Pi scope accepted: %v", err)
			}
			dp06SameSnapshot(t, home, before)
			dp06Write(t, path, definition("2.0.0"))
			dp06Write(t, filepath.Join(filepath.Dir(path), "SKILL.md"), "# Invented legacy v2\n")
			if _, _, err := runUpdate(t, "fixture-legacy", "--deploy"); err != nil {
				t.Fatal(err)
			}
			for _, scope := range []string{"global", "local"} {
				dp06OwnedVersion(t, home, root, scope, "fixture-legacy", tool, "2.0.0")
			}
			if _, _, err := execRemove(t, "fixture-legacy", "--deploy"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPiLifecycleImplicitScopeAndTargetAllRefuse(t *testing.T) {
	root, home := dp06Fixture(t)
	if _, _, err := runInstall(t, "sample-isolation", "--target", "pi", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
	before := dp01SnapshotFiles(t, home)
	project := dp01SnapshotFiles(t, root)
	for _, args := range [][]string{{"sample-isolation", "--deploy"}, {"sample-isolation", "--local", "--global", "--deploy"}, {"--all", "--target", "all", "--deploy"}} {
		if _, _, err := runUpdate(t, args...); err == nil {
			t.Fatalf("accepted ambiguous Pi update %v", args)
		}
		dp06SameSnapshot(t, home, before)
		dp06SameSnapshot(t, root, project)
	}
}

func TestPiLifecycleUpdateMissingGlobalRefuses(t *testing.T) {
	for _, condition := range []string{"missing", "stale", "unowned-equal"} {
		t.Run(condition, func(t *testing.T) {
			f := dp06Dependencies(t)
			dp06InstallProfile(t, "global")
			dp06InstallProfile(t, "local")
			dp06Setting(t, f.root, "fixture-feature", "2.0.0", "[fix-bin, fixture-tree]")
			path := removeStatePath("global", f.home, f.root)
			st, err := state.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			var items []state.Item
			for _, it := range st.Items {
				if it.Artifact == "fix-bin" {
					if condition != "stale" {
						continue
					}
					it.ItemVersion = "stale"
				}
				items = append(items, it)
			}
			st.Items = items
			if err := state.Save(path, st); err != nil {
				t.Fatal(err)
			}
			if condition == "missing" {
				if err := os.Remove(filepath.Join(f.home, ".patronus/bin/fix-bin")); err != nil {
					t.Fatal(err)
				}
			}
			before := dp01SnapshotFiles(t, f.root)
			homeBefore := dp01SnapshotFiles(t, f.home)
			_, _, err = runUpdate(t, "fixture-pi", "--target", "pi", "--local", "--deploy")
			if err == nil || !strings.Contains(err.Error(), "separate global") {
				t.Fatalf("invalid prerequisite admitted: %v", err)
			}
			dp06SameSnapshot(t, f.root, before)
			dp06SameSnapshot(t, f.home, homeBefore)
		})
	}
}

func TestPiLifecycleDryPreviewNeverAcquires(t *testing.T) {
	f := dp06Dependencies(t)
	// Neither recipe payload exists in its acquisition seam. Preview still succeeds
	// because it validates declarations/pins, not online availability.
	fetcherForDeploy = &servingFetcher{bodies: map[string][]byte{}}
	f.fetcher.bodies = map[string][]byte{}
	if _, _, err := runInstall(t, "--profile", "fixture-pi", "--target", "pi", "--global"); err != nil {
		t.Fatal(err)
	}
	if f.fetcher.calls != 0 {
		t.Fatal("preview acquired directory bytes")
	}
	// A remote cold-cache Pi preview must not even bootstrap the registry.
	remote := &servingFetcher{bodies: map[string][]byte{}}
	home := withRemoteEnv(t, remote)
	denied := &dp06DenyFetcher{}
	registryFetcher = denied
	fetcherForCommands = denied
	_, _, err := runInstall(t, "sample-isolation", "--target", "pi", "--global")
	if err == nil || !strings.Contains(err.Error(), "network-free") {
		t.Fatalf("cold remote preview: %v", err)
	}
	if denied.calls != 0 {
		t.Fatalf("remote preview attempted acquisition: %v", denied.calls)
	}
	if len(dp01SnapshotFiles(t, home)) != 0 {
		t.Fatal("dry preview wrote cache or state")
	}
}

type dp06DenyFetcher struct{ calls int }

func (f *dp06DenyFetcher) Fetch(context.Context, string) (io.ReadCloser, error) {
	f.calls++
	return nil, fmt.Errorf("fixture denies acquisition")
}

func TestPiLifecycleDefaultScope(t *testing.T) {
	for _, condition := range []string{"owned", "missing", "stale"} {
		t.Run("local-"+condition, func(t *testing.T) {
			f := dp06Dependencies(t)
			if condition != "missing" {
				dp06InstallProfile(t, "global")
			}
			if condition == "stale" {
				f.recipe(t, "fixture-tree", "2.0.0", "new declaration")
			}
			release, err := packagestate.Acquire(f.home)
			if err != nil {
				t.Fatal(err)
			}
			if err := release(); err != nil {
				t.Fatal(err)
			}
			before := dp01SnapshotFiles(t, f.home)
			localBefore := dp01SnapshotFiles(t, f.root)
			out, _, err := runInstall(t, "--profile", "fixture-pi", "--target", "pi", "--deploy")
			if condition == "owned" {
				if err != nil {
					t.Fatal(err)
				}
				dp06OwnedVersion(t, f.home, f.root, "local", "fixture-feature", "pi", "1.0.0")
				if !strings.Contains(out, "default scope: local") {
					t.Fatal(out)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), "separate global") {
					t.Fatalf("default local wrote globals: %v", err)
				}
				dp06SameSnapshot(t, f.root, localBefore)
			}
			dp06SameSnapshot(t, f.home, before)
		})
	}
	t.Run("mixed-defaults", func(t *testing.T) {
		f := dp06Dependencies(t)
		p := filepath.Join(f.root, "artifacts/fixture-optional/patronus.yaml")
		dp06Write(t, p, string(mustRead(t, p))+"defaults:\n  scope: global\n")
		_, _, err := runInstall(t, "--profile", "fixture-pi", "--target", "pi", "--deploy")
		if err == nil || !strings.Contains(err.Error(), "mixed local/global") {
			t.Fatalf("mixed defaults admitted: %v", err)
		}
		requireNoPackageWrites(t, f.home)
	})
	t.Run("global-defaults-and-delivery-only", func(t *testing.T) {
		f := dp06Dependencies(t)
		p := filepath.Join(f.root, "artifacts/fixture-feature/patronus.yaml")
		dp06Write(t, p, string(mustRead(t, p))+"defaults:\n  scope: global\n")
		if _, _, err := runInstall(t, "fixture-feature", "--target", "pi", "--deploy"); err != nil {
			t.Fatal(err)
		}
		dp06OwnedVersion(t, f.home, f.root, "global", "fixture-feature", "pi", "1.0.0")
		if _, _, err := runInstall(t, "fixture-tree", "--target", "pi", "--deploy"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPiLifecycleWholeUpdatePreflight(t *testing.T) {
	f := dp06Dependencies(t)
	dp06InstallProfile(t, "global")
	dp06Setting(t, f.root, "fixture-feature", "2.0.0", "[fix-bin, fixture-tree]")
	dp06Setting(t, f.root, "fixture-optional", "2.0.0", "[]")
	path := filepath.Join(f.home, ".pi/agent/settings.json")
	body := string(mustRead(t, path))
	body = strings.Replace(body, `"fixture-optional": "version-1.0.0"`, `"fixture-optional": "external edit"`, 1)
	dp06Write(t, path, body)
	before := dp01SnapshotFiles(t, f.home)
	project := dp01SnapshotFiles(t, f.root)
	_, _, err := runUpdate(t, "fixture-pi", "--target", "pi", "--global", "--deploy")
	if err == nil {
		t.Fatal("dirty second member admitted whole update")
	}
	dp06SameSnapshot(t, f.home, before)
	dp06SameSnapshot(t, f.root, project)
}

func TestPiLifecycleMissingRequiredProfileRow(t *testing.T) {
	f := dp06Dependencies(t)
	p := filepath.Join(f.root, "profiles/fixture-pi.yaml")
	body := string(mustRead(t, p))
	body = strings.Replace(body, "fixture-optional@pi", "absent-required@pi", 1)
	dp06Write(t, p, body)
	_, _, err := runInstall(t, "--profile", "fixture-pi", "--target", "pi", "--global", "--deploy")
	if err == nil || !strings.Contains(err.Error(), "required profile member") {
		t.Fatalf("required member skipped: %v", err)
	}
	requireNoPackageWrites(t, f.home)
}

func TestPiLifecycleUpdateAcquisitionBarrier(t *testing.T) {
	f := dp06Dependencies(t)
	dp06InstallProfile(t, "global")
	dp06Setting(t, f.root, "fixture-feature", "2.0.0", "[fix-bin, fixture-tree]")
	rec := f.recipe(t, "fixture-tree", "2.0.0", "new inert payload")
	rec.Role = manifest.RoleTools
	f.saveRecipe(t, rec)
	f.fetcher.bodies = map[string][]byte{}
	before := dp01SnapshotFiles(t, f.home)
	project := dp01SnapshotFiles(t, f.root)
	_, _, err := runUpdate(t, "fixture-pi", "--target", "pi", "--global", "--deploy")
	if err == nil || !strings.Contains(err.Error(), "acquisition") {
		t.Fatalf("bad update acquisition: %v", err)
	}
	dp06SameSnapshot(t, f.home, before)
	dp06SameSnapshot(t, f.root, project)
}

func TestPiLifecycleUpdateRefusesImplicitRelocation(t *testing.T) {
	root, home := dp06Fixture(t)
	if _, _, err := runInstall(t, "sample-isolation", "--target", "pi", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "artifacts/sample-isolation/patronus.yaml")
	dp06Write(t, source, strings.Replace(string(mustRead(t, source)), "version: 1.0.0", "version: 2.0.0", 1))
	destination := t.TempDir()
	t.Setenv("PI_CODING_AGENT_DIR", destination)
	before := dp01SnapshotFiles(t, home)
	_, _, err := runUpdate(t, "sample-isolation", "--global", "--deploy")
	if err == nil || !strings.Contains(err.Error(), "relocate") {
		t.Fatalf("silent relocation: %v", err)
	}
	dp06SameSnapshot(t, home, before)
	if len(dp01SnapshotFiles(t, destination)) != 0 {
		t.Fatal("relocation wrote destination")
	}
}

func TestPiLifecycleScanRetainsTargetWithoutRuntimeProbe(t *testing.T) {
	f := dp06Dependencies(t)
	path := filepath.Join(f.root, "patronus.lock")
	if err := lock.Save(path, &lock.Lock{Version: 3, Target: "pi", Profile: "fixture-pi", Entries: []lock.Entry{{Name: "fixture-executable", Kind: "plugin", Source: "registry", Version: "1.0.0", Status: lock.StatusUnverified}}}); err != nil {
		t.Fatal(err)
	}
	before := mustRead(t, path)
	probe := &dp06Lister{}
	if err := reconcilePluginLock(context.Background(), f.root, detectedInv(f.home, "claude", "pi"), probe, func(string, ...any) {}); err != nil {
		t.Fatal(err)
	}
	if probe.calls != 0 || !bytes.Equal(before, mustRead(t, path)) {
		t.Fatal("Pi scan probed runtime or converted desired lock")
	}
}

type dp06Lister struct{ calls int }

func (f *dp06Lister) List(context.Context, string) ([]byte, bool) { f.calls++; return nil, false }

func TestPiLifecycleLocalArchivePinEvidenceRefuses(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(fmt.Sprintf("changed-pin=%t", changed), func(t *testing.T) {
			f := dp06Dependencies(t)
			fetcherForDeploy = &servingFetcher{bodies: map[string][]byte{fixArchiveURL: fixArchiveTarGz(t)}}
			if _, _, err := runInstall(t, "fix-archive-bin", "--target", "pi", "--global", "--deploy"); err != nil {
				t.Fatal(err)
			}
			if changed {
				// Same delivered member, different same-version archive pin: installed
				// member ownership cannot establish the selected archive provenance.
				data := mustTarGz(t, map[string][]byte{"fix-archive-bin": fixArchivedBinary, "extra": []byte("new metadata")})
				path := filepath.Join(f.root, "recipes/fix-archive-bin.yaml")
				dp06Write(t, path, strings.ReplaceAll(string(mustRead(t, path)), shaHex(fixArchiveTarGz(t)), shaHex(data)))
			}
			dp06Setting(t, f.root, "fixture-feature", "1.0.0", "[fix-archive-bin]")
			before := dp01SnapshotFiles(t, f.home)
			localBefore := dp01SnapshotFiles(t, f.root)
			_, _, err := runInstall(t, "fixture-feature", "--target", "pi", "--local", "--deploy")
			if err == nil || !strings.Contains(err.Error(), "not the selected archive pin") {
				t.Fatalf("unproved archive prerequisite admitted: %v", err)
			}
			dp06SameSnapshot(t, f.home, before)
			dp06SameSnapshot(t, f.root, localBefore)
		})
	}
}

func TestPiLifecycleLocalRechecksGlobalBeforeFirstWrite(t *testing.T) {
	f := dp06Dependencies(t)
	dp06InstallProfile(t, "global")
	cmd := newInstallCmd()
	p, err := planInstall(cmd, installPlanRequest{Names: []string{"fixture-feature"}, Tool: "pi", Scope: "local", Home: f.home, ProjectDir: f.root})
	if err != nil {
		t.Fatal(err)
	}
	dp06Write(t, filepath.Join(f.home, ".patronus/bin/fix-bin"), "external edit after verified planning")
	before := dp01SnapshotFiles(t, f.home)
	localBefore := dp01SnapshotFiles(t, f.root)
	err = runDeployWith(cmd, p.Changes, p.Resolver, deployOptions{target: "pi", globalPrerequisites: p.GlobalPrerequisites, home: f.home, projectDir: f.root}, &fakeRunner{})
	if err == nil || !strings.Contains(err.Error(), "fresh preview") {
		t.Fatalf("stale global admitted: %v", err)
	}
	dp06SameSnapshot(t, f.home, before)
	dp06SameSnapshot(t, f.root, localBefore)
}
