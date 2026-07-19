package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/registry"
)

// artifactsDir is the tree whose items carry a version: that must move on a content
// change. The check scopes the diff to it so a change elsewhere in the repo is never
// a false positive.
const _artifactsDir = "artifacts"

// manifestFile is the per-artifact manifest. It is the one file that is NOT content:
// the version: it carries is the bump we are checking FOR, and a canonical re-marshal
// of it must not, by itself, demand a bump.
const _manifestFile = "patronus.yaml"

// artifactChange is the reconciled state of one artifact directory across a PR diff.
// gatherChanges fills it from `git diff` (ContentChanged) and the version: line read
// on each side (BaseVersion at the merge-base, HeadVersion in the working tree).
type artifactChange struct {
	// Name is the artifact directory's relative path, used to name it in a failure.
	Name string
	// ContentChanged reports whether any file OTHER than patronus.yaml changed.
	// patronus.yaml is never counted as content: a canonical re-marshal or a
	// version-only edit must not, by itself, demand a bump.
	ContentChanged bool
	// ExistedInBase is false for an artifact added in this PR — a new artifact has
	// no baseline version to compare against, so it can never violate the rule.
	ExistedInBase bool
	// BaseVersion is the version: at the merge-base (empty when unreadable/absent).
	BaseVersion string
	// HeadVersion is the version: in the working tree (empty when absent).
	HeadVersion string
}

// violation is one artifact that changed content without bumping its version.
type violation struct {
	Name    string
	Version string // the un-bumped version, shown so the author sees what to move past
}

func (v violation) String() string {
	return fmt.Sprintf("%s: content changed but version: is still %s — bump it", v.Name, v.Version)
}

// checkVersions returns the artifacts that changed content without a version bump. It
// is the pure heart of the guard — no git, no I/O — so it is table-driven testable.
// It preserves input order and returns nil when everything is clean.
//
// An artifact is a violation exactly when: it existed at the base, its content
// changed, and its version: is unchanged from base to head. Every other shape is
// allowed — a new artifact (no base to compare), a deleted one (no head content), a
// version-only change, a content change WITH a bump, or a no-op (git lists no changed
// content, so ContentChanged is false).
func checkVersions(changes []artifactChange) []violation {
	var out []violation
	for _, c := range changes {
		if !c.ContentChanged || !c.ExistedInBase {
			continue
		}
		if c.HeadVersion != c.BaseVersion {
			continue // bumped (or cleared) — the rule is satisfied
		}
		out = append(out, violation{Name: c.Name, Version: c.HeadVersion})
	}
	return out
}

// newCheckVersionsCmd is the PR-side guard for pat-3mz5: it fails when an artifact's
// CONTENT changed against the PR base but its patronus.yaml version: did not. The
// rule lives in CONTRIBUTING.md ("bump version: on any content change") but was
// unenforced until this — PR #26 shipped 8 un-bumped content changes that only
// surfaced post-merge, when publish-catalog hit R2's write-once key. This catches it
// at review instead.
//
// The real decision is checkVersions (pure, table-driven); this wrapper only gathers
// its inputs from git. checkVersions lives in this package, not internal/, because it
// has exactly one caller — mirroring build.go's own in-package helpers.
func newCheckVersionsCmd() *cobra.Command {
	var base string
	cmd := &cobra.Command{
		Use:   "check-versions",
		Short: "Fail when an artifact's content changed without a version: bump (PR/CI guard)",
		Long: "Compares the working tree against --base (a merge-base ref, e.g. origin/main) and\n" +
			"fails when any artifacts/<...>/ item has a changed content file but an unchanged\n" +
			"version: in its patronus.yaml. Enforces CONTRIBUTING.md's version-bump rule at PR\n" +
			"review, before the catalog deploy hits R2's write-once keys. Needs full history\n" +
			"(fetch-depth: 0) so the base ref is present.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := registry.DiscoverRoot(wd)
			if err != nil {
				return err
			}
			violations := checkVersions(gatherChanges(cmd.Context(), root, base))
			if len(violations) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "version-bump check: ok")
				return nil
			}
			for _, v := range violations {
				fmt.Fprintln(cmd.ErrOrStderr(), v.String())
			}
			return fmt.Errorf("%d artifact(s) changed content without a version bump", len(violations))
		},
	}
	cmd.Flags().StringVar(&base, "base", "origin/main", "git ref to diff against (the PR base / merge-base)")
	return cmd
}

