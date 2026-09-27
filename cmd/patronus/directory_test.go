package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagedelivery"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type directoryFixtureFetcher struct {
	bodies map[string][]byte
	calls  int
}

func (f *directoryFixtureFetcher) Open(_ context.Context, url string) (io.ReadCloser, error) {
	f.calls++
	b, ok := f.bodies[url]
	if !ok {
		return nil, fmt.Errorf("unserved directory URL %s", url)
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

type directoryFixture struct {
	home, root string
	fetcher    *directoryFixtureFetcher
}

func newDirectoryFixture(t *testing.T) directoryFixture {
	t.Helper()
	root := fixtureCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(home, ".config", "opencode"))
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Chdir(root)
	f := &directoryFixtureFetcher{bodies: map[string][]byte{}}
	oldFetch, oldLook, oldRunner := directoryFetcherForDeploy, directoryLookPath, runnerForCommands
	directoryFetcherForDeploy = f
	directoryLookPath = func(string) (string, error) { return "", os.ErrNotExist }
	runnerForCommands = &fakeRunner{}
	t.Cleanup(func() { directoryFetcherForDeploy, directoryLookPath, runnerForCommands = oldFetch, oldLook, oldRunner })
	return directoryFixture{home: home, root: root, fetcher: f}
}
func (f directoryFixture) recipe(t *testing.T, name, version, payload string) *manifest.Recipe {
	t.Helper()
	identity := packagebundle.Identity{Name: name, Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH}
	data, err := packagebundle.Build(identity, "sha256:"+strings.Repeat("a", 64), []packagebundle.File{{Path: "README.md", Mode: 0644, Data: []byte(payload)}})
	if err != nil {
		t.Fatal(err)
	}
	url := "https://example.test/" + name + "-" + version + ".tar.gz"
	f.fetcher.bodies[url] = data
	rec := &manifest.Recipe{Meta: manifest.Meta{APIVersion: "patronus/v3", Family: manifest.FamilyRecipe, Name: name, Version: version, Role: manifest.RoleSandbox}, Delivery: &manifest.Delivery{Via: manifest.ViaFetch, Unpack: "directory", Package: &manifest.PackageIdentity{Name: name, Version: version}, Assets: []manifest.Asset{{OS: runtime.GOOS, Arch: runtime.GOARCH, URL: url, SHA256: shaHex(data), Archive: "tar.gz"}}}}
	f.saveRecipe(t, rec)
	return rec
}
func (f directoryFixture) saveRecipe(t *testing.T, rec *manifest.Recipe) {
	t.Helper()
	b, err := yaml.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, "recipes", rec.Name+".yaml"), b, 0644); err != nil {
		t.Fatal(err)
	}
}
func (f directoryFixture) readme(name string) string {
	return filepath.Join(f.home, ".patronus", "packages", name, "README.md")
}
func (f directoryFixture) install(t *testing.T, name string) {
	t.Helper()
	if _, _, err := runInstall(t, name, "--deploy"); err != nil {
		t.Fatal(err)
	}
}
func (f directoryFixture) profile(t *testing.T, names ...string) {
	t.Helper()
	p := &manifest.Profile{Meta: manifest.Meta{APIVersion: "patronus/v2", Family: manifest.FamilyProfile, Name: "directory-test", Version: "1.0.0", Role: manifest.RoleLifecycle}, Layers: manifest.ProfileLayers{Sandbox: names}}
	b, err := yaml.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(f.root, "profiles"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, "profiles", "directory-test.yaml"), b, 0644); err != nil {
		t.Fatal(err)
	}
}
func requireNoPackageWrites(t *testing.T, home string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(home, ".patronus")); !os.IsNotExist(err) {
		t.Fatalf("unexpected package/state writes: %v", err)
	}
}

func TestDirectoryDefaultFetchDenied(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("default directory fetch was not denied")
		}
	}()
	_, _ = directoryFetcherForDeploy.Open(context.Background(), "https://example.test/not-served")
}

