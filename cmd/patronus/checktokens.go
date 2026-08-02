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

// knownSkillTokens are the two body tokens the skill transform substitutes. A
// {skill…}-shaped string that is not one of these ships as a literal and fails at
// runtime, which is what this guard exists to catch.
var knownSkillTokens = map[string]bool{
	"{skillDir}":  true,
	"{skillsDir}": true,
}

// skillTokenPattern matches a near-miss of the two real tokens, case-insensitively:
// a brace-wrapped word that starts with "ski" AND ends in "dir(s)". Both halves are
// load-bearing.
//
// The prefix is "ski" rather than "skill" because the likeliest typos drop a letter
// ({skilDir}, {skilsDir}), which a "skill"-anchored pattern would miss. The "dir"
// suffix is what keeps the prefix from over-reaching: an ordinary variable that
// merely begins with those letters — {skip_count} in an f-string, {skimmed} — is
// not a botched skill token and must not red the build.
//
// The cost of this shape is a typo mangling BOTH halves ({skilPath}) going
// unreported. That is the right trade: a guard that cries wolf on ordinary code
// gets disabled, and this one stays credible.
var skillTokenPattern = regexp.MustCompile(`\{[Ss][Kk][Ii][A-Za-z0-9_]*[Dd][Ii][Rr][Ss]?\}`)

// badToken is one malformed token occurrence in one artifact file.
type badToken struct {
	File  string // repo-relative path
	Line  int    // 1-indexed
	Token string // the offending {…} string, verbatim
}

func (b badToken) String() string {
	return fmt.Sprintf("%s:%d: %s is not a known skill token (want {skillDir} or {skillsDir})", b.File, b.Line, b.Token)
}

// checkTokens returns every malformed skill token in content. It is the pure heart
// of the guard — no filesystem — so it is table-driven testable. Order follows the
// file, and it returns nil when content is clean.
//
// Two shapes are deliberately NOT reported, because both are how real skill bodies
// legitimately spell braces:
//
//   - A shell or JS interpolation (`${skillName}`), which the reader's own runtime
//     expands. The preceding "$" is what distinguishes it from a Patronus token.
//   - Any brace-wrapped word that is not {skill…}-shaped ({installPath}, {n},
//     f-string and JSON braces). Substitution leaves those alone by design, and
//     they are none of this guard's business.
func checkTokens(file string, content []byte) []badToken {
	if !utf8.Valid(content) {
		return nil // a binary asset is copied untouched; it carries no tokens
	}
	var out []badToken
	for i, line := range strings.Split(string(content), "\n") {
		for _, loc := range skillTokenPattern.FindAllStringIndex(line, -1) {
			tok := line[loc[0]:loc[1]]
			if knownSkillTokens[tok] {
				continue
			}
			// `${…}` is an interpolation in the body's own language, not a
			// Patronus token, so the transform is right to leave it alone.
			if loc[0] > 0 && line[loc[0]-1] == '$' {
				continue
			}
			out = append(out, badToken{File: file, Line: i + 1, Token: tok})
		}
	}
	return out
}

// newCheckTokensCmd is the build guard for pat-twgb. Token substitution leaves
// unknown braces untouched by design (skill bodies carry JSON, f-strings, and
// shell expansions, so erroring on every unrecognised {…} would be unusable), which
// means a typo like {skilDir} ships silently as a literal and fails only when a
// user runs the skill. This narrows the check to {skill…}-shaped strings, where a
// non-token is always a mistake.
//
// It mirrors check-versions: a pure decision function (checkTokens) plus a thin
// wrapper that walks the tree. Both live in this package rather than internal/
// because each has exactly one caller.
func newCheckTokensCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-tokens",
		Short: "Fail when an artifact body carries a malformed {skill...} token (build guard)",
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
			bad, err := scanArtifactTokens(root)
			if err != nil {
				return err
			}
			if len(bad) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "skill-token check: ok")
				return nil
			}
			for _, b := range bad {
				fmt.Fprintln(cmd.ErrOrStderr(), b.String())
			}
			return fmt.Errorf("%d malformed skill token(s)", len(bad))
		},
	}
}

// scanArtifactTokens walks every regular file under artifacts/ and collects the
// malformed tokens, sorted by path so the report is deterministic.
func scanArtifactTokens(root string) ([]badToken, error) {
	artifacts := filepath.Join(root, _artifactsDir)
	var out []badToken
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
		out = append(out, checkTokens(filepath.ToSlash(rel), content)...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan artifact tokens: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}
