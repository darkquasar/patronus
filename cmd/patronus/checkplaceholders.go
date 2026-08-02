package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/registry"
)

// knownSkillPlaceholders are the two body placeholders the skill transform substitutes. A
// {skill…}-shaped string that is not one of these ships as a literal and fails at
// runtime, which is what this guard exists to catch.
var knownSkillPlaceholders = map[string]bool{
	"{skillDir}":  true,
	"{skillsDir}": true,
}

// skillPlaceholderPattern matches a near-miss of the two real placeholders, case-insensitively:
// a brace-wrapped word that starts with "ski" AND ends in "dir(s)". Both halves are
// load-bearing.
//
// The prefix is "ski" rather than "skill" because the likeliest typos drop a letter
// ({skilDir}, {skilsDir}), which a "skill"-anchored pattern would miss. The "dir"
// suffix is what keeps the prefix from over-reaching: an ordinary variable that
// merely begins with those letters — {skip_count} in an f-string, {skimmed} — is
// not a botched skill placeholder and must not red the build.
//
// The cost of this shape is a typo mangling BOTH halves ({skilPath}) going
// unreported. That is the right trade: a guard that cries wolf on ordinary code
// gets disabled, and this one stays credible.
var skillPlaceholderPattern = regexp.MustCompile(`\{[Ss][Kk][Ii][A-Za-z0-9_]*[Dd][Ii][Rr][Ss]?\}`)

// badPlaceholder is one malformed placeholder occurrence in one artifact file.
type badPlaceholder struct {
	File string // repo-relative path
	Line int    // 1-indexed
	Text string // the offending {…} string, verbatim
}

func (b badPlaceholder) String() string {
	return fmt.Sprintf("%s:%d: %s is not a known skill placeholder (want {skillDir} or {skillsDir})", b.File, b.Line, b.Text)
}

// checkPlaceholders returns every malformed skill placeholder in content. It is the pure heart
// of the guard — no filesystem — so it is table-driven testable. Order follows the
// file, and it returns nil when content is clean.
//
// Two shapes are deliberately NOT reported, because both are how real skill bodies
// legitimately spell braces:
//
//   - A shell or JS interpolation (`${skillName}`), which the reader's own runtime
//     expands. The preceding "$" is what distinguishes it from a Patronus placeholder.
//   - Any brace-wrapped word that is not {skill…}-shaped ({installPath}, {n},
//     f-string and JSON braces). Substitution leaves those alone by design, and
//     they are none of this guard's business.
func checkPlaceholders(file string, content []byte) []badPlaceholder {
	if !utf8.Valid(content) {
		return nil // a binary asset is copied untouched; it carries no placeholders
	}
	var out []badPlaceholder
	for i, line := range strings.Split(string(content), "\n") {
		for _, loc := range skillPlaceholderPattern.FindAllStringIndex(line, -1) {
			tok := line[loc[0]:loc[1]]
			if knownSkillPlaceholders[tok] {
				continue
			}
			// `${…}` is an interpolation in the body's own language, not a
			// Patronus placeholder, so the transform is right to leave it alone.
			if loc[0] > 0 && line[loc[0]-1] == '$' {
				continue
			}
			out = append(out, badPlaceholder{File: file, Line: i + 1, Text: tok})
		}
	}
	return out
}

// newCheckPlaceholdersCmd is the build guard for pat-twgb. Placeholder substitution leaves
// unknown braces untouched by design (skill bodies carry JSON, f-strings, and
// shell expansions, so erroring on every unrecognised {…} would be unusable), which
// means a typo like {skilDir} ships silently as a literal and fails only when a
// user runs the skill. This narrows the check to {skill…}-shaped strings, where a
// non-placeholder is always a mistake.
//
// It mirrors check-versions: a pure decision function (checkPlaceholders) plus a thin
// wrapper that walks the tree. Both live in this package rather than internal/
// because each has exactly one caller.
func newCheckPlaceholdersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-placeholders",
		Short: "Fail when an artifact body carries a malformed {skill...} placeholder (build guard)",
		Long: "Scans every file under artifacts/ for {skill...}-shaped strings that are not\n" +
			"exactly {skillDir} or {skillsDir}. Substitution leaves unknown braces alone by\n" +
			"design, so a typo would otherwise ship as a literal and fail at runtime. Shell\n" +
			"and JS interpolations (${skillName}) are not reported — they belong to the\n" +
			"body's own language.",
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
			bad, err := scanArtifactPlaceholders(root)
			if err != nil {
				return err
			}
			if len(bad) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "skill-placeholder check: ok")
				return nil
			}
			for _, b := range bad {
				fmt.Fprintln(cmd.ErrOrStderr(), b.String())
			}
			return fmt.Errorf("%d malformed skill placeholder(s)", len(bad))
		},
	}
}

// scanArtifactPlaceholders walks every regular file under artifacts/ and collects the
// malformed placeholders, sorted by path so the report is deterministic.
func scanArtifactPlaceholders(root string) ([]badPlaceholder, error) {
	artifacts := filepath.Join(root, _artifactsDir)
	var out []badPlaceholder
	err := filepath.WalkDir(artifacts, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		out = append(out, checkPlaceholders(filepath.ToSlash(rel), content)...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan artifact placeholders: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}
