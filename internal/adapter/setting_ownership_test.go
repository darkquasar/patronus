package adapter

import (
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
)

func TestSettingOwnershipPairs(t *testing.T) {
	owner := SettingOwner{Artifact: "fixture", Tool: "pi", Scope: "global"}
	scalar := &diff.SettingEdit{Target: diff.FileTargetRef{File: "settings.json", Format: "json"}, Dotted: "ui.color", ScalarValue: true}
	for _, tt := range []struct {
		name   string
		dotted string
		value  any
		owner  SettingOwner
	}{
		{"different artifact", "ui.color", true, SettingOwner{Artifact: "other", Tool: "pi", Scope: "global"}},
		{"different tool", "ui.color", true, SettingOwner{Artifact: "fixture", Tool: "claude", Scope: "global"}},
		{"different scope", "ui.color", true, SettingOwner{Artifact: "fixture", Tool: "pi", Scope: "local"}},
		{"same owner different scalar", "ui.color", false, owner},
		{"ancestor", "ui", map[string]any{"color": true}, owner},
		{"descendant", "ui.color.extra", true, owner},
	} {
		t.Run(tt.name, func(t *testing.T) {
			other := *scalar
			other.Dotted, other.ScalarValue = tt.dotted, tt.value
			if err := CheckSettingPair(scalar, owner, &other, tt.owner); err == nil {
				t.Fatal("overlap accepted")
			}
		})
	}
	if err := CheckSettingPair(scalar, owner, scalar, owner); err != nil {
		t.Fatal(err)
	}
}

func TestSettingOwnershipHookIdentity(t *testing.T) {
	a := &diff.SettingEdit{Target: diff.FileTargetRef{File: "settings.json", Format: "json"}, Dotted: "hooks.Before", IdentityKey: "id", Identity: "first", Elem: map[string]any{"id": "first"}}
	b := *a
	b.Identity, b.Elem = "second", map[string]any{"id": "second"}
	if err := CheckSettingPair(a, SettingOwner{Artifact: "one"}, &b, SettingOwner{Artifact: "two"}); err != nil {
		t.Fatal(err)
	}
	if SettingEditsOverlap(a, &b) {
		t.Fatal("disjoint hook identities overlap")
	}
	b.Identity, b.Elem = a.Identity, a.Elem
	if err := CheckSettingPair(a, SettingOwner{Artifact: "one"}, &b, SettingOwner{Artifact: "two"}); err == nil {
		t.Fatal("shared hook identity accepted")
	}
}

func TestSettingStatusInvalidEvidence(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `{"ui":`, `{"ui":true}`, `{"ui":{"color":false,"color":true}}`} {
		t.Run(raw, func(t *testing.T) {
			e := &diff.SettingEdit{Target: diff.FileTargetRef{File: "settings.json", Format: "json"}, Dotted: "ui.color", ScalarValue: true}
			if _, _, err := SettingStatus([]byte(raw), e); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}

func TestSettingStatusSemanticFormats(t *testing.T) {
	for _, tt := range []struct{ format, raw string }{
		{"json", `{"count":1,"unrelated":false}`},
		{"jsonc", "{ // comment\n \"count\": 1 }"},
		{"toml", "count = 1\nunrelated = false\n"},
	} {
		t.Run(tt.format, func(t *testing.T) {
			e := &diff.SettingEdit{Target: diff.FileTargetRef{File: "config", Format: tt.format}, Dotted: "count", ScalarValue: 1}
			present, equal, err := SettingStatus([]byte(tt.raw), e)
			if err != nil || !present || !equal {
				t.Fatalf("semantic equality: %v %v %v", present, equal, err)
			}
		})
	}
}

func TestSettingStatusAbsentAndNull(t *testing.T) {
	e := &diff.SettingEdit{Target: diff.FileTargetRef{File: "settings.json", Format: "json"}, Dotted: "value"}
	present, equal, err := SettingStatus([]byte(`{"value":null}`), e)
	if err != nil || !present || !equal {
		t.Fatalf("null not present/equal: %v %v %v", present, equal, err)
	}
	present, equal, err = SettingStatus([]byte(`{}`), e)
	if err != nil || present || equal {
		t.Fatalf("absent inferred as null: %v %v %v", present, equal, err)
	}
	e.PriorValue = true
	if err := ValidateSettingEdit(e); err == nil {
		t.Fatal("inconsistent prior accepted")
	}
}