func TestDirectoryInstallDryRunAndLocal(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	for _, args := range [][]string{{"pi-sandbox"}, {"pi-sandbox", "--target", "all"}, {"pi-sandbox", "--target", "claude"}} {
		out, _, err := runInstall(t, args...)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "PATH readiness") || !strings.Contains(out, "SHA-256:") || !strings.Contains(out, "Package: pi-sandbox@1.0.0") {
			t.Fatal(out)
		}
	}
	if _, _, err := runInstall(t, "pi-sandbox", "--local", "--deploy"); err == nil {
		t.Fatal("local directory install succeeded")
	}
	if f.fetcher.calls != 0 || len(runnerForCommands.(*fakeRunner).ran) != 0 {
		t.Fatal("dry run fetched or executed")
	}
	requireNoPackageWrites(t, f.home)
}

func TestDirectoryInstallReadinessAndReferenceRepair(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	out, _, err := runInstall(t, "pi-sandbox", "--deploy")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Running this kit requires sbx and separate provider authentication.", "Package installed; install sbx before using it.", f.readme("pi-sandbox"), "Packages: 1 committed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q: %s", want, out)
		}
	}
	if string(mustRead(t, f.readme("pi-sandbox"))) != "kit" {
		t.Fatal("payload mismatch")
	}
	path := filepath.Join(f.home, ".patronus", "state.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	directoryLookPath = func(string) (string, error) { return "/bin/sbx", nil }
	out, _, err = runInstall(t, "pi-sandbox", "--deploy")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "install sbx before") || !strings.Contains(out, "Packages: 0 committed, 1 unchanged") {
		t.Fatal(out)
	}
	s, err := state.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Items) != 1 || s.Items[0].PackageReceipt != "pi-sandbox" || len(s.Items[0].Files) != 0 {
		t.Fatalf("reference = %#v", s)
	}
	if f.fetcher.calls != 1 {
		t.Fatalf("unchanged fetched again: %d", f.fetcher.calls)
	}
}

func TestDirectoryInstallYesPreservesEditsAndForceRepairs(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	f.install(t, "pi-sandbox")
	if err := os.WriteFile(f.readme("pi-sandbox"), []byte("user edit"), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := runInstall(t, "pi-sandbox", "--deploy", "--yes")
	if err == nil || !strings.Contains(err.Error(), "--force") || !strings.Contains(err.Error(), "README.md") {
		t.Fatalf("conflict = %v", err)
	}
	if string(mustRead(t, f.readme("pi-sandbox"))) != "user edit" {
		t.Fatal("edit overwritten")
	}
	if _, _, err := runInstall(t, "pi-sandbox", "--deploy", "--force"); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, f.readme("pi-sandbox"))) != "kit" || f.fetcher.calls != 2 {
		t.Fatal("force did not reacquire/repair")
	}
}

func TestDirectoryForceRejectsUnknownOwnership(t *testing.T) {
	for _, owned := range []bool{false, true} {
		t.Run(fmt.Sprint(owned), func(t *testing.T) {
			f := newDirectoryFixture(t)
			f.recipe(t, "pi-sandbox", "1.0.0", "kit")
			if owned {
				f.install(t, "pi-sandbox")
			}
			path := filepath.Join(filepath.Dir(f.readme("pi-sandbox")), "unknown")
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("user"), 0644); err != nil {
				t.Fatal(err)
			}
			calls := f.fetcher.calls
			if _, _, err := runInstall(t, "pi-sandbox", "--deploy", "--force"); err == nil {
				t.Fatal("unknown ownership accepted")
			}
			if string(mustRead(t, path)) != "user" || calls != f.fetcher.calls {
				t.Fatal("unknown tree changed/fetched")
			}
		})
	}
}

