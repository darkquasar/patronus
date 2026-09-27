package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func packageGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	s, err := runGit(context.Background(), root, args...)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(s)
}
func packageGitFixture(t *testing.T) string {
	t.Helper()
	root := packageFixture(t)
	packageGit(t, root, "branch", "base")
	return root
}

func TestPackageVersionSixteenComponentPayload(t *testing.T) {
	root := packageFixture(t)
	path := strings.Repeat("dir/", 15) + "spec.yaml"
	packageWrite(t, root, "packages/kit/package.yaml", strings.ReplaceAll(testPackageDescriptor, "spec.yaml", path))
	packageWrite(t, root, "packages/kit/"+path, "deep payload")
	packageGit(t, root, "add", ".")
	packageGit(t, root, "commit", "-m", "deep payload")
	packageGit(t, root, "branch", "base")
	if err := checkPackageVersions(context.Background(), root, "base"); err != nil {
		t.Fatal(err)
	}
	packageWrite(t, root, "packages/kit/"+path, "changed payload")
	if err := checkPackageVersions(context.Background(), root, "base"); err == nil || !strings.Contains(err.Error(), "without changing version") {
		t.Fatalf("expected version violation, got %v", err)
	}
}
func TestPackageVersionChangedInputs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*testing.T, string)
	}{
		{"payload", func(t *testing.T, r string) { packageWrite(t, r, "packages/kit/spec.yaml", "changed") }},
		{"platform", func(t *testing.T, r string) {
			packageWrite(t, r, "packages/kit/package.yaml", strings.ReplaceAll(testPackageDescriptor, "darwin", "linux"))
		}},
		{"mode", func(t *testing.T, r string) {
			packageWrite(t, r, "packages/kit/package.yaml", strings.ReplaceAll(testPackageDescriptor, "false", "true"))
		}},
		{"deleted payload", func(t *testing.T, r string) {
			if err := os.Remove(filepath.Join(r, "packages/kit/spec.yaml")); err != nil {
				t.Fatal(err)
			}
		}},
		{"payload removed from descriptor", func(t *testing.T, r string) {
			packageWrite(t, r, "packages/kit/README.md", "readme")
			packageWrite(t, r, "packages/kit/package.yaml", strings.ReplaceAll(testPackageDescriptor, "spec.yaml", "README.md"))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := packageGitFixture(t)
			tc.change(t, root)
			if err := checkPackageVersions(context.Background(), root, "base"); err == nil {
				t.Fatal("accepted unchanged version")
			}
		})
	}
}
func TestPackageVersionAllowedChanges(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*testing.T, string)
	}{
		{"unchanged", func(*testing.T, string) {}},
		{"version only", func(t *testing.T, r string) {
			packageWrite(t, r, "packages/kit/package.yaml", strings.ReplaceAll(testPackageDescriptor, "1.0.0", "1.0.1"))
		}},
		{"version bump", func(t *testing.T, r string) {
			packageWrite(t, r, "packages/kit/spec.yaml", "changed")
			packageWrite(t, r, "packages/kit/package.yaml", strings.ReplaceAll(testPackageDescriptor, "1.0.0", "1.0.1"))
		}},
		{"test only", func(t *testing.T, r string) { packageWrite(t, r, "packages/kit/tests/test.txt", "changed") }},
		{"deleted package", func(t *testing.T, r string) {
			if err := os.RemoveAll(filepath.Join(r, "packages/kit")); err != nil {
				t.Fatal(err)
			}
		}},
		{"new package", func(t *testing.T, r string) {
			packageWrite(t, r, "packages/new/package.yaml", strings.ReplaceAll(testPackageDescriptor, "name: kit", "name: new"))
			packageWrite(t, r, "packages/new/spec.yaml", "new")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := packageGitFixture(t)
			tc.change(t, root)
			if err := checkPackageVersions(context.Background(), root, "base"); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestPackageVersionUsesMergeBase(t *testing.T) {
	root := packageGitFixture(t)
	packageGit(t, root, "checkout", "-b", "feature")
	packageGit(t, root, "checkout", "main")
	packageWrite(t, root, "packages/kit/spec.yaml", "main changed")
	packageGit(t, root, "add", ".")
	packageGit(t, root, "commit", "-m", "main changed")
	packageGit(t, root, "checkout", "feature")
	if err := checkPackageVersions(context.Background(), root, "main"); err != nil {
		t.Fatal(err)
	}
	if err := checkPackageVersions(context.Background(), root, "nonexistent"); err == nil {
		t.Fatal("unreadable merge base passed")
	}
}

func TestPackageVersionCommittedChangeAndCLI(t *testing.T) {
	root := packageGitFixture(t)
	packageWrite(t, root, "packages/kit/spec.yaml", "committed change")
	packageGit(t, root, "add", ".")
	packageGit(t, root, "commit", "-m", "changed")
	if err := checkPackageVersions(context.Background(), root, "base"); err == nil {
		t.Fatal("committed change passed")
	}
	t.Chdir(root)
	cmd := newCheckVersionsCmd()
	cmd.SetArgs([]string{"--base", "missing"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("CLI discarded package version check error")
	}
}
