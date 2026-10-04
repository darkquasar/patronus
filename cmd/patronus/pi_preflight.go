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

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
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

func piPreflightPlan(cs *diff.ChangeSet, res toolpath.Resolver, home, project string) ([]piContextReview, error) {
	var selected []diff.FileDiff
	agents, resources, mcp := false, false, false
	for _, d := range cs.Diffs {
		if d.Native != nil {
			continue
		}
		if contains(strings.Split(d.Tool, "+"), "pi") {
			if d.Tool != "pi" {
				return nil, fmt.Errorf("pi cross-target destination conflict %s (%s)", d.Path, d.Tool)
			}
			selected = append(selected, d)
			agents = agents || d.Type == "agent"
			resources = resources || d.Type == "agent" || d.Type == "skill" || d.Type == "command"
			mcp = mcp || filepath.Base(d.Path) == "mcp-adapter.json"
		}
	}
	if len(selected) == 0 {
		return nil, nil
	}
	warn := func(message string) {
		for i := range cs.Diffs {
			if cs.Diffs[i].Tool == "pi" {
				if !contains(strings.Split(cs.Diffs[i].Warning, "\n"), message) {
					if cs.Diffs[i].Warning != "" {
						cs.Diffs[i].Warning += "\n"
					}
					cs.Diffs[i].Warning += message
				}
				return
			}
		}
	}
	warn("Pi static inventory only: checks cover inspected files, configuration and declared resources; extension callbacks and runtime-only discovery/registrations remain runtime-unverified. Cold-start/reload qualification is separate.")
	if err := scan.PiRootOverride(os.LookupEnv); err != nil {
		return nil, err
	}
	agentRoot := res.ResolveMarker("~/.pi/agent", "pi", "global")
	for _, root := range []string{home, project, agentRoot} {
		if err := scan.PiSafePath(root); err != nil {
			return nil, err
		}
	}
	var discovered []scan.PiResource
	if resources {
		roots, err := scan.PiResourceRoots(home, project, agentRoot, agents, os.LookupEnv, warn)
		if err != nil {
			return nil, err
		}
		discovered, err = scan.DiscoverPiResources(roots)
		if err != nil {
			return nil, err
		}
	}
	var owners []state.Item
	for _, root := range []string{home, project} {
		s, err := state.Load(filepath.Join(root, ".patronus/state.json"))
		if err != nil {
			return nil, err
		}
		owners = append(owners, s.Items...)
	}
	var reviews []piContextReview
	for _, d := range selected {
		if err := scan.PiSafePath(d.Path); err != nil {
			return nil, err
		}
		root := agentRoot
		if d.Scope != "global" {
			root = filepath.Join(project, ".pi")
		}
		if d.Section != nil {
			if d.Scope != "global" {
				root = project
			}
		}
		rel, err := filepath.Rel(root, d.Path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
			return nil, fmt.Errorf("pi destination %s escapes selected root %s", d.Path, root)
		}
		if d.Type == "agent" {
			if _, err := adapter.ValidatePiAgent(d.Artifact, d.After); err != nil {
				return nil, err
			}
		}
		if d.Type == "agent" || d.Type == "command" || d.Type == "skill" {
			if d.Type != "skill" || filepath.Base(d.Path) == "SKILL.md" {
				selectedResource := scan.PiResource{Kind: d.Type, Name: d.Artifact, Path: d.Path, Origin: "selected " + d.Scope}
				if err := scan.PiResourceConflict(discovered, selectedResource); err != nil {
					return nil, err
				}
			}
			// A different owner at this exact path cannot be approved by --force.
			ownedUnchanged := false
			for _, owner := range owners {
				for _, f := range owner.Files {
					if f.Path == d.Path && (owner.Tool != "pi" || owner.Artifact != d.Artifact || owner.Scope != d.Scope) {
						return nil, fmt.Errorf("pi destination %s owned by %s (%s), selected %s", d.Path, owner.Artifact, owner.Tool, d.Artifact)
					}
					if f.Path == d.Path && owner.Tool == "pi" && owner.Artifact == d.Artifact && owner.Scope == d.Scope && f.Checksum == fmt.Sprintf("sha256:%x", sha256.Sum256(d.Before)) {
						ownedUnchanged = true
					}
				}
			}
			if d.Action == diff.Conflict && !ownedUnchanged {
				return nil, fmt.Errorf("pi resource conflict at %s: existing source and selected %s; resolve migration before applying selection", d.Path, d.Artifact)
			}
		}
		if d.Section != nil {
			context, err := scan.DiscoverPiContext(filepath.Dir(d.Path), scan.ReadPiFile)
			if err != nil {
				return nil, err
			}
			if context.Path != d.Path {
				return nil, fmt.Errorf("pi context %s is shadowed by %s", d.Path, context.Path)
			}
			// Check every inherited context as well, including unreadable candidates.
			if d.Scope != "global" {
				if _, err := scan.DiscoverPiContext(agentRoot, scan.ReadPiFile); err != nil {
					return nil, err
				}
				for dir := filepath.Dir(project); ; dir = filepath.Dir(dir) {
					if _, err := scan.DiscoverPiContext(dir, scan.ReadPiFile); err != nil {
						return nil, err
					}
					if filepath.Dir(dir) == dir {
						break
					}
				}
			}
			var otherOwners []string
			sections := []struct{ artifact, name string }{{d.Artifact, d.Section.Name}}
			for _, c := range d.Contrib {
				sections = append(sections, struct{ artifact, name string }{c.Artifact, c.Section})
			}
			for _, section := range sections {
				sectionOwned := false
				for _, owner := range owners {
					for _, f := range owner.Files {
						if owner.Tool == "pi" && owner.Artifact == section.artifact && owner.Scope == d.Scope && f.Section == section.name {
							if f.Path != d.Path {
								return nil, fmt.Errorf("pi context ownership at %s would be shadowed/relocated to %s", f.Path, d.Path)
							}
							// Existing state records the complete post-apply file hash,
							// not a section hash. Conservatively refuse any drift;
							// identity alone cannot authorize replacing user edits.
							if f.Checksum != fmt.Sprintf("sha256:%x", sha256.Sum256(d.Before)) {
								return nil, fmt.Errorf("pi context drift at %s section %s: recorded bytes differ; preserve edits and reconcile before apply", d.Path, section.name)
							}
							sectionOwned = true
						}
						if f.Path == d.Path && owner.Tool != "pi" {
							context.Mixed = true
							otherOwners = append(otherOwners, owner.Artifact+" ("+owner.Tool+"/"+owner.Scope+") at "+f.Path)
						}
					}
				}
				if _, exists := adapter.SectionBody(d.Before, section.name); exists && !sectionOwned {
					return nil, fmt.Errorf("pi context %s section %s has unknown ownership; refuse replacement", d.Path, section.name)
				}
			}
			if context.Mixed {
				claudeSource, err := piClaudeContextSources(res, project, d.Scope)
				if err != nil {
					return nil, err
				}
				reviews = append(reviews, piContextReview{Consent: piContextConsent{Path: d.Path, PiSource: context.Path, ClaudeSource: claudeSource, PriorHash: sha256.Sum256(d.Before), ResultHash: sha256.Sum256(d.After)}, Proposed: append([]byte(nil), d.After...), OtherOwners: otherOwners})
			}
		}
		if mcp && filepath.Base(d.Path) == "mcp-adapter.json" {
			if _, err := scan.DiscoverPiMCP(home, project, agentRoot, d.Path, os.LookupEnv, scan.ReadPiFile); err != nil {
				return nil, err
			}
			// Inspect the complete proposed source set, including collisions added
			// within one file or by two selected scopes, not just existing files.
			readProposed := func(path string) ([]byte, bool, error) {
				for _, proposed := range selected {
					if proposed.Path == path {
						return proposed.After, true, nil
					}
				}
				return scan.ReadPiFile(path)
			}
			if _, err := scan.DiscoverPiMCP(home, project, agentRoot, d.Path, os.LookupEnv, readProposed); err != nil {
				return nil, err
			}
		}
	}
	return reviews, nil
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
	reviews, err := piPreflightPlan(cs, res, opts.home, opts.projectDir)
	if err != nil {
		return err
	}
	for _, d := range cs.Diffs {
		if d.Tool == "pi" && d.Warning != "" {
			fmt.Fprintln(cmd.OutOrStdout(), d.Warning)
		}
	}
	opts.piConsents, err = confirmPiContexts(reviews, piInteractiveInput(cmd.InOrStdin()), opts.yes, opts.force, cmd.InOrStdin(), cmd.OutOrStdout())
	if err != nil {
		return err
	}
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