// gatherChanges walks the artifact dirs touched between the merge-base of base..HEAD
// and the working tree, and reconciles each into an artifactChange. It shells out to
// git rather than importing a git library — the same dependency-light choice the rest
// of cmd/patronus makes, and the diff is trivial plumbing.
func gatherChanges(ctx context.Context, root, base string) []artifactChange {
	// Diff the merge-base (base...HEAD) against the working tree, so commits on base
	// after the branch forked don't count as this PR's changes. A missing base ref /
	// no merge-base is a CI-setup shape, not a content violation: degrade to "nothing
	// to check" rather than a false red, the same way the drift guard stays silent
	// when it cannot know. The CI job sets fetch-depth: 0 precisely so this does not
	// happen for a real PR.
	names, err := gitDiffNames(ctx, root, base)
	if err != nil {
		return nil
	}

	dirs := artifactDirs(names)
	changes := make([]artifactChange, 0, len(dirs))
	for _, dir := range dirs {
		c := artifactChange{Name: dir}
		// Content changed if ANY touched file under this dir is not patronus.yaml.
		for _, n := range names {
			if artifactDirOf(n) == dir && filepath.Base(n) != _manifestFile {
				c.ContentChanged = true
				break
			}
		}
		manifestPath := filepath.Join(dir, _manifestFile)
		if baseData, ok := gitShow(ctx, root, base, manifestPath); ok {
			c.ExistedInBase = true
			c.BaseVersion = versionLine(baseData)
		}
		if headData, err := os.ReadFile(filepath.Join(root, manifestPath)); err == nil {
			c.HeadVersion = versionLine(headData)
		}
		changes = append(changes, c)
	}
	return changes
}

// gitDiffNames returns the repo-relative paths under artifacts/ that differ between
// the merge-base of base..HEAD and the working tree.
func gitDiffNames(ctx context.Context, root, base string) ([]string, error) {
	// We resolve the merge-base explicitly and `git diff <mergeBase>` against the
	// working tree: a two-dot range would drop uncommitted edits, and three-dot diff
	// syntax does not include the working tree.
	mb, err := runGit(ctx, root, "merge-base", base, "HEAD")
	if err != nil {
		return nil, err
	}
	out, err := runGit(ctx, root, "diff", "--name-only", strings.TrimSpace(mb), "--", _artifactsDir)
	if err != nil {
		return nil, err
	}
	var names []string
	sc := bufio.NewScanner(bytes.NewReader([]byte(out)))
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			names = append(names, line)
		}
	}
	return names, sc.Err()
}

// gitShow returns the bytes of path at ref, and whether it exists there. A file
// absent at the base (a new artifact) returns ok=false — no baseline to compare.
func gitShow(ctx context.Context, root, ref, path string) ([]byte, bool) {
	out, err := runGitRaw(ctx, root, "show", ref+":"+path)
	if err != nil {
		return nil, false
	}
	return out, true
}

// artifactDirs returns the distinct artifact directories among the changed paths,
// sorted, so the report is deterministic.
func artifactDirs(names []string) []string {
	seen := map[string]bool{}
	for _, n := range names {
		if d := artifactDirOf(n); d != "" {
			seen[d] = true
		}
	}
	out := make([]string, 0, len(seen))
	for d := range seen {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// artifactDirOf maps a changed path to its artifact directory —
// artifacts/<type>/<name> — or "" when the path is not that deep (e.g. a stray file
// directly under artifacts/). Every artifact is exactly two levels below artifacts/.
func artifactDirOf(name string) string {
	parts := strings.Split(filepath.ToSlash(name), "/")
	if len(parts) < 4 || parts[0] != _artifactsDir {
		return ""
	}
	return filepath.Join(parts[0], parts[1], parts[2])
}

// versionLine extracts the value of the top-level `version:` key from a patronus.yaml
// without a full YAML parse — the manifest may be unreadable at the base for reasons
// that must not fail the check, and we only ever need this one scalar. It returns ""
// when no version: line is present (the field is omitempty).
func versionLine(data []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "version:") {
			continue
		}
		v := strings.TrimSpace(strings.TrimPrefix(line, "version:"))
		v = strings.Trim(v, `"'`)
		return v
	}
	return ""
}

func runGit(ctx context.Context, root string, args ...string) (string, error) {
	out, err := runGitRaw(ctx, root, args...)
	return string(out), err
}

func runGitRaw(ctx context.Context, root string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), stderr.String(), err)
	}
	return stdout.Bytes(), nil
}
