package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/scan"
)

// piValidator holds the Pi-only admission rules: the root override guard and
// multi-kind resource discovery in Prepare, root containment and agent
// validation before ownership, effective-context discovery with the ancestor
// walk after ownership, and the mcp-adapter.json collision check after the
// context phase. It is stateless; Pi's mixed-context reviews leave through
// Admission.Review and consent stays operation-local in piConsentSet.
type piValidator struct{}

var _ harness.Validator = piValidator{}

// piConsentSet is the opaque mutation consent for Pi mixed-context writes.
type piConsentSet []piContextConsent

var _ mutationConsent = piConsentSet(nil)

// Validate rechecks every consent bound to path against the bytes on disk and
// the result about to be written; a stale consent is never silently renewed.
func (c piConsentSet) Validate(path string, result []byte) error {
	for _, consent := range c {
		if consent.Path != path {
			continue
		}
		prior, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("pi mixed-context consent stale for %s: %w; new interactive preview required", path, err)
		}
		if err := validatePiConsentBytes(consent, path, prior, result); err != nil {
			return err
		}
	}
	return nil
}

func piAgentRoot(d harness.Descriptor, in *harness.Admission) string {
	return in.Resolver.ResolveMarker(d.GlobalMarker, d.Name, "global")
}

func piSelectedDiffs(d harness.Descriptor, in *harness.Admission) []diff.FileDiff {
	var out []diff.FileDiff
	for _, fd := range in.Changes.Diffs {
		if fd.Native == nil && fd.Tool == d.Name {
			out = append(out, fd)
		}
	}
	return out
}

func (piValidator) Prepare(d harness.Descriptor, in *harness.Admission) error {
	if err := scan.PiRootOverride(in.Env); err != nil {
		return err
	}
	agentRoot := piAgentRoot(d, in)
	for _, root := range []string{in.Home, in.Project, agentRoot} {
		if err := scan.PiSafePath(root); err != nil {
			return err
		}
	}
	agents, resources := false, false
	for _, fd := range piSelectedDiffs(d, in) {
		agents = agents || fd.Type == "agent"
		resources = resources || fd.Type == "agent" || fd.Type == "skill" || fd.Type == "command"
	}
	in.InventoryResolved = true
	if !resources {
		return nil
	}
	var warnings []func(string)
	if in.Warn != nil {
		warnings = append(warnings, in.Warn)
	}
	roots, err := scan.PiResourceRoots(in.Home, in.Project, agentRoot, agents, in.Env, warnings...)
	if err != nil {
		return err
	}
	// Prepare precedes shared owner loading. Reference discovery needs that same
	// both-scope state evidence before it can judge duplicate payloads.
	if in.LoadOwners != nil {
		in.Owners, err = in.LoadOwners()
		if err != nil {
			return err
		}
	}
	proof := harness.ManagedReferenceCoexistence(in.Owners)
	discovered, err := scan.DiscoverPiResources(roots, proof)
	if err != nil {
		return err
	}
	if len(in.Roots) > 0 {
		foreign, err := scan.DiscoverSkills(in.Roots, nil, proof)
		if err != nil {
			return err
		}
		for _, r := range foreign {
			discovered = append(discovered, scan.PiResource{Kind: r.Kind, Name: r.Name, Path: r.Path, Origin: r.Origin, Aliases: r.Aliases, Content: r.Content, Legacy: r.Legacy})
		}
		discovered, err = scan.ReconcilePiResources(discovered, proof)
		if err != nil {
			return err
		}
		// Prepare resolved both inventories. Do not append the same foreign roots
		// again in shared admission, which would undo logical-reference collapse.
		in.Roots = nil
	}
	in.Inventory = make([]scan.Resource, 0, len(discovered))
	for _, r := range discovered {
		in.Inventory = append(in.Inventory, scan.Resource{Kind: r.Kind, Name: r.Name, Path: r.Path, Origin: r.Origin, Aliases: r.Aliases, Content: r.Content, Legacy: r.Legacy})
	}
	return nil
}

func (piValidator) BeforeOwnership(d harness.Descriptor, in *harness.Admission, fd diff.FileDiff) error {
	root := piAgentRoot(d, in)
	if fd.Scope != "global" {
		root = filepath.Join(in.Project, ".pi")
		if fd.Section != nil {
			root = in.Project
		}
	}
	rel, err := filepath.Rel(root, fd.Path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return fmt.Errorf("%s destination %s escapes selected root %s", d.ErrorPrefix, fd.Path, root)
	}
	if fd.Type == "agent" {
		if _, err := adapter.ValidatePiAgent(fd.Artifact, fd.After); err != nil {
			return err
		}
	}
	return nil
}

// AfterOwnership discovers the effective context for a section diff: the
// destination must be the discovered file, every inherited context must be
// readable, and a CLAUDE source or foreign sections mark the context mixed.
func (piValidator) AfterOwnership(d harness.Descriptor, in *harness.Admission, fd diff.FileDiff, facts *harness.ContextFacts) error {
	if fd.Section == nil {
		return nil
	}
	context, err := scan.DiscoverPiContext(filepath.Dir(fd.Path), scan.ReadPiFile)
	if err != nil {
		return err
	}
	if context.Path != fd.Path {
		return fmt.Errorf("%s context %s is shadowed by %s", d.ErrorPrefix, fd.Path, context.Path)
	}
	if fd.Scope != "global" {
		if _, err := scan.DiscoverPiContext(piAgentRoot(d, in), scan.ReadPiFile); err != nil {
			return err
		}
		for dir := filepath.Dir(in.Project); ; dir = filepath.Dir(dir) {
			if _, err := scan.DiscoverPiContext(dir, scan.ReadPiFile); err != nil {
				return err
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	facts.Mixed = facts.Mixed || context.Mixed
	facts.Sources = []string{context.Path}
	return nil
}

// AfterContext inspects the complete proposed mcp-adapter.json source set,
// including collisions added within one file or by two selected scopes.
func (piValidator) AfterContext(d harness.Descriptor, in *harness.Admission, fd diff.FileDiff, _ *harness.ContextFacts) error {
	if filepath.Base(fd.Path) != "mcp-adapter.json" {
		return nil
	}
	agentRoot := piAgentRoot(d, in)
	if _, err := scan.DiscoverPiMCP(in.Home, in.Project, agentRoot, fd.Path, in.Env, scan.ReadPiFile); err != nil {
		return err
	}
	selected := piSelectedDiffs(d, in)
	readProposed := func(path string) ([]byte, bool, error) {
		for _, proposed := range selected {
			if proposed.Path == path {
				return proposed.After, true, nil
			}
		}
		return scan.ReadPiFile(path)
	}
	if _, err := scan.DiscoverPiMCP(in.Home, in.Project, agentRoot, fd.Path, in.Env, readProposed); err != nil {
		return err
	}
	return nil
}
