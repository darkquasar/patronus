package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
)

// SettingOwner is the existing installed identity, not merely an artifact name.
type SettingOwner struct {
	Artifact string
	Tool     string
	Scope    string
}

// ValidateSettingEdit rejects ambiguous structural evidence without reading disk.
func ValidateSettingEdit(e *diff.SettingEdit) error {
	if e == nil || e.Target.File == "" {
		return fmt.Errorf("setting: missing structural evidence")
	}
	switch e.Target.Format {
	case "", "json", "jsonc", "toml":
	default:
		return fmt.Errorf("setting: unsupported format %q", e.Target.Format)
	}
	for _, part := range strings.Split(e.Dotted, ".") {
		if strings.TrimSpace(part) == "" {
			return fmt.Errorf("setting: invalid path %q", e.Dotted)
		}
	}
	if !e.PriorPresent && e.PriorValue != nil {
		return fmt.Errorf("setting %s: value with absent prior", e.Dotted)
	}
	if e.IdentityKey != "" && (e.Identity == "" || identityOf(e.Elem, e.IdentityKey) != e.Identity) {
		return fmt.Errorf("setting %s: invalid list identity", e.Dotted)
	}
	for _, v := range []any{e.ScalarValue, e.PriorValue, e.Elem} {
		if _, err := json.Marshal(v); err != nil {
			return fmt.Errorf("setting %s: invalid value: %w", e.Dotted, err)
		}
	}
	return nil
}

// SettingEditsOverlap compares path segments and preserves hook-list identity semantics.
func SettingEditsOverlap(a, b *diff.SettingEdit) bool {
	// Legacy whole-file ownership is not a leaf claim.
	if a == nil || b == nil {
		return true
	}
	if a.IdentityKey != "" && a.IdentityKey == b.IdentityKey && a.Dotted == b.Dotted {
		return a.Identity == b.Identity
	}
	return a.Dotted == b.Dotted || strings.HasPrefix(a.Dotted, b.Dotted+".") || strings.HasPrefix(b.Dotted, a.Dotted+".")
}

// SettingValuesEqual compares serialized semantic values across decoded numeric types.
func SettingValuesEqual(a, b any) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	return err == nil && bytes.Equal(left, right)
}

// SameSettingTarget identifies the same leaf (or the same hook-list element).
func SameSettingTarget(a, b *diff.SettingEdit) bool {
	return a != nil && b != nil && a.Dotted == b.Dotted && a.IdentityKey == b.IdentityKey && a.Identity == b.Identity && configFormat(a.Target.Format) == configFormat(b.Target.Format)
}

func configFormat(format string) string {
	if format == "" || format == "jsonc" {
		return "json"
	}
	return format
}

// CheckSettingPair allows only identical same-owner duplicates or disjoint edits.
// Ownership transfers cannot be authorized by an ordinary overwrite flag.
func CheckSettingPair(a *diff.SettingEdit, ownerA SettingOwner, b *diff.SettingEdit, ownerB SettingOwner) error {
	if err := ValidateSettingEdit(a); err != nil {
		return err
	}
	if err := ValidateSettingEdit(b); err != nil {
		return err
	}
	if configFormat(a.Target.Format) != configFormat(b.Target.Format) {
		return fmt.Errorf("setting: incompatible config formats")
	}
	if !SettingEditsOverlap(a, b) {
		return nil
	}
	if ownerA == ownerB && ownerA.Artifact != "" && SameSettingTarget(a, b) && SettingValuesEqual(settingValue(a), settingValue(b)) && a.PriorPresent == b.PriorPresent && SettingValuesEqual(a.PriorValue, b.PriorValue) {
		return nil
	}
	return fmt.Errorf("setting ownership conflict: %s (%s/%s) at %s overlaps %s (%s/%s) at %s", ownerA.Artifact, ownerA.Tool, ownerA.Scope, a.Dotted, ownerB.Artifact, ownerB.Tool, ownerB.Scope, b.Dotted)
}

func settingValue(e *diff.SettingEdit) any {
	if e.IdentityKey != "" {
		return e.Elem
	}
	return e.ScalarValue
}

// SettingStatus reports presence and equality of the recorded contribution. Invalid
// containers and duplicate identities are errors, never absence or equality.
func SettingStatus(existing []byte, e *diff.SettingEdit) (present, equal bool, err error) {
	if err := ValidateSettingEdit(e); err != nil {
		return false, false, err
	}
	value, present, err := ReadDotted(existing, ftOf(e), e.Dotted)
	if err != nil || !present {
		return present, false, err
	}
	if e.IdentityKey == "" {
		return true, SettingValuesEqual(value, e.ScalarValue), nil
	}
	list, ok := value.([]any)
	if !ok {
		return false, false, fmt.Errorf("setting %s: expected list", e.Dotted)
	}
	found := false
	for _, elem := range list {
		if identityOf(elem, e.IdentityKey) != e.Identity {
			continue
		}
		if found {
			return false, false, fmt.Errorf("setting %s: duplicate identity %q", e.Dotted, e.Identity)
		}
		found = true
		equal = SettingValuesEqual(elem, e.Elem)
	}
	return found, equal, nil
}