func TestDirectoryUpdateUsesReceiptsAndExplicitForce(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "old")
	f.install(t, "pi-sandbox")
	if err := os.Remove(filepath.Join(f.home, ".patronus", "state.json")); err != nil {
		t.Fatal(err)
	}
	f.recipe(t, "pi-sandbox", "2.0.0", "new")
	if err := os.WriteFile(f.readme("pi-sandbox"), []byte("edit"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runUpdate(t, "pi-sandbox", "--deploy"); err == nil {
		t.Fatal("update implicitly forced")
	}
	if string(mustRead(t, f.readme("pi-sandbox"))) != "edit" {
		t.Fatal("update overwrote edit")
	}
	if _, _, err := runUpdate(t, "pi-sandbox", "--deploy", "--force"); err != nil {
		t.Fatal(err)
	}
	receipt, err := packagestate.Load(f.home, "pi-sandbox")
	if err != nil || receipt.RecipeVersion != "2.0.0" {
		t.Fatalf("receipt %v %v", receipt, err)
	}
	if err := os.WriteFile(f.readme("pi-sandbox"), []byte("edit again"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runUpdate(t, "pi-sandbox", "--deploy"); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, f.readme("pi-sandbox"))) != "edit again" {
		t.Fatal("unchanged version repaired without explicit force")
	}
	if _, _, err := runUpdate(t, "pi-sandbox", "--deploy", "--force"); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, f.readme("pi-sandbox"))) != "new" {
		t.Fatal("same-version force did not repair")
	}
}

func TestDirectoryBatchPreflightBeforeLegacyOrDirectory(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "aa-kit", "1.0.0", "old")
	f.install(t, "aa-kit")
	bad := f.recipe(t, "zz-kit", "1.0.0", "old")
	f.install(t, "zz-kit")
	f.recipe(t, "aa-kit", "2.0.0", "new")
	bad.Version = "2.0.0"
	bad.Delivery.Assets[0].OS = "windows"
	if runtime.GOOS == "windows" {
		bad.Delivery.Assets[0].OS = "linux"
	}
	f.saveRecipe(t, bad)
	before := f.fetcher.calls
	if _, _, err := runUpdate(t, "aa-kit", "zz-kit", "--deploy"); err == nil {
		t.Fatal("unsupported batch accepted")
	}
	if string(mustRead(t, f.readme("aa-kit"))) != "old" || f.fetcher.calls != before {
		t.Fatal("earlier recipe mutated before full preflight")
	}
	f.profile(t, "aa-kit", "zz-kit", "fix-bin")
	if _, _, err := runInstall(t, "--profile", "directory-test", "--deploy"); err == nil {
		t.Fatal("unsupported profile accepted")
	}
	if _, err := os.Stat(filepath.Join(f.home, ".patronus", "bin", "fix-bin")); !os.IsNotExist(err) {
		t.Fatal("legacy wrote before directory preflight")
	}
}

func TestDirectoryProfilePinsAndDirectInstallIgnoresLock(t *testing.T) {
	f := newDirectoryFixture(t)
	old := f.recipe(t, "pi-sandbox", "1.0.0", "old")
	f.profile(t, "pi-sandbox")
	f.recipe(t, "pi-sandbox", "2.0.0", "new")
	pin := &lock.Lock{Version: lock.Version, Profile: "directory-test", Entries: []lock.Entry{{Name: old.Name, Kind: "recipe", Version: old.Version, SHA256: strings.Repeat("b", 64), Delivery: old.Delivery}}}
	if err := lock.Save(filepath.Join(f.root, "patronus.lock"), pin); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runInstall(t, "--profile", "directory-test", "--deploy"); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, f.readme(old.Name))) != "old" {
		t.Fatal("profile ignored delivery pin")
	}
	f.install(t, old.Name)
	if string(mustRead(t, f.readme(old.Name))) != "new" {
		t.Fatal("direct install applied profile lock")
	}
	f.recipe(t, "pi-sandbox", "3.0.0", "latest")
	if _, _, err := runUpdate(t, "pi-sandbox", "--deploy"); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, f.readme(old.Name))) != "latest" {
		t.Fatal("update applied profile lock")
	}
}

func TestDirectoryMalformedProfileLockFailsBeforeWrites(t *testing.T) {
	for _, body := range []string{"{broken", `{"version":99}`, `{"version":2,"entries":[{"name":"missing","kind":"recipe","delivery":{"unpack":"directory"}}]}`} {
		t.Run(body, func(t *testing.T) {
			f := newDirectoryFixture(t)
			f.recipe(t, "pi-sandbox", "1.0.0", "kit")
			f.profile(t, "pi-sandbox")
			if err := os.WriteFile(filepath.Join(f.root, "patronus.lock"), []byte(body), 0644); err != nil {
				t.Fatal(err)
			}
			if _, _, err := runInstall(t, "--profile", "directory-test", "--deploy"); err == nil {
				t.Fatal("bad lock ignored")
			}
			requireNoPackageWrites(t, f.home)
		})
	}
}

