package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// Ledger item_suffix values are one safe lowercase alphanumeric component: the
// disposition `port-<item_suffix>` must stay a plain literal.
var _itemSuffixPattern = regexp.MustCompile(`^[a-z0-9]+$`)

type profileCase struct {
	Profile string `json:"profile"`
	Target  string `json:"target"`
}

func (c *catalog) cases() ([]profileCase, error) {
	m, err := readDocument(c.root, "docs/compatibility/profile-targets.yaml")
	if err != nil {
		return nil, fmt.Errorf("profile cases metadata: %w", err)
	}
	if m["schema_version"] != 1 {
		return nil, fmt.Errorf("profile cases: unsupported schema_version")
	}
	profiles := object(m, "profiles")
	if len(profiles) == 0 {
		return nil, fmt.Errorf("profile cases: empty profiles")
	}
	var cases []profileCase
	for _, name := range sortedNames(c.items) {
		if text(c.items[name].manifest, "family") != "profile" {
			continue
		}
		if !caseComponent(name) {
			return nil, fmt.Errorf("profile cases: unsafe profile name %s", name)
		}
		targets, err := stringList(object(profiles, name)["targets"])
		if err != nil || len(targets) == 0 {
			return nil, fmt.Errorf("profile cases: missing/empty targets for %s", name)
		}
		sort.Strings(targets)
		seen := map[string]bool{}
		for _, target := range targets {
			if !caseComponent(target) || !c.targets[target] || seen[target] {
				return nil, fmt.Errorf("profile cases: invalid/duplicate target %s for %s", target, name)
			}
			seen[target] = true
			cases = append(cases, profileCase{Profile: name, Target: target})
		}
		if err := agreeDeclaredTargets(c.items[name].manifest, targets); err != nil {
			return nil, fmt.Errorf("profile cases: %s: %w", name, err)
		}
	}
	for name := range profiles {
		if it := c.items[name]; it == nil || text(it.manifest, "family") != "profile" {
			return nil, fmt.Errorf("profile cases: unknown profile %s", name)
		}
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("profile cases: empty selection")
	}
	return cases, nil
}

