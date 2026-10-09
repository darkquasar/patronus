package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
	"github.com/spf13/cobra"
)

// piContextConsent is operation-local, never a persisted grant. DP-05 rechecks
// the prior hash under its mutation lock; this carries the prepared bytes only.
type piContextConsent struct {
	Path, PiSource, ClaudeSource string
	PriorHash, ResultHash        [32]byte
}
type piContextReview struct {
	Consent     piContextConsent
	Proposed    []byte
	OtherOwners []string
}

// piPreflightPlan is Pi admission over an already computed shared ChangeSet:
// the shared rules run in harness.Admit with the Pi descriptor, Pi-only rules
// in piValidator, and mixed-context reviews return through Admission.Review
// as piContextReview values for the deploy consent step.
func piPreflightPlan(reg *harness.Registry, cs *diff.ChangeSet, res toolpath.Resolver, home, project string) ([]piContextReview, error) {
	d, ok := reg.Lookup("pi")
	if !ok {
		return nil, harness.ErrUnknownTarget
	}
	var reviews []piContextReview
	in := &harness.Admission{
		Changes:  cs,
		Resolver: res,
		Home:     home,
		Project:  project,
		Env:      os.LookupEnv,
		Read:     scan.ReadPiFile,
		LoadOwners: func() ([]state.Item, error) {
			return loadOwners([]string{filepath.Join(home, ".patronus/state.json"), filepath.Join(project, ".patronus/state.json")}, nil, nil)
		},
		Warn: func(message string) { appendPlacementWarning(cs, d.Name, message) },
		Review: func(r harness.ContextReview) error {
			claudeSource, err := piClaudeContextSources(res, project, r.Scope)
			if err != nil {
				return err
			}
			piSource := r.Path
			if len(r.Sources) > 0 {
				piSource = r.Sources[0]
			}
			reviews = append(reviews, piContextReview{Consent: piContextConsent{Path: r.Path, PiSource: piSource, ClaudeSource: claudeSource, PriorHash: sha256.Sum256(r.Prior), ResultHash: sha256.Sum256(r.Result)}, Proposed: append([]byte(nil), r.Result...), OtherOwners: r.OtherOwners})
			return nil
		},
	}
	if err := harness.Admit(d, in); err != nil {
		return nil, err
	}
	return reviews, nil
}

// appendPlacementWarning records message once on the first diff of tool, the
// channel the plan renderer and the deploy preview already print.
func appendPlacementWarning(cs *diff.ChangeSet, tool, message string) {
	for i := range cs.Diffs {
		if cs.Diffs[i].Tool != tool {
			continue
		}
		if !contains(strings.Split(cs.Diffs[i].Warning, "\n"), message) {
			if cs.Diffs[i].Warning != "" {
				cs.Diffs[i].Warning += "\n"
			}
			cs.Diffs[i].Warning += message
		}
		return
	}
}