func TestDirectoryCommittedCleanupErrorRepairsReference(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	old := directoryServiceForDeploy
	directoryServiceForDeploy = func(home string) *packagedelivery.Service {
		return &packagedelivery.Service{Home: home, Fetcher: directoryFetcherForDeploy, Fault: func(point string) error {
			if point == "after-committed" {
				return errors.New("committed fault")
			}
			return nil
		}}
	}
	t.Cleanup(func() { directoryServiceForDeploy = old })
	_, _, err := runInstall(t, "pi-sandbox", "--deploy")
	if err == nil || !strings.Contains(err.Error(), "committed fault") {
		t.Fatal(err)
	}
	s, err := state.Load(filepath.Join(f.home, ".patronus", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Items) != 1 || s.Items[0].PackageReceipt != "pi-sandbox" {
		t.Fatal("committed reference missing")
	}
	if string(mustRead(t, f.readme("pi-sandbox"))) != "kit" {
		t.Fatal("committed tree rolled back")
	}
	directoryServiceForDeploy = old
	f.install(t, "pi-sandbox")
}

func TestDirectoryReferenceFailureIsFatalAndRepairable(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	old := directoryServiceForDeploy
	directoryServiceForDeploy = func(home string) *packagedelivery.Service {
		return &packagedelivery.Service{Home: home, Fetcher: directoryFetcherForDeploy, Fault: func(point string) error {
			if point == "after-committed" {
				return os.Mkdir(filepath.Join(home, ".patronus", "state.json"), 0755)
			}
			return nil
		}}
	}
	t.Cleanup(func() { directoryServiceForDeploy = old })
	_, _, err := runInstall(t, "pi-sandbox", "--deploy")
	if err == nil || !strings.Contains(err.Error(), "repair discovery reference") {
		t.Fatal(err)
	}
	if string(mustRead(t, f.readme("pi-sandbox"))) != "kit" {
		t.Fatal("reference failure rolled back package")
	}
	if err := os.Remove(filepath.Join(f.home, ".patronus", "state.json")); err != nil {
		t.Fatal(err)
	}
	directoryServiceForDeploy = old
	f.install(t, "pi-sandbox")
}

func TestDirectoryLegacyOnlyDeployDoesNotCreatePackageState(t *testing.T) {
	home := t.TempDir()
	cs := &diff.ChangeSet{}
	result, err := deployDirectories(context.Background(), home, cs, false)
	if err != nil || result.Legacy != cs {
		t.Fatalf("legacy result %v %v", result, err)
	}
	requireNoPackageWrites(t, home)
}

func TestDirectoryPlanCatalogSnapshotIsCloned(t *testing.T) {
	f := newDirectoryFixture(t)
	rec := f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	cat := &registry.Catalog{Recipes: []registry.RecipeEntry{{Manifest: rec}}}
	cmd := &cobra.Command{}
	p, err := planInstall(cmd, installPlanRequest{Names: []string{rec.Name, rec.Name}, Home: f.home, ProjectDir: f.root, Catalog: cat})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Changes.Diffs) != 1 || cat.Recipes[0].Manifest != rec {
		t.Fatal("snapshot mutated or duplicated")
	}
}