// agreeDeclaredTargets requires a profile manifest that declares its own
// `targets:` to admit exactly the profile-targets.yaml set. A profile without
// the field (including one that only `extends:` a declaring profile) is not
// declaring and is skipped; profile-targets.yaml stays the admitted-case source.
func agreeDeclaredTargets(m document, admitted []string) error {
	declared, err := stringList(m["targets"])
	if err != nil {
		return fmt.Errorf("manifest targets: %w", err)
	}
	if len(declared) == 0 {
		return nil
	}
	declared = slices.Clone(declared)
	sort.Strings(declared)
	if !slices.Equal(declared, admitted) {
		return fmt.Errorf("manifest declares targets %v but profile-targets.yaml admits %v", declared, admitted)
	}
	return nil
}
func (c *catalog) writeCases(w io.Writer) error {
	cases, err := c.cases()
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(cases)
}
func directMembers(m document) ([]string, error) {
	var out []string
	for _, v := range object(m, "layers") {
		list, err := stringList(v)
		if err != nil {
			return nil, err
		}
		out = append(out, list...)
	}
	return out, nil
}
func (c *catalog) readClosures(dir string) (map[profileCase]map[string]bool, error) {
	cases, err := c.cases()
	if err != nil {
		return nil, err
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	locks := map[profileCase]map[string]bool{}
	for _, pc := range cases {
		rel := pc.Profile + "--" + pc.Target + ".lock"
		m, err := readDocument(root, rel)
		if err != nil {
			return nil, fmt.Errorf("closure %s: %w", rel, err)
		}
		version, ok := m["version"].(int)
		if !ok || version < 1 || version > 4 {
			return nil, fmt.Errorf("closure %s: unsupported lock version", rel)
		}
		if text(m, "profile") != pc.Profile || (text(m, "target") != "" && text(m, "target") != pc.Target) || (version >= 3 && text(m, "target") == "") {
			return nil, fmt.Errorf("closure %s: profile/target identity mismatch", rel)
		}
		entries, ok := m["entries"].([]any)
		if !ok || len(entries) == 0 {
			return nil, fmt.Errorf("closure %s: empty/unresolved selection", rel)
		}
		names := map[string]bool{}
		for _, value := range entries {
			entry, ok := asDocument(value)
			if !ok {
				return nil, fmt.Errorf("closure %s: malformed entry", rel)
			}
			name := text(entry, "name")
			it := c.items[name]
			if it == nil || text(it.manifest, "family") == "profile" || names[name] {
				return nil, fmt.Errorf("closure %s: unknown/duplicate entry %s", rel, name)
			}
			if text(entry, "version") != text(it.manifest, "version") {
				return nil, fmt.Errorf("closure %s: entry version mismatch %s", rel, name)
			}
			if kind := text(entry, "kind"); kind != "" && kind != text(it.manifest, "family") {
				return nil, fmt.Errorf("closure %s: entry family mismatch %s", rel, name)
			}
			names[name] = true
		}
		locks[pc] = names
	}
	return locks, nil
}
func (c *catalog) checkLedgers(closureDir string) error {
	var locks map[profileCase]map[string]bool
	if closureDir != "" {
		var err error
		locks, err = c.readClosures(closureDir)
		if err != nil {
			return err
		}
	}
	dir := filepath.Join(c.root, "docs/compatibility")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml") || entry.Name() == "profile-targets.yaml" {
			continue
		}
		rel := "docs/compatibility/" + entry.Name()
		m, err := readDocument(c.root, rel)
		if err != nil {
			return err
		}
		// Other compatibility documents are not profile ledgers. Presence of the
		// profile/entries contract, not a filename or artifact name, selects a ledger.
		if m["entries"] == nil && m["profile"] == nil {
			continue
		}
		if err := c.checkLedger(m, locks); err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
	}
	return nil
}
func (c *catalog) checkLedger(m document, locks map[profileCase]map[string]bool) error {
	if m["schema_version"] != 1 {
		return fmt.Errorf("unsupported ledger schema_version")
	}
	profile, baseline := text(m, "profile"), text(m, "baseline_profile")
	selected, base := c.items[profile], c.items[baseline]
	if selected == nil || base == nil || text(selected.manifest, "family") != "profile" || text(base.manifest, "family") != "profile" {
		return fmt.Errorf("missing/unknown profile or baseline_profile")
	}
	baselineTarget := text(m, "baseline_target")
	if !c.targets[baselineTarget] {
		return fmt.Errorf("missing/unknown baseline_target")
	}
	target, suffix := text(m, "target"), text(m, "item_suffix")
	if target == "" || suffix == "" {
		return fmt.Errorf("ledger requires target: and item_suffix:")
	}
	if target == "all" || !c.targets[target] {
		return fmt.Errorf("unknown ledger target %s", target)
	}
	if !_itemSuffixPattern.MatchString(suffix) {
		return fmt.Errorf("unsafe item_suffix %q", suffix)
	}
	itemsKey, portDisposition := target+"_items", "port-"+suffix
	members, err := directMembers(base.manifest)
	if err != nil {
		return err
	}
	baselineSet := map[string]bool{}
	for _, name := range members {
		baselineSet[name] = true
	}
	entries, ok := m["entries"].([]any)
	if !ok || len(entries) == 0 || len(baselineSet) == 0 {
		return fmt.Errorf("empty ledger/baseline members")
	}
	seen, mapped := map[string]bool{}, map[string]bool{}
	for _, value := range entries {
		row, ok := asDocument(value)
		if !ok {
			return fmt.Errorf("malformed ledger row")
		}
		core := text(row, "core_item")
		if !baselineSet[core] || seen[core] {
			return fmt.Errorf("unknown/duplicate core_item %s", core)
		}
		seen[core] = true
		if strings.TrimSpace(text(row, "reason")) == "" {
			return fmt.Errorf("%s: missing reason", core)
		}
		switch text(row, "outcome") {
		case "Equivalent", "Partial", "N/A", "EnvironmentBlocked":
		default:
			return fmt.Errorf("%s: unsupported outcome", core)
		}
		switch text(row, "evidence") {
		case "static-confirmed", "runtime-pass", "runtime-pending", "environment-blocked", "historical", "not-run":
		default:
			return fmt.Errorf("%s: unsupported evidence label", core)
		}
		endpoints, err := stringList(row[itemsKey])
		if err != nil {
			return err
		}
		switch text(row, "disposition") {
		case "omit":
			if len(endpoints) != 0 || text(row, "outcome") == "Equivalent" {
				return fmt.Errorf("%s: invalid omit", core)
			}
		case portDisposition, "reuse-unsuffixed":
			if len(endpoints) == 0 {
				return fmt.Errorf("%s: missing endpoints", core)
			}
		default:
			return fmt.Errorf("%s: unsupported disposition", core)
		}
		if err := c.mapEndpoints(core, endpoints, mapped); err != nil {
			return err
		}
	}
	for member := range baselineSet {
		if !seen[member] {
			return fmt.Errorf("missing mapping for %s", member)
		}
	}
	if err := c.checkAdditionalItems(m, mapped); err != nil {
		return err
	}
	companions, err := stringList(m["companion_items"])
	if err != nil {
		return err
	}
	for _, name := range companions {
		if c.items[name] == nil || mapped[name] {
			return fmt.Errorf("missing/duplicate companion %s", name)
		}
		mapped[name] = true
	}
	direct, err := directMembers(selected.manifest)
	if err != nil {
		return err
	}
	for _, name := range direct {
		unqualified, _, _ := strings.Cut(name, "@")
		if !mapped[unqualified] {
			return fmt.Errorf("profile member %s lacks ledger evidence", name)
		}
	}
	if locks == nil {
		return nil
	} // Packaging success explicitly labels closure coverage NOT CHECKED.
	if locks[profileCase{Profile: baseline, Target: baselineTarget}] == nil {
		return fmt.Errorf("baseline closure case not supplied")
	}
	// The ledger binds one profile/target case: its endpoints are required in
	// that closure only, and that closure must have been supplied.
	closure := locks[profileCase{Profile: profile, Target: target}]
	if closure == nil {
		return fmt.Errorf("no supplied closure for ledger profile %s target %s", profile, target)
	}
	for name := range mapped {
		if !closure[name] {
			return fmt.Errorf("%s--%s closure missing ledger endpoint %s", profile, target, name)
		}
	}
	return nil
}

