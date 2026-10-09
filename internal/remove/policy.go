package remove

// SectionRule is one tool's section-composition rule: whether its APPEND rows
// compose and invert together (Structural), the prefix its section names carry
// (Prefix) and whether the strict Codex rules apply (Strict: ambiguous sibling
// refusal, section-equals-prefixed-artifact identity, no --force drift bypass,
// marker validation of the whole file).
type SectionRule struct {
	Tool, Prefix       string
	Structural, Strict bool
}

// legacySectionRules is the zero-value default of SectionPolicy: the two
// targets whose sections composed before the harness registry.
func legacySectionRules() []SectionRule {
	return []SectionRule{{Tool: "codex", Structural: true, Strict: true}, {Tool: "pi", Prefix: "pi:", Structural: true}}
}

// SectionPolicy says how each tool's recorded sections compose on removal. The
// zero value reproduces today's behaviour; the registry derives the production
// value from descriptor flags (internal/harness cannot be imported here).
type SectionPolicy struct {
	rules    []SectionRule
	explicit bool
}

// NewSectionPolicy returns a policy holding exactly rules (copied); with no
// arguments no tool composes sections.
func NewSectionPolicy(rules ...SectionRule) SectionPolicy {
	return SectionPolicy{rules: append([]SectionRule(nil), rules...), explicit: true}
}

func (p SectionPolicy) rule(tool string) (SectionRule, bool) {
	rules := p.rules
	if !p.explicit {
		rules = legacySectionRules()
	}
	for _, r := range rules {
		if r.Tool == tool {
			return r, true
		}
	}
	return SectionRule{}, false
}

// Structural reports whether tool's APPEND rows compose and invert together.
func (p SectionPolicy) Structural(tool string) bool {
	r, ok := p.rule(tool)
	return ok && r.Structural
}

// Prefix returns the prefix tool's section names carry ("" when none).
func (p SectionPolicy) Prefix(tool string) string {
	r, _ := p.rule(tool)
	return r.Prefix
}

// Strict reports whether the strict section rules apply to tool.
func (p SectionPolicy) Strict(tool string) bool {
	r, ok := p.rule(tool)
	return ok && r.Strict
}