// Report the qualified canonical Claude sources independently. A CLAUDE name
// under the Pi global root is only a mixed-content signal, not Claude discovery.
func piClaudeContextSources(res toolpath.Resolver, project, scope string) (string, error) {
	if value := os.Getenv("CLAUDE_CONFIG_DIR"); value != "" {
		return "", fmt.Errorf("pi mixed-context: active CLAUDE_CONFIG_DIR override makes Claude provenance uncertain; qualify it before apply")
	}
	global := res.ResolveMarker("~/.claude/CLAUDE.md", "claude", "global")
	candidates := []string{global}
	if scope != "global" {
		for dir := project; ; dir = filepath.Dir(dir) {
			candidates = append(candidates, filepath.Join(dir, "CLAUDE.md"))
			// Additional native Claude source spellings are reported rather than
			// collapsed into the Pi candidate. No precedence rewrite is attempted.
			candidates = append(candidates, filepath.Join(dir, ".claude/CLAUDE.md"), filepath.Join(dir, "CLAUDE.local.md"))
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	var found []string
	for _, path := range candidates {
		_, exists, err := scan.ReadPiFile(path)
		if err != nil {
			return "", fmt.Errorf("pi mixed-context Claude source %s: %w", path, err)
		}
		if exists {
			found = append(found, path)
		}
	}
	if len(found) == 0 {
		path := global
		if scope != "global" {
			path = filepath.Join(project, "CLAUDE.md")
		}
		return path + " (absent; no effective canonical Claude source found)", nil
	}
	return strings.Join(found, "; "), nil
}

func confirmPiContexts(reviews []piContextReview, interactive, yes, force bool, in io.Reader, out io.Writer) ([]piContextConsent, error) {
	if len(reviews) == 0 {
		return nil, nil
	}
	if !interactive || yes || force {
		return nil, fmt.Errorf("pi mixed-context conflict: only a separate interactive confirmation of an operator-prepared combined file can consent; --yes/--force/headless cannot")
	}
	reader := bufio.NewReader(in)
	var consents []piContextConsent
	for _, review := range reviews {
		c := review.Consent
		fmt.Fprintf(out, "\nPi mixed-context migration — OPERATOR-PREPARED combined file only\nCanonical path: %s\nPi effective source: %s\nClaude effective source: %s\nPrepared-prior SHA256: %x\nResult SHA256: %x\nProposed Pi section / prepared result:\n%s\n", c.Path, c.PiSource, c.ClaudeSource, c.PriorHash, c.ResultHash, review.Proposed)
		fmt.Fprintln(out, "Mixed-content signal: CLAUDE filename, foreign Patronus section, or other-target ownership; filename alone does not establish Claude loading.")
		for _, owner := range review.OtherOwners {
			fmt.Fprintln(out, "Other owner:", owner)
		}
		fmt.Fprintln(out, "Original pre-migration bytes and grant must already be captured separately. Patronus does not synthesize combined instructions; removal will not undo that migration.")
		fmt.Fprint(out, "Confirm you prepared and reviewed complete shared and harness-specific instructions; type 'prepared combined file': ")
		answer, err := reader.ReadString('\n')
		if err != nil || strings.TrimSpace(answer) != "prepared combined file" {
			return nil, fmt.Errorf("pi mixed-context consent declined; whole selection refused")
		}
		consents = append(consents, c)
	}
	return consents, nil
}

func piInteractiveInput(in io.Reader) bool {
	// Fail closed on non-terminal and platforms without a verifiable terminal
	// descriptor. /dev/null is a character device, but is not interactive consent.
	file, ok := in.(*os.File)
	if !ok {
		return false
	}
	target, err := os.Readlink("/proc/self/fd/" + strconv.FormatUint(uint64(file.Fd()), 10))
	if err != nil {
		return false
	}
	return strings.HasPrefix(target, "/dev/pts/") || target == "/dev/tty" || strings.HasPrefix(target, "/dev/tty")
}

// validatePiConsentBytes is also the DP-05 reread seam: neither an ordinary
// overwrite answer nor equality of newly computed bytes can renew old consent.
func validatePiConsentBytes(c piContextConsent, path string, prior, result []byte) error {
	if path != c.Path || sha256.Sum256(prior) != c.PriorHash || sha256.Sum256(result) != c.ResultHash {
		return fmt.Errorf("pi mixed-context consent stale for %s; new interactive preview required", path)
	}
	return nil
}

func preflightPiDeploy(cmd *cobra.Command, cs *diff.ChangeSet, res toolpath.Resolver, opts *deployOptions) error {
	reviews, err := piPreflightPlan(opts.Harness, cs, res, opts.home, opts.projectDir)
	if err != nil {
		return err
	}
	for _, d := range cs.Diffs {
		if d.Tool == "pi" && d.Warning != "" {
			fmt.Fprintln(cmd.OutOrStdout(), d.Warning)
		}
	}
	consents, err := confirmPiContexts(reviews, piInteractiveInput(cmd.InOrStdin()), opts.yes, opts.force, cmd.InOrStdin(), cmd.OutOrStdout())
	if err != nil {
		return err
	}
	opts.consent = piConsentSet(consents)
	for _, review := range reviews {
		// Bind the confirmed operation to the bytes it actually displayed, not a
		// synthesis or an unrelated confirmation. Last-moment rereads belong DP-05.
		for _, d := range cs.Diffs {
			if d.Path == review.Consent.Path {
				if err := validatePiConsentBytes(review.Consent, d.Path, d.Before, d.After); err != nil {
					return err
				}
			}
		}
	}
	for _, d := range cs.Diffs {
		if d.Tool == "pi" {
			fmt.Fprintf(cmd.OutOrStdout(), "Pi planned placement, runtime-unverified: %s scope=%s; eligible after apply at next startup/reload (static apply does not activate packages).\n", d.Path, d.Scope)
		}
	}
	return nil
}