// mapEndpoints records existing non-profile catalog items as mapped ledger
// endpoints; label is the row identity used in diagnostics.
func (c *catalog) mapEndpoints(label string, endpoints []string, mapped map[string]bool) error {
	for _, endpoint := range endpoints {
		it := c.items[endpoint]
		if it == nil || text(it.manifest, "family") == "profile" {
			return fmt.Errorf("%s: missing endpoint %s", label, endpoint)
		}
		mapped[endpoint] = true
	}
	return nil
}

// checkAdditionalItems validates the optional `additional_items:` rows (lane
// items with no baseline core member) with the entry vocabulary and counts
// their items as mapped endpoints for closure coverage.
func (c *catalog) checkAdditionalItems(m document, mapped map[string]bool) error {
	raw, present := m["additional_items"]
	if !present || raw == nil {
		return nil
	}
	rows, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("additional_items: expected list, got %T", raw)
	}
	for i, value := range rows {
		row, ok := asDocument(value)
		if !ok {
			return fmt.Errorf("additional_items[%d]: malformed row", i)
		}
		label := fmt.Sprintf("additional_items[%d]", i)
		if strings.TrimSpace(text(row, "reason")) == "" {
			return fmt.Errorf("%s: missing reason", label)
		}
		switch text(row, "outcome") {
		case "Equivalent", "Partial", "N/A", "EnvironmentBlocked":
		default:
			return fmt.Errorf("%s: unsupported outcome", label)
		}
		switch text(row, "evidence") {
		case "static-confirmed", "runtime-pass", "runtime-pending", "environment-blocked", "historical", "not-run":
		default:
			return fmt.Errorf("%s: unsupported evidence label", label)
		}
		items, err := stringList(row["items"])
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		if len(items) == 0 && text(row, "outcome") == "Equivalent" {
			return fmt.Errorf("%s: Equivalent requires items", label)
		}
		if err := c.mapEndpoints(label, items, mapped); err != nil {
			return err
		}
	}
	return nil
}
