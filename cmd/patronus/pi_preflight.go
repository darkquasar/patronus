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

// preflightPiDeploy is the consent step for a descriptor that models
// mixed-context consent: it prints the plan warnings recorded for the target,
// confirms every review interactively, binds the confirmed operation to the
// bytes it displayed and reports the planned placements.
func preflightPiDeploy(cmd *cobra.Command, cs *diff.ChangeSet, d harness.Descriptor, reviews []piContextReview, opts *deployOptions) error {
	for _, fd := range cs.Diffs {
		if fd.Tool == d.Name && fd.Warning != "" {
			fmt.Fprintln(cmd.OutOrStdout(), fd.Warning)
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
		for _, fd := range cs.Diffs {
			if fd.Path == review.Consent.Path {
				if err := validatePiConsentBytes(review.Consent, fd.Path, fd.Before, fd.After); err != nil {
					return err
				}
			}
		}
	}
	for _, fd := range cs.Diffs {
		if fd.Tool == d.Name {
			fmt.Fprintf(cmd.OutOrStdout(), "Pi planned placement, runtime-unverified: %s scope=%s; eligible after apply at next startup/reload (static apply does not activate packages).\n", fd.Path, fd.Scope)
		}
	}
	return nil
}
