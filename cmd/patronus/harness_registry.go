package main

import (
	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/scan"
)

// defaultRegistry builds the production target registry once, in main. Codex
// and Pi carry today's behaviour as flags plus their stateless validators;
// claude and opencode exist for target-enum parity with every flag false and
// no validator, so the legacy shared-lane semantics are unchanged.
func defaultRegistry() (*harness.Registry, error) {
	return harness.NewRegistry(
		harness.Descriptor{
			Name:                "codex",
			ItemSuffix:          "cx",
			Profiles:            []string{"core-profile-cx"},
			ContextFile:         "AGENTS.md",
			ShadowFiles:         []string{"AGENTS.override.md"},
			SectionPrefix:       "",
			GlobalMarker:        "~/.codex",
			SkillRoots:          codexSkillRoots,
			SkillName:           scan.CodexSkillName,
			ResourcePhase:       harness.ResourceAfterContext,
			LockTargeted:        true,
			LockCoexistence:     true,
			OwnsEffects:         true,
			StrictUnowned:       true,
			StrictSections:      true,
			SafePathCheck:       true,
			AdmissionPrefix:     "codex admission",
			MixedContextConsent: false,
			ErrorPrefix:         "codex",
			PlacementWarning:    "Codex static admission/placement only; skill loading, hooks, trust and fresh-session activation remain runtime-unverified",
			Validator:           codexValidator{},
		},
		harness.Descriptor{
			Name:                "pi",
			ItemSuffix:          "pi",
			Profiles:            []string{"core-profile-pi"},
			ContextFile:         "AGENTS.md",
			SectionPrefix:       "pi:",
			GlobalMarker:        "~/.pi/agent",
			SkillRoots:          piSkillRoots,
			ResourcePhase:       harness.ResourceBeforeOwnership,
			LockTargeted:        true,
			LockCoexistence:     false,
			OwnsEffects:         true,
			StrictUnowned:       false,
			StrictSections:      false,
			SafePathCheck:       true,
			AdmissionPrefix:     "",
			MixedContextConsent: true,
			ErrorPrefix:         "pi",
			PlacementWarning:    "Pi static inventory only: checks cover inspected files, configuration and declared resources; extension callbacks and runtime-only discovery/registrations remain runtime-unverified. Cold-start/reload qualification is separate.",
			Validator:           piValidator{},
		},
		harness.Descriptor{
			Name:         "claude",
			ContextFile:  "CLAUDE.md",
			GlobalMarker: "~/.claude",
		},
		harness.Descriptor{
			Name:         "opencode",
			ItemSuffix:   "oc",
			ContextFile:  "AGENTS.md",
			GlobalMarker: "~/.config/opencode",
		},
	)
}

func codexSkillRoots(in harness.RootInputs) ([]scan.SkillRoot, error) {
	return scan.CodexDiscoveryRoots(in.Home, in.Project, in.Config), nil
}

func piSkillRoots(in harness.RootInputs) ([]scan.SkillRoot, error) {
	var warnings []func(string)
	if in.Warn != nil {
		warnings = append(warnings, in.Warn)
	}
	return scan.PiSkillDiscoveryRoots(in.Home, in.Project, in.Config, in.Env, warnings...)
}
