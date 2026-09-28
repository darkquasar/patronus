package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagedelivery"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
)

type lifecycleFixture struct {
	directoryFixture
	server   *httptest.Server
	mu       sync.Mutex
	archives map[string][]byte
	requests int
}

func newLifecycleFixture(t *testing.T) *lifecycleFixture {
	t.Helper()
	f := &lifecycleFixture{directoryFixture: newDirectoryFixture(t), archives: make(map[string][]byte)}
	// Receipts use canonical roots, including macOS temporary-directory aliases.
	canonical, err := filepath.EvalSymlinks(f.home)
	if err != nil {
		t.Fatal(err)
	}
	f.home = canonical
	f.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.requests++
		body, ok := f.archives[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(f.server.Close)
	directoryFetcherForDeploy = packagedelivery.HTTPSFetcher{Client: f.server.Client()}
	directoryLookPath = func(string) (string, error) { return "/fixture/sbx", nil }
	t.Cleanup(func() {
		runner, ok := runnerForCommands.(*fakeRunner)
		if !ok || len(runner.ran) != 0 {
			t.Errorf("runtime runner invoked: %+v", runner)
		}
	})
	return f
}

func (f *lifecycleFixture) publish(t *testing.T, name, version string, files ...packagebundle.File) *manifest.Recipe {
	t.Helper()
	rec := f.recipe(t, name, version, "unused")
	data, err := packagebundle.Build(packagebundle.Identity{Name: name, Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH}, "sha256:"+strings.Repeat("a", 64), files)
	if err != nil {
		t.Fatal(err)
	}
	path := "/" + name + "-" + version + ".tar.gz"
	f.mu.Lock()
	f.archives[path] = data
	f.mu.Unlock()
	rec.Delivery.Assets[0].URL = f.server.URL + path
	rec.Delivery.Assets[0].SHA256 = shaHex(data)
	f.saveRecipe(t, rec)
	return rec
}

func lifecycleFile(name, body string) packagebundle.File {
	return packagebundle.File{Path: name, Mode: 0644, Data: []byte(body)}
}

// snapshot includes directories, permission bits and every file byte, including receipts.
func lifecycleSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) && path == root {
			return nil
		}
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := fmt.Sprintf("%o", info.Mode())
		if !entry.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += " " + string(data)
		}
		result[rel] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func lifecycleEqual(t *testing.T, before, after map[string]string) {
	t.Helper()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("state changed\nbefore: %#v\nafter: %#v", before, after)
	}
}

func lifecycleReceipt(t *testing.T, home, name string) *packagestate.Receipt {
	t.Helper()
	receipt, err := packagestate.Load(home, name)
	if err != nil || receipt == nil {
		t.Fatalf("receipt %s: %+v %v", name, receipt, err)
	}
	return receipt
}