func TestDirectoryRecoveryRepairsReferenceBeforeNextFetchFailure(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "old")
	original := directoryServiceForDeploy
	directoryServiceForDeploy = func(home string) *packagedelivery.Service {
		return &packagedelivery.Service{Home: home, Fetcher: directoryFetcherForDeploy, Fault: func(point string) error {
			if point == "after-committed" {
				return errors.New("committed fault")
			}
			return nil
		}}
	}
	t.Cleanup(func() { directoryServiceForDeploy = original })
	if _, _, err := runInstall(t, "pi-sandbox", "--deploy"); err == nil {
		t.Fatal("fault not reached")
	}
	if err := os.Remove(filepath.Join(f.home, ".patronus", "state.json")); err != nil {
		t.Fatal(err)
	}
	directoryServiceForDeploy = original
	rec := f.recipe(t, "pi-sandbox", "2.0.0", "new")
	delete(f.fetcher.bodies, rec.Delivery.Assets[0].URL)
	if _, _, err := runInstall(t, "pi-sandbox", "--deploy"); err == nil {
		t.Fatal("fetch should fail")
	}
	s, err := state.Load(filepath.Join(f.home, ".patronus", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Items) != 1 || s.Items[0].ItemVersion != "1.0.0" {
		t.Fatalf("recovered reference missing: %#v", s.Items)
	}
}

func TestDirectoryIncompletePinIsRejected(t *testing.T) {
	f := newDirectoryFixture(t)
	rec := f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	f.profile(t, "pi-sandbox")
	l := &lock.Lock{Version: lock.Version, Profile: "directory-test", Entries: []lock.Entry{{Name: rec.Name, Kind: "recipe", Version: "0.1.0"}}}
	if err := lock.Save(filepath.Join(f.root, "patronus.lock"), l); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runInstall(t, "--profile", "directory-test", "--deploy"); err == nil {
		t.Fatal("directory lock without delivery silently followed index")
	}
	requireNoPackageWrites(t, f.home)
}

func TestDirectoryMissingReceiptIsActionable(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	f.install(t, "pi-sandbox")
	release, err := packagestate.Acquire(f.home)
	if err != nil {
		t.Fatal(err)
	}
	if err := packagestate.DeleteReceipt(f.home, "pi-sandbox"); err != nil {
		t.Fatal(err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runUpdate(t, "pi-sandbox", "--deploy"); err == nil || !strings.Contains(err.Error(), "no receipt") {
		t.Fatalf("missing receipt = %v", err)
	}
}

func TestDirectoryMixedDeployStopsAfterFailedRecipe(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "aa-kit", "1.0.0", "first")
	bad := f.recipe(t, "zz-kit", "1.0.0", "second")
	delete(f.fetcher.bodies, bad.Delivery.Assets[0].URL)
	if _, _, err := runInstall(t, "aa-kit", "zz-kit", "fix-bin", "--deploy"); err == nil {
		t.Fatal("fetch failure ignored")
	}
	if string(mustRead(t, f.readme("aa-kit"))) != "first" {
		t.Fatal("earlier committed recipe rolled back")
	}
	if _, err := os.Stat(f.readme("zz-kit")); !os.IsNotExist(err) {
		t.Fatal("failed recipe committed")
	}
	if _, err := os.Stat(filepath.Join(f.home, ".patronus", "bin", "fix-bin")); !os.IsNotExist(err) {
		t.Fatal("legacy applied after failure")
	}
}

func TestDirectoryUpdateUnknownContentBlocksWholeBatch(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "aa-kit", "1.0.0", "old")
	f.install(t, "aa-kit")
	f.recipe(t, "zz-kit", "1.0.0", "old")
	f.install(t, "zz-kit")
	f.recipe(t, "aa-kit", "2.0.0", "new")
	f.recipe(t, "zz-kit", "2.0.0", "new")
	if err := os.Mkdir(filepath.Join(filepath.Dir(f.readme("zz-kit")), "user-cache"), 0755); err != nil {
		t.Fatal(err)
	}
	calls := f.fetcher.calls
	if _, _, err := runUpdate(t, "--all", "--deploy", "--force"); err == nil {
		t.Fatal("unknown content bypassed")
	}
	if string(mustRead(t, f.readme("aa-kit"))) != "old" || f.fetcher.calls != calls {
		t.Fatal("earlier directory mutated")
	}
}

