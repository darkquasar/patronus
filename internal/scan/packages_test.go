package scan

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagestate"
)

func packageReceipt(t *testing.T) (string, *packagestate.Receipt) {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(home, ".patronus", "packages", "kit")
	mustMkdir(t, root)
	mustWrite(t, filepath.Join(root, "file"))
	r := &packagestate.Receipt{SchemaVersion: 1, Recipe: "kit", RecipeVersion: "1.0.0", Root: root, URL: "https://example.test/kit.tar.gz", ArchiveSHA256: fmt.Sprintf("sha256:%064d", 0), Identity: packagebundle.Identity{Name: "kit", Version: "1.0.0", OS: "linux", Arch: "amd64"}, Files: []packagebundle.Entry{{Path: "file", Mode: 0644, SHA256: "sha256:2d711642b726b04401627ca9fbac32f5c8530fb1903cc4db02258717921a4881"}}, Directories: []string{"."}}
	if err := packagestate.Save(home, r); err != nil {
		t.Fatal(err)
	}
	return home, r
}
func TestPackagesReceiptWithoutReference(t *testing.T) {
	home, r := packageReceipt(t)
	inv, err := Scan(Options{ProjectDir: home, Env: envFrom(map[string]string{"HOME": home})})
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Packages) != 1 || inv.Packages[0].Status != "incomplete" || inv.Packages[0].Recipe != r.Recipe {
		t.Fatalf("inventory: %+v", inv)
	}
}
func TestPackagesPendingWithoutReceiptReadOnly(t *testing.T) {
	home, r := packageReceipt(t)
	if err := packagestate.DeleteReceipt(home, r.Recipe); err != nil {
		t.Fatal(err)
	}
	tx := &packagestate.Transaction{SchemaVersion: 1, Recipe: r.Recipe, Root: r.Root, Operation: "metadata", Phase: packagestate.Prepared, Candidate: r}
	if err := packagestate.WriteTransaction(home, tx); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(home, ".patronus", "package-state", "transactions", "kit", "transaction.json"))
	if err != nil {
		t.Fatal(err)
	}
	packages, err := Packages(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 1 || packages[0].Status != "recovery-required" {
		t.Fatalf("packages: %+v", packages)
	}
	after, err := os.ReadFile(filepath.Join(home, ".patronus", "package-state", "transactions", "kit", "transaction.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("read-only scan changed journal")
	}
}

func TestPackagesReportsInstalledModifiedAndMissing(t *testing.T) {
	home, r := packageReceipt(t)
	metadata, err := json.Marshal(packagebundle.Metadata{SchemaVersion: 1, Identity: r.Identity, Files: r.Files})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Root, "package.json"), metadata, 0644); err != nil {
		t.Fatal(err)
	}
	r.Files = append(r.Files, packagebundle.Entry{Path: "package.json", Mode: 0644, SHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(metadata))})
	if err := packagestate.Save(home, r); err != nil {
		t.Fatal(err)
	}
	packages, err := Packages(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 1 || packages[0].Status != "installed" {
		t.Fatalf("packages: %+v", packages)
	}
	if err := os.WriteFile(filepath.Join(r.Root, "file"), []byte("edit"), 0644); err != nil {
		t.Fatal(err)
	}
	packages, err = Packages(home)
	if err != nil {
		t.Fatal(err)
	}
	if packages[0].Status != "modified" || !slices.Contains(packages[0].Paths, "file") {
		t.Fatalf("packages: %+v", packages)
	}
	if err := os.Remove(filepath.Join(r.Root, "file")); err != nil {
		t.Fatal(err)
	}
	packages, err = Packages(home)
	if err != nil {
		t.Fatal(err)
	}
	if packages[0].Status != "incomplete" || !slices.Contains(packages[0].Paths, "file") {
		t.Fatalf("packages: %+v", packages)
	}
}

func TestPackagesReducedReceiptWithMetadataIsIncomplete(t *testing.T) {
	home, r := packageReceipt(t)
	metadata, err := json.Marshal(packagebundle.Metadata{SchemaVersion: 1, Identity: r.Identity, Files: r.Files})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Root, "package.json"), metadata, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(r.Root, "file")); err != nil {
		t.Fatal(err)
	}
	r.Files = []packagebundle.Entry{{Path: "package.json", Mode: 0644, SHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(metadata))}}
	if err := packagestate.Save(home, r); err != nil {
		t.Fatal(err)
	}
	packages, err := Packages(home)
	if err != nil {
		t.Fatal(err)
	}
	if packages[0].Status != "incomplete" {
		t.Fatalf("partial ownership reported complete: %+v", packages)
	}
}

func TestPackagesSkipsRemovalAcknowledgedDuringDiscovery(t *testing.T) {
	home, receipt := packageReceipt(t)
	if err := packagestate.DeleteReceipt(home, receipt.Recipe); err != nil {
		t.Fatal(err)
	}
	tx := &packagestate.Transaction{
		SchemaVersion: 1, Recipe: receipt.Recipe, Root: receipt.Root,
		Operation: "remove", Phase: packagestate.Committed, Previous: receipt,
	}
	if err := packagestate.WriteTransaction(home, tx); err != nil {
		t.Fatal(err)
	}
	reads := 0
	packages, err := packagesWithTransactionReader(home, func(home, recipe string) (*packagestate.Transaction, error) {
		reads++
		if reads == 2 {
			// Discovery has recorded the journal-only identity. Acknowledge
			// removal before the status pass reads that journal again.
			if err := packagestate.ClearTransaction(home, recipe); err != nil {
				return nil, err
			}
		}
		return packagestate.ReadTransaction(home, recipe)
	})
	if err != nil {
		t.Fatal(err)
	}
	if reads != 2 {
		t.Fatalf("transaction reads = %d, want 2", reads)
	}
	if len(packages) != 0 {
		t.Fatalf("acknowledged removal remains discoverable: %+v", packages)
	}
}
