package harness

import (
	"fmt"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
)

// Message construction keeps every historical text byte-identical per error
// prefix. Each function is a switch on the prefix with a neutral default for
// targets that had no text before; no map and no package-level state.

func crossTargetMessage(prefix, path, tool string) error {
	switch prefix {
	case "codex":
		return fmt.Errorf("codex cross-target destination %s (%s): explicit mixed-context migration required", path, tool)
	default:
		return fmt.Errorf("%s cross-target destination conflict %s (%s)", prefix, path, tool)
	}
}

func incompatibleOwnerMessage(prefix string, fd diff.FileDiff, owner state.Item) error {
	switch prefix {
	case "codex":
		return fmt.Errorf("codex destination %s has incompatible owner %s (%s)", fd.Path, owner.Artifact, owner.Tool)
	default:
		return fmt.Errorf("%s destination %s owned by %s (%s), selected %s", prefix, fd.Path, owner.Artifact, owner.Tool, fd.Artifact)
	}
}

func ownershipDriftMessage(prefix, path string) error {
	return fmt.Errorf("%s ownership drift at %s; preserve edits, reconcile before mutation (no --force bypass)", prefix, path)
}

func unknownOwnershipMessage(prefix, path string) error {
	return fmt.Errorf("%s destination %s has unknown ownership; explicit adoption/migration required", prefix, path)
}

func resourceConflictMessage(prefix string, fd diff.FileDiff) error {
	return fmt.Errorf("%s resource conflict at %s: existing source and selected %s; resolve migration before applying selection", prefix, fd.Path, fd.Artifact)
}

func shadowMessage(prefix, path, shadow string) error {
	switch prefix {
	case "codex":
		return fmt.Errorf("codex context %s is shadowed by %s; reconcile before mutation", path, shadow)
	default:
		return fmt.Errorf("%s context %s is shadowed by %s", prefix, path, shadow)
	}
}

func relocationMessage(prefix, from, to string) error {
	switch prefix {
	case "codex":
		return fmt.Errorf("codex context ownership at %s cannot be silently relocated to %s", from, to)
	default:
		return fmt.Errorf("%s context ownership at %s would be shadowed/relocated to %s", prefix, from, to)
	}
}

func mixedContextMessage(prefix, path string, owner state.Item) error {
	return fmt.Errorf("%s mixed-context %s owned by %s (%s); explicit migration/consent required", prefix, path, owner.Artifact, owner.Tool)
}

func contextDriftMessage(prefix, path, section string) error {
	switch prefix {
	case "codex":
		return fmt.Errorf("codex context ownership drift at %s; preserve edits before mutation", path)
	default:
		return fmt.Errorf("%s context drift at %s section %s: recorded bytes differ; preserve edits and reconcile before apply", prefix, path, section)
	}
}

func ambiguousSectionMessage(prefix, path string) error {
	return fmt.Errorf("%s context %s has ambiguous section ownership", prefix, path)
}

func composedInverseMessage(prefix, path string) error {
	return fmt.Errorf("%s context %s lacks a verified composed section inverse; retain sections and ownership", prefix, path)
}

func unknownContextMessage(prefix, path string) error {
	return fmt.Errorf("%s context %s has unknown ownership; preserve prose and obtain explicit migration/consent", prefix, path)
}

func unknownSectionMessage(prefix, path, section string) error {
	switch prefix {
	case "codex":
		return fmt.Errorf("codex context %s section %s has incompatible ownership", path, section)
	default:
		return fmt.Errorf("%s context %s section %s has unknown ownership; refuse replacement", prefix, path, section)
	}
}

func legacyDriftMessage(prefix, path string) error {
	return fmt.Errorf("%s legacy ownership checksum drift at %s; preserve edits, explicit migration required (no --force bypass)", prefix, path)
}

func legacyOwnershipMessage(prefix, path string) error {
	switch prefix {
	case "codex":
		return fmt.Errorf("codex legacy ownership at %s requires explicit migration/retirement (patronus migrate codex-skills); automatic migration is Partial even if the old file is absent", path)
	default:
		return fmt.Errorf("%s legacy ownership at %s requires explicit migration/retirement", prefix, path)
	}
}

func legacyDiscoveredMessage(prefix, path, artifact string) error {
	return fmt.Errorf("%s legacy skill %s: automatic migration is Partial; preserve files and reconcile ownership/checksums and retire the old root explicitly before installing %s", prefix, path, artifact)
}

func selectedConflictMessage(prefix, name string, other, selected scan.Resource) error {
	switch prefix {
	case "codex":
		return fmt.Errorf("codex skill %q conflicts with %s; incompatible same-name resource", name, other.Path)
	default:
		return fmt.Errorf("%s %s identity %q conflict: %s (%s) and %s (%s); silent shadowing refused", prefix, selected.Kind, name, other.Path, other.Origin, selected.Path, selected.Origin)
	}
}