func TestDirectoryPendingWithoutReceiptIsReadOnlyUntilSelectedDeploy(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	original := directoryServiceForDeploy
	var unknown string
	directoryServiceForDeploy = func(home string) *packagedelivery.Service {
		return &packagedelivery.Service{Home: home, Fetcher: directoryFetcherForDeploy, Fault: func(point string) error {
			if point != "after-prepared" {
				return nil
			}
			tx, err := packagestate.ReadTransaction(home, "pi-sandbox")
			if err != nil {
				return err
			}
			unknown = filepath.Join(tx.Stage, "intervening")
			if err := os.WriteFile(unknown, []byte("user"), 0644); err != nil {
				return err
			}
			return errors.New("interrupt with intervening stage content")
		}}
	}
	t.Cleanup(func() { directoryServiceForDeploy = original })
	if _, _, err := runInstall(t, "pi-sandbox", "--deploy"); !errors.Is(err, packagedelivery.ErrRecoveryRequired) {
		t.Fatalf("pending error = %v", err)
	}
	directoryServiceForDeploy = original
	receipt, err := packagestate.Load(f.home, "pi-sandbox")
	if err != nil || receipt != nil {
		t.Fatalf("receipt = %v %v", receipt, err)
	}
	journal := filepath.Join(f.home, ".patronus", "package-state", "transactions", "pi-sandbox", "transaction.json")
	before := mustRead(t, journal)
	calls := f.fetcher.calls
	out, _, err := runInstall(t, "pi-sandbox")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "pending recovery") || !bytes.Equal(before, mustRead(t, journal)) || f.fetcher.calls != calls {
		t.Fatal("read-only plan recovered/fetched or hid pending work")
	}
	f.recipe(t, "independent", "1.0.0", "other")
	f.install(t, "independent")
	if !bytes.Equal(before, mustRead(t, journal)) {
		t.Fatal("unrelated pending recipe was changed")
	}
	if err := os.Remove(unknown); err != nil {
		t.Fatal(err)
	}
	f.install(t, "pi-sandbox")
	tx, err := packagestate.ReadTransaction(f.home, "pi-sandbox")
	if err != nil || tx != nil {
		t.Fatalf("recovery did not finish: %v %v", tx, err)
	}
	if string(mustRead(t, f.readme("pi-sandbox"))) != "kit" {
		t.Fatal("retry did not install")
	}
}

func TestDirectoryMixedUpdatePreservesLegacyOverwriteBehavior(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	f.install(t, "pi-sandbox")
	// Reuse the in-memory legacy fetch protocol; never fall back to its HTTP default.
	original := fetcherForDeploy
	fetcherForDeploy = &directoryLegacyFetcher{body: fixRawBinary}
	t.Cleanup(func() { fetcherForDeploy = original })
	f.install(t, "fix-bin")
	binary := filepath.Join(f.home, ".patronus", "bin", "fix-bin")
	if err := os.WriteFile(binary, []byte("legacy edit"), 0755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(f.root, "recipes", "fix-bin.yaml")
	content := strings.Replace(string(mustRead(t, manifestPath)), "version: 1.0.0", "version: 2.0.0", 1)
	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	f.recipe(t, "pi-sandbox", "2.0.0", "new kit")
	if _, _, err := runUpdate(t, "fix-bin", "pi-sandbox", "--deploy"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(mustRead(t, binary), fixRawBinary) {
		t.Fatal("legacy update stopped overwriting")
	}
	if string(mustRead(t, f.readme("pi-sandbox"))) != "new kit" {
		t.Fatal("directory update missing")
	}
}

type directoryLegacyFetcher struct{ body []byte }

func (f *directoryLegacyFetcher) Fetch(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f.body)), nil
}

func TestDirectoryLockBusyNeverFetches(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	release, err := packagestate.Acquire(f.home)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := release(); err != nil {
			t.Error(err)
		}
	}()
	if _, _, err := runInstall(t, "pi-sandbox", "--deploy"); !errors.Is(err, packagestate.ErrBusy) {
		t.Fatalf("lock conflict = %v", err)
	}
	if f.fetcher.calls != 0 {
		t.Fatal("busy deployment fetched")
	}
}