func (f *lifecycleFixture) assertPackage(t *testing.T, rec *manifest.Recipe, files ...packagebundle.File) {
	t.Helper()
	receipt := lifecycleReceipt(t, f.home, rec.Name)
	if receipt.SchemaVersion != 1 || receipt.Recipe != rec.Name || receipt.Root != filepath.Join(f.home, ".patronus", "packages", rec.Name) || receipt.Identity.Name != rec.Delivery.Package.Name || receipt.Identity.OS != runtime.GOOS || receipt.Identity.Arch != runtime.GOARCH || receipt.RecipeVersion != rec.Version || receipt.Identity.Version != rec.Delivery.Package.Version || receipt.URL != rec.Delivery.Assets[0].URL || receipt.ArchiveSHA256 != "sha256:"+strings.TrimPrefix(rec.Delivery.Assets[0].SHA256, "sha256:") {
		t.Fatalf("wrong receipt: %+v", receipt)
	}
	f.mu.Lock()
	archive := f.archives["/"+rec.Name+"-"+rec.Delivery.Package.Version+".tar.gz"]
	f.mu.Unlock()
	bundle, err := packagebundle.Decode(bytes.NewReader(archive), receipt.Identity, packagebundle.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{".": "20000000755"}
	entries := make(map[string]packagebundle.Entry)
	for _, file := range bundle.Files {
		expected[file.Path] = fmt.Sprintf("%o %s", file.Mode, file.Data)
		entries[file.Path] = packagebundle.Entry{Path: file.Path, Mode: file.Mode, SHA256: "sha256:" + shaHex(file.Data)}
	}
	if len(receipt.Files) != len(entries) {
		t.Fatalf("inventory: %+v", receipt.Files)
	}
	for _, entry := range receipt.Files {
		if entries[entry.Path] != entry {
			t.Fatalf("inventory entry: %+v", entry)
		}
	}
	if !reflect.DeepEqual(receipt.Directories, []string{"."}) {
		t.Fatalf("directories: %v", receipt.Directories)
	}
	lifecycleEqual(t, expected, lifecycleSnapshot(t, receipt.Root))
	for _, file := range files {
		if !bytes.Equal(mustRead(t, filepath.Join(receipt.Root, file.Path)), file.Data) {
			t.Fatalf("payload %s", file.Path)
		}
	}
	tx, err := packagestate.ReadTransaction(f.home, rec.Name)
	if err != nil || tx != nil {
		t.Fatalf("pending transaction: %+v %v", tx, err)
	}
}

func lifecycleFault(t *testing.T, fault func(string) error) func() {
	t.Helper()
	old := directoryServiceForDeploy
	directoryServiceForDeploy = func(home string) *packagedelivery.Service { s := old(home); s.Fault = fault; return s }
	restore := func() { directoryServiceForDeploy = old }
	t.Cleanup(restore)
	return restore
}

func TestDirectoryLifecycle(t *testing.T) {
	f := newLifecycleFixture(t)
	first := lifecycleFile("README.md", "first")
	obsolete := lifecycleFile("obsolete.txt", "old config")
	rec := f.publish(t, "kit", "1.0.0", first, obsolete)
	if _, _, err := runInstall(t, "kit", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	requireNoPackageWrites(t, f.home)
	f.mu.Lock()
	requests := f.requests
	f.mu.Unlock()
	if requests != 0 {
		t.Fatal("dry-run fetched")
	}
	f.install(t, "kit")
	f.assertPackage(t, rec, first, obsolete)
	lifecycleStatus(t, "kit", "installed")
	before := lifecycleSnapshot(t, filepath.Join(f.home, ".patronus"))
	f.install(t, "kit")
	lifecycleEqual(t, before, lifecycleSnapshot(t, filepath.Join(f.home, ".patronus")))
	rootBefore := lifecycleSnapshot(t, filepath.Dir(f.readme("kit")))
	rec.Version = "1.1.0"
	f.saveRecipe(t, rec)
	if _, _, err := runUpdate(t, "kit", "--deploy"); err != nil {
		t.Fatal(err)
	}
	f.assertPackage(t, rec, first, obsolete)
	lifecycleEqual(t, rootBefore, lifecycleSnapshot(t, filepath.Dir(f.readme("kit"))))
	second := lifecycleFile("README.md", "second")
	rec = f.publish(t, "kit", "2.0.0", second)
	if _, _, err := runUpdate(t, "kit", "--deploy"); err != nil {
		t.Fatal(err)
	}
	f.assertPackage(t, rec, second)
	writeLifecycle(t, f.readme("kit"), "user edit")
	before = lifecycleSnapshot(t, filepath.Join(f.home, ".patronus"))
	if _, _, err := runInstall(t, "kit", "--deploy", "--yes"); err == nil {
		t.Fatal("owned edit accepted")
	}
	lifecycleEqual(t, before, lifecycleSnapshot(t, filepath.Join(f.home, ".patronus")))
	if _, _, err := runInstall(t, "kit", "--deploy", "--force"); err != nil {
		t.Fatal(err)
	}
	f.assertPackage(t, rec, second)
	unknown := filepath.Join(filepath.Dir(f.readme("kit")), "mine")
	writeLifecycle(t, unknown, "personal")
	before = lifecycleSnapshot(t, filepath.Join(f.home, ".patronus"))
	if _, _, err := runInstall(t, "kit", "--deploy", "--force"); err == nil {
		t.Fatal("unknown accepted")
	}
	lifecycleEqual(t, before, lifecycleSnapshot(t, filepath.Join(f.home, ".patronus")))
	if err := os.Remove(unknown); err != nil {
		t.Fatal(err)
	}
	oldRec := rec
	f.publish(t, "kit", "3.0.0", lifecycleFile("README.md", "third"))
	restore := lifecycleFault(t, func(point string) error {
		if point == "after-stage-rename" {
			return errors.New("returned precommit fault")
		}
		return nil
	})
	if _, _, err := runInstall(t, "kit", "--deploy"); err == nil || !strings.Contains(err.Error(), "returned precommit fault") {
		t.Fatalf("fault: %v", err)
	}
	restore()
	f.assertPackage(t, oldRec, second)
	f.crash(t, "replace")
	tx, err := packagestate.ReadTransaction(f.home, "kit")
	if err != nil || tx == nil || tx.Operation != "replace" || tx.Phase != packagestate.Prepared || tx.Intent != "promote-new" {
		t.Fatalf("crash journal: %+v %v", tx, err)
	}
	if string(mustRead(t, f.readme("kit"))) != "third" {
		t.Fatal("crash did not place candidate")
	}
	if lifecycleReceipt(t, f.home, "kit").RecipeVersion != "2.0.0" {
		t.Fatal("uncommitted receipt changed")
	}
	lifecycleStatus(t, "kit", "recovery-required")
	pendingBefore := lifecycleSnapshot(t, filepath.Join(f.home, ".patronus"))
	if _, _, err := runInstall(t, "kit", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	lifecycleEqual(t, pendingBefore, lifecycleSnapshot(t, filepath.Join(f.home, ".patronus")))
	// Select the previous package: restart must restore it before deciding to skip.
	f.saveRecipe(t, oldRec)
	f.install(t, "kit")
	f.assertPackage(t, oldRec, second)
	if err := os.Remove(filepath.Join(f.home, ".patronus", "state.json")); err != nil {
		t.Fatal(err)
	}
	lifecycleStatus(t, "kit", "installed")
	if _, _, err := runUpdate(t, "kit", "--deploy"); err != nil {
		t.Fatal(err)
	}
	f.assertPackage(t, oldRec, second)
	f.crash(t, "remove")
	tx, err = packagestate.ReadTransaction(f.home, "kit")
	if err != nil || tx == nil || tx.Operation != "remove" || len(tx.PendingRemove) != 2 {
		t.Fatalf("removal journal: %+v %v", tx, err)
	}
	if _, err := os.Stat(filepath.Join(tx.Root, tx.PendingRemove[0])); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unlink was not durable: %v", err)
	}
	lifecycleStatus(t, "kit", "recovery-required")
	f.install(t, "kit")
	f.assertPackage(t, oldRec, second)
	writeLifecycle(t, f.readme("kit"), "keep edited")
	writeLifecycle(t, unknown, "personal")
	if _, _, err := execRemove(t, "kit", "--deploy"); err != nil {
		t.Fatal(err)
	}
	reduced := lifecycleReceipt(t, f.home, "kit")
	if len(reduced.Files) != 1 || reduced.Files[0].Path != "README.md" {
		t.Fatalf("reduced receipt: %+v", reduced)
	}
	lifecycleEqual(t, map[string]string{".": "20000000755", "README.md": "644 keep edited", "mine": "644 personal"}, lifecycleSnapshot(t, reduced.Root))
	if _, _, err := execRemove(t, "kit", "--deploy", "--force"); err != nil {
		t.Fatal(err)
	}
	receipt, err := packagestate.Load(f.home, "kit")
	if err != nil || receipt != nil {
		t.Fatalf("final receipt: %+v %v", receipt, err)
	}
	lifecycleEqual(t, map[string]string{".": "20000000755", "mine": "644 personal"}, lifecycleSnapshot(t, reduced.Root))
	tx, err = packagestate.ReadTransaction(f.home, "kit")
	if err != nil || tx != nil {
		t.Fatalf("final journal: %+v %v", tx, err)
	}
	discovery, err := state.Load(filepath.Join(f.home, ".patronus", "state.json"))
	if err != nil || len(discovery.Items) != 0 {
		t.Fatalf("final discovery: %+v %v", discovery, err)
	}
}

func writeLifecycle(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func (f *lifecycleFixture) crash(t *testing.T, operation string) {
	t.Helper()
	cert := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.server.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestDirectoryLifecycleCrashHelper$")
	cmd.Env = append(os.Environ(), "PATRONUS_LIFECYCLE_CHILD="+operation, "PATRONUS_LIFECYCLE_CA="+cert)
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 97 {
		t.Fatalf("crash exit: %v %s", err, output)
	}
}

func TestDirectoryLifecycleCrashHelper(t *testing.T) {
	operation := os.Getenv("PATRONUS_LIFECYCLE_CHILD")
	if operation == "" {
		return
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(mustRead(t, os.Getenv("PATRONUS_LIFECYCLE_CA"))) {
		t.Fatal("test CA")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
	defer transport.CloseIdleConnections()
	directoryFetcherForDeploy = packagedelivery.HTTPSFetcher{Client: &http.Client{Transport: transport}}
	runnerForCommands = &fakeRunner{}
	directoryLookPath = func(string) (string, error) { return "/fixture/sbx", nil }
	lifecycleFault(t, func(point string) error {
		if operation == "replace" && point == "after-stage-rename" || operation == "remove" && point == "after-unlink" || operation == "update-receipt" && point == "after-receipt-save" {
			if len(runnerForCommands.(*fakeRunner).ran) != 0 {
				os.Exit(98)
			}
			os.Exit(97)
		}
		return nil
	})
	var err error
	switch operation {
	case "update-receipt":
		_, _, err = runUpdate(t, "kit", "--deploy")
	case "remove":
		_, _, err = execRemove(t, "kit", "--deploy")
	default:
		_, _, err = runInstall(t, "kit", "--deploy")
	}
	t.Fatalf("crash hook missed: %v", err)
}

func lifecyclePreflightFixture(t *testing.T) (*lifecycleFixture, *manifest.Recipe) {
	t.Helper()
	f := newLifecycleFixture(t)
	f.publish(t, "aa-kit", "1.0.0", lifecycleFile("README.md", "a old"))
	f.install(t, "aa-kit")
	f.publish(t, "zz-kit", "1.0.0", lifecycleFile("README.md", "b old"))
	f.install(t, "zz-kit")
	f.publish(t, "aa-kit", "2.0.0", lifecycleFile("README.md", "a new"))
	b := f.publish(t, "zz-kit", "2.0.0", lifecycleFile("README.md", "b new"))
	f.profile(t, "aa-kit", "zz-kit", "fix-bin")
	return f, b
}

func lifecycleRejectPreflight(t *testing.T, f *lifecycleFixture) {
	t.Helper()
	before := lifecycleSnapshot(t, filepath.Join(f.home, ".patronus"))
	f.mu.Lock()
	requests := f.requests
	f.mu.Unlock()
	if _, _, err := runInstall(t, "--profile", "directory-test", "--deploy", "--force"); err == nil {
		t.Fatal("preflight accepted")
	}
	lifecycleEqual(t, before, lifecycleSnapshot(t, filepath.Join(f.home, ".patronus")))
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.requests != requests {
		t.Fatal("preflight fetched")
	}
}

func TestDirectoryMultiRecipePreflightPlatform(t *testing.T) {
	f, b := lifecyclePreflightFixture(t)
	b.Delivery.Assets[0].OS = "windows"
	f.saveRecipe(t, b)
	lifecycleRejectPreflight(t, f)
}

func TestDirectoryMultiRecipePreflightUnknown(t *testing.T) {
	f, _ := lifecyclePreflightFixture(t)
	writeLifecycle(t, filepath.Join(filepath.Dir(f.readme("zz-kit")), "mine"), "unknown")
	lifecycleRejectPreflight(t, f)
}

func TestDirectoryMultiRecipeLaterFailure(t *testing.T) {
	f := newLifecycleFixture(t)
	f.publish(t, "aa-kit", "1.0.0", lifecycleFile("README.md", "a old"))
	f.install(t, "aa-kit")
	oldB := f.publish(t, "zz-kit", "1.0.0", lifecycleFile("README.md", "b old"))
	f.install(t, "zz-kit")
	a := f.publish(t, "aa-kit", "2.0.0", lifecycleFile("README.md", "a new"))
	f.publish(t, "zz-kit", "2.0.0", lifecycleFile("README.md", "b new"))
	f.profile(t, "aa-kit", "zz-kit", "fix-bin")
	legacyPath := filepath.Join(f.root, "recipes", "fix-bin.yaml")
	writeLifecycle(t, legacyPath, string(mustRead(t, legacyPath))+"\nwire:\n  method: exec\n  actor: patronus\n  tools: [claude]\n  run: [\"fix-bin setup\"]\n")
	originalFetcher := fetcherForDeploy
	fetcherForDeploy = &directoryLegacyFetcher{body: fixRawBinary}
	t.Cleanup(func() { fetcherForDeploy = originalFetcher })
	sentinel := filepath.Join(f.home, "runtime-user-data")
	writeLifecycle(t, sentinel, "unmanaged runtime data")
	placements := 0
	lifecycleFault(t, func(point string) error {
		if point == "after-stage-rename" {
			placements++
			if placements == 2 {
				return errors.New("second recipe failed")
			}
		}
		return nil
	})
	if _, _, err := runInstall(t, "--profile", "directory-test", "--target", "claude", "--global", "--deploy"); err == nil || !strings.Contains(err.Error(), "second recipe failed") {
		t.Fatalf("later failure: %v", err)
	}
	f.assertPackage(t, a, lifecycleFile("README.md", "a new"))
	f.assertPackage(t, oldB, lifecycleFile("README.md", "b old"))
	if _, err := os.Stat(filepath.Join(f.home, ".patronus", "bin", "fix-bin")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy executed: %v", err)
	}
	if string(mustRead(t, sentinel)) != "unmanaged runtime data" {
		t.Fatal("runtime data changed")
	}
}

func TestDirectoryMultiRecipeReferenceFailure(t *testing.T) {
	f := newLifecycleFixture(t)
	a := f.publish(t, "aa-kit", "1.0.0", lifecycleFile("README.md", "a"))
	f.publish(t, "zz-kit", "1.0.0", lifecycleFile("README.md", "b"))
	f.profile(t, "aa-kit", "zz-kit", "fix-bin")
	lifecycleFault(t, func(point string) error {
		if point == "after-committed" {
			return os.Mkdir(filepath.Join(f.home, ".patronus", "state.json"), 0755)
		}
		return nil
	})
	if _, _, err := runInstall(t, "--profile", "directory-test", "--deploy"); err == nil || !strings.Contains(err.Error(), "repair discovery reference") {
		t.Fatalf("reference failure: %v", err)
	}
	f.assertPackage(t, a, lifecycleFile("README.md", "a"))
	for _, name := range []string{"zz-kit", "fix-bin"} {
		r, err := packagestate.Load(f.home, name)
		if err != nil || r != nil {
			t.Fatalf("later recipe %s: %+v %v", name, r, err)
		}
	}
}

func TestDirectoryMultiRecipeLockContention(t *testing.T) {
	f := newLifecycleFixture(t)
	f.publish(t, "kit", "1.0.0", lifecycleFile("README.md", "old"))
	f.install(t, "kit")
	f.publish(t, "kit", "2.0.0", lifecycleFile("README.md", "new"))
	release, err := packagestate.Acquire(f.home)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := release(); err != nil {
			t.Error(err)
		}
	}()
	before := lifecycleSnapshot(t, filepath.Join(f.home, ".patronus"))
	started := time.Now()
	if _, _, err := runInstall(t, "kit", "--deploy"); !errors.Is(err, packagestate.ErrBusy) {
		t.Fatalf("busy: %v", err)
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("lock conflict did not return promptly")
	}
	lifecycleEqual(t, before, lifecycleSnapshot(t, filepath.Join(f.home, ".patronus")))
}

func TestDirectoryMultiRecipeMixedUpdateSharedConfig(t *testing.T) {
	f := newLifecycleFixture(t)
	rec := f.publish(t, "kit", "1.0.0", lifecycleFile("README.md", "old"))
	f.install(t, "kit")
	// Two wire-only recipes share Claude's mcpServers object.
	source := string(mustRead(t, filepath.Join(f.root, "recipes", "fix-mcp-two.yaml")))
	for _, name := range []string{"legacy-a", "legacy-b"} {
		writeLifecycle(t, filepath.Join(f.root, "recipes", name+".yaml"), strings.ReplaceAll(source, "fix-mcp-two", name))
		if _, _, err := runInstall(t, name, "--target", "claude", "--global", "--deploy"); err != nil {
			t.Fatal(err)
		}
		updated := strings.ReplaceAll(strings.ReplaceAll(source, "fix-mcp-two", name), "version: 1.0.0", "version: 2.0.0")
		updated = strings.ReplaceAll(updated, "https://fixture.invalid/mcp/", "https://fixture.invalid/"+name+"/v2")
		writeLifecycle(t, filepath.Join(f.root, "recipes", name+".yaml"), updated)
	}
	rec.Version = "2.0.0"
	f.saveRecipe(t, rec)
	if _, _, err := runUpdate(t, "kit", "legacy-a", "legacy-b", "--deploy"); err != nil {
		t.Fatal(err)
	}
	config := string(mustRead(t, filepath.Join(f.home, ".claude.json")))
	for _, name := range []string{"legacy-a", "legacy-b"} {
		if !strings.Contains(config, "https://fixture.invalid/"+name+"/v2") {
			t.Fatalf("lost contribution %s: %s", name, config)
		}
	}
	f.assertPackage(t, rec, lifecycleFile("README.md", "old"))
}

func lifecycleStatus(t *testing.T, name, want string) {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	before := lifecycleSnapshot(t, filepath.Join(home, ".patronus"))
	oldJSON := jsonOutput
	jsonOutput = true
	defer func() { jsonOutput = oldJSON }()
	out, _, err := runScan(t)
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct {
		Packages []scan.PackageStatus `json:"packages"`
	}
	if err := json.Unmarshal([]byte(out), &inventory); err != nil {
		t.Fatal(err)
	}
	if len(inventory.Packages) != 1 || inventory.Packages[0].Recipe != name || inventory.Packages[0].Status != want {
		t.Fatalf("scan status: %s", out)
	}
	lifecycleEqual(t, before, lifecycleSnapshot(t, filepath.Join(home, ".patronus")))
}

func TestDirectoryMultiRecipeInvalidPin(t *testing.T) {
	f := newLifecycleFixture(t)
	f.publish(t, "aa-kit", "1.0.0", lifecycleFile("README.md", "old"))
	f.install(t, "aa-kit")
	f.publish(t, "aa-kit", "2.0.0", lifecycleFile("README.md", "new"))
	f.publish(t, "zz-kit", "1.0.0", lifecycleFile("README.md", "b"))
	f.profile(t, "aa-kit", "zz-kit", "fix-bin")
	pin := &lock.Lock{Version: lock.Version, Entries: []lock.Entry{{Name: "zz-kit", Kind: "recipe", Version: "1.0.0", SHA256: strings.Repeat("b", 64)}}}
	if err := lock.Save(filepath.Join(f.root, "patronus.lock"), pin); err != nil {
		t.Fatal(err)
	}
	before := lifecycleSnapshot(t, filepath.Join(f.home, ".patronus"))
	if _, _, err := runInstall(t, "--profile", "directory-test", "--deploy"); err == nil {
		t.Fatal("incomplete directory pin accepted")
	}
	lifecycleEqual(t, before, lifecycleSnapshot(t, filepath.Join(f.home, ".patronus")))
}

func TestDirectoryUpdateRecoversSameVersionAfterReceiptCrash(t *testing.T) {
	f := newLifecycleFixture(t)
	f.publish(t, "kit", "1.0.0", lifecycleFile("README.md", "old"))
	if _, _, err := runInstall(t, "kit", "--deploy"); err != nil {
		t.Fatal(err)
	}
	rec := f.publish(t, "kit", "2.0.0", lifecycleFile("README.md", "new"))
	f.crash(t, "update-receipt")
	tx, err := packagestate.ReadTransaction(f.home, "kit")
	if err != nil || tx == nil || tx.Phase != packagestate.PackagePlaced {
		t.Fatalf("precommit journal: %+v %v", tx, err)
	}
	if got := lifecycleReceipt(t, f.home, "kit").RecipeVersion; got != "2.0.0" {
		t.Fatalf("candidate receipt version = %s", got)
	}
	before := lifecycleSnapshot(t, filepath.Join(f.home, ".patronus"))
	if out, _, err := runUpdate(t, "kit"); err != nil || !strings.Contains(out, "pending recovery") {
		t.Fatalf("read-only update: %s %v", out, err)
	}
	lifecycleEqual(t, before, lifecycleSnapshot(t, filepath.Join(f.home, ".patronus")))
	recovered := false
	lifecycleFault(t, func(point string) error {
		if point == "after-prepared" {
			// A new replacement can only start after rollback restored the previous receipt/tree.
			if got := lifecycleReceipt(t, f.home, "kit").RecipeVersion; got != "1.0.0" {
				t.Fatalf("recovery did not restore old receipt: %s", got)
			}
			if got := string(mustRead(t, filepath.Join(f.home, ".patronus", "packages", "kit", "README.md"))); got != "old" {
				t.Fatalf("recovery did not restore old tree: %s", got)
			}
			recovered = true
		}
		return nil
	})
	if _, _, err := runUpdate(t, "kit", "--deploy"); err != nil {
		t.Fatal(err)
	}
	if !recovered {
		t.Fatal("same-version update skipped locked recovery and replacement")
	}
	f.assertPackage(t, rec, lifecycleFile("README.md", "new"))
}
