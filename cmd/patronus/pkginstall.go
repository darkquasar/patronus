package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
)

// lookPathFunc mirrors exec.LookPath (injected so tests need no real binaries on
// PATH). Production wires exec.LookPath.
type lookPathFunc func(bin string) (string, error)

// firstPresentCandidate returns the first candidate whose manager is on PATH, in
// list order (the list is pre-sorted by preference upstream). preferred reports
// whether the chosen candidate was the FIRST (most-preferred) in the list — false
// means we fell back to a lower-preference manager (the caller warns). ok is false
// when no candidate's manager is present.
func firstPresentCandidate(cands []diff.InstallCandidateSpec, look lookPathFunc) (chosen diff.InstallCandidateSpec, preferred, ok bool) {
	for i, c := range cands {
		if _, err := look(c.Manager); err == nil {
			return c, i == 0, true
		}
	}
	return diff.InstallCandidateSpec{}, false, false
}

// installConsent decides whether Patronus runs a package install rather than just
// surfacing it. It threads the flag state and a PATH probe into runExecs.
type installConsent struct {
	allow bool          // --allow-package-installs: run non-interactively, all-or-nothing
	yes   bool          // --yes: non-interactive; skip prompts (do NOT install)
	look  lookPathFunc  // PATH probe (exec.LookPath in prod)
	in    *bufio.Reader // interactive reader (nil unless prompting)
	out   io.Writer
}

// readinessRow reports, per package-install artifact, which candidate managers are
// present/missing and whether at least one is available.
type readinessRow struct {
	Artifact    string
	Present     []string
	Missing     []string
	Satisfiable bool
}

// readinessReport inspects every package-install advisory EXEC in the change set
// and reports, per artifact, which candidate managers are present/missing and
// whether at least one is available (Satisfiable). This is the always-on Layer-1
// value: it tells the user what they'd need even when nothing is executed.
func readinessReport(cs *diff.ChangeSet, look lookPathFunc) []readinessRow {
	var rows []readinessRow
	for _, d := range cs.Diffs {
		if d.Action != diff.Exec || d.Exec == nil || len(d.Exec.Candidates) == 0 {
			continue
		}
		row := readinessRow{Artifact: d.Artifact}
		for _, c := range d.Exec.Candidates {
			if _, err := look(c.Manager); err == nil {
				row.Present = append(row.Present, c.Manager)
			} else {
				row.Missing = append(row.Missing, c.Manager)
			}
		}
		row.Satisfiable = len(row.Present) > 0
		rows = append(rows, row)
	}
	return rows
}

// printReadiness renders the readiness report, one line per package-install item.
func printReadiness(out io.Writer, rows []readinessRow) {
	if len(rows) == 0 {
		return
	}
	fmt.Fprintln(out, "\nPackage-install readiness:")
	for _, r := range rows {
		mark := "✓"
		if !r.Satisfiable {
			mark = "✗ MISSING"
		}
		fmt.Fprintf(out, "  %s  %s  (present: %v, missing: %v)\n", mark, r.Artifact, r.Present, r.Missing)
	}
}

// preflightAllOrNothing (only under --allow-package-installs) errors if ANY
// package-install item has no candidate manager on PATH — we never deploy a
// subset in the trust-me path. Returns nil when every item is satisfiable.
func preflightAllOrNothing(cs *diff.ChangeSet, look lookPathFunc) error {
	var unsat []string
	for _, d := range cs.Diffs {
		if d.Action != diff.Exec || d.Exec == nil || len(d.Exec.Candidates) == 0 {
			continue
		}
		if _, _, ok := firstPresentCandidate(d.Exec.Candidates, look); !ok {
			unsat = append(unsat, d.Artifact)
		}
	}
	if len(unsat) > 0 {
		return fmt.Errorf("--allow-package-installs: no package manager available for: %v (install one, or drop the flag to be prompted)", unsat)
	}
	return nil
}

// promptInstall mirrors conflictPrompt: interactive y/N, default No.
func promptInstall(c installConsent, artifact string, chosen diff.InstallCandidateSpec) bool {
	fmt.Fprintf(c.out, "Install %s via `%s`? [y/N]: ", artifact, chosen.Command)
	line, _ := c.in.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