func TestDirectoryUpdateDryRunNeverFetchesOrRecovers(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "old")
	f.install(t, "pi-sandbox")
	old := mustRead(t, filepath.Join(f.home, ".patronus", "package-state", "pi-sandbox.json"))
	calls := f.fetcher.calls
	f.recipe(t, "pi-sandbox", "2.0.0", "new")
	out, _, err := runUpdate(t, "pi-sandbox")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Package: pi-sandbox@2.0.0") || f.fetcher.calls != calls {
		t.Fatal("dry update fetched or omitted pin")
	}
	if !bytes.Equal(old, mustRead(t, filepath.Join(f.home, ".patronus", "package-state", "pi-sandbox.json"))) || string(mustRead(t, f.readme("pi-sandbox"))) != "old" {
		t.Fatal("dry update wrote")
	}
}

func TestDirectorySelectedSnapshotPinsWithoutRegistryFetch(t *testing.T) {
	f := newDirectoryFixture(t)
	old := f.recipe(t, "pi-sandbox", "1.0.0", "old")
	f.profile(t, "pi-sandbox")
	f.recipe(t, "pi-sandbox", "2.0.0", "new")
	cat, err := registry.NewLocalRegistry(f.root).Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	l := &lock.Lock{Version: lock.Version, Profile: "directory-test", Entries: []lock.Entry{{Name: old.Name, Kind: "recipe", Version: old.Version, SHA256: strings.Repeat("a", 64), Delivery: old.Delivery}}}
	if err := lock.Save(filepath.Join(project, "patronus.lock"), l); err != nil {
		t.Fatal(err)
	}
	p, err := planInstall(&cobra.Command{}, installPlanRequest{Profile: "directory-test", Home: f.home, ProjectDir: project, Catalog: cat})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Changes.Diffs) != 1 || p.Changes.Diffs[0].Directory.RecipeVersion != "1.0.0" || findRecipe(cat, "pi-sandbox").Manifest.Version != "2.0.0" {
		t.Fatal("snapshot was fetched/mutated or lock pin omitted")
	}
	requireNoPackageWrites(t, f.home)
}

func TestDirectoryMixedUpdatePreflightBlocksLegacyWrites(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	f.install(t, "pi-sandbox")
	original := fetcherForDeploy
	fetcherForDeploy = &directoryLegacyFetcher{body: fixRawBinary}
	t.Cleanup(func() { fetcherForDeploy = original })
	f.install(t, "fix-bin")
	binary := filepath.Join(f.home, ".patronus", "bin", "fix-bin")
	if err := os.WriteFile(binary, []byte("legacy edit"), 0755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(f.root, "recipes", "fix-bin.yaml")
	body := strings.Replace(string(mustRead(t, manifestPath)), "version: 1.0.0", "version: 2.0.0", 1)
	if err := os.WriteFile(manifestPath, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	f.recipe(t, "pi-sandbox", "2.0.0", "new")
	if err := os.WriteFile(filepath.Join(filepath.Dir(f.readme("pi-sandbox")), "unknown"), []byte("user"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runUpdate(t, "fix-bin", "pi-sandbox", "--deploy", "--force"); err == nil {
		t.Fatal("unknown content ignored")
	}
	if string(mustRead(t, binary)) != "legacy edit" {
		t.Fatal("legacy applied before batch preflight")
	}
}

func TestDirectoryDigestFormsAreCanonical(t *testing.T) {
	f := newDirectoryFixture(t)
	rec := f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	f.install(t, "pi-sandbox")
	rec.Delivery.Assets[0].SHA256 = "sha256:" + strings.ToUpper(rec.Delivery.Assets[0].SHA256)
	f.saveRecipe(t, rec)
	out, _, err := runInstall(t, "pi-sandbox", "--deploy")
	if err != nil {
		t.Fatal(err)
	}
	if f.fetcher.calls != 1 || !strings.Contains(out, "Packages: 0 committed, 1 unchanged") {
		t.Fatal("equivalent digest caused replacement")
	}
}

func TestDirectoryUnchangedKeepsDiscoveryReference(t *testing.T) {
	f := newDirectoryFixture(t)
	f.recipe(t, "pi-sandbox", "1.0.0", "kit")
	f.install(t, "pi-sandbox")
	path := filepath.Join(f.home, ".patronus", "state.json")
	if err := os.Chtimes(path, time.Unix(1, 0), time.Unix(1, 0)); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	f.install(t, "pi-sandbox")
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("unchanged discovery reference was rewritten")
	}
}
