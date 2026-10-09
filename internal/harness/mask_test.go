package harness

import (
	"reflect"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/scan"
)

func TestSkillMaskEdits(t *testing.T) {
	shared := []scan.SkillRoot{{Path: "/shared/skills"}}
	masked := Descriptor{Name: "toy", ItemSuffix: "tz", SkillMask: SkillMask{File: "toy.json", Format: "json", Dotted: "permission.skill", Value: "deny"}}
	unmasked := Descriptor{Name: "plain", ItemSuffix: "pl"}
	aa := DescriptorRoots{Descriptor: Descriptor{Name: "alpha", ItemSuffix: "aa"}, Roots: shared}
	bb := DescriptorRoots{Descriptor: Descriptor{Name: "beta", ItemSuffix: "bb"}, Roots: shared}
	unsuffixed := DescriptorRoots{Descriptor: Descriptor{Name: "legacy"}, Roots: shared}
	elsewhere := DescriptorRoots{Descriptor: Descriptor{Name: "gamma", ItemSuffix: "gg"}, Roots: []scan.SkillRoot{{Path: "/elsewhere"}}}
	skills := func(names ...string) []scan.Resource {
		var out []scan.Resource
		for _, n := range names {
			out = append(out, scan.Resource{Kind: "skill", Name: n, Path: "/shared/skills/" + n + "/SKILL.md"})
		}
		return out
	}
	edit := func(suffix string) diff.SettingEdit {
		return diff.SettingEdit{Target: diff.FileTargetRef{File: "toy.json", Format: "json"}, Dotted: `permission.skill."*-` + suffix + `"`, ScalarValue: "deny"}
	}
	for _, tc := range []struct {
		name       string
		target     Descriptor
		discovered []scan.Resource
		others     []DescriptorRoots
		wantEdits  []diff.SettingEdit
		wantMsgs   []string
	}{
		{"foreign skills present", masked, skills("one-aa", "two-aa", "mine-tz"), []DescriptorRoots{aa}, []diff.SettingEdit{edit("aa")},
			[]string{"Masked 2 co-visible alpha skills (*-aa) from toy via permission.skill in toy.json; re-run update after adding or removing alpha skills"}},
		{"none present", masked, skills("mine-tz"), []DescriptorRoots{aa}, nil, nil},
		{"empty-suffix descriptor ignored", masked, skills("one-aa"), []DescriptorRoots{unsuffixed}, nil, nil},
		{"roots not intersecting", masked, skills("one-gg"), []DescriptorRoots{elsewhere}, nil, nil},
		{"two foreign families", masked, skills("one-aa", "one-bb", "two-bb"), []DescriptorRoots{aa, bb}, []diff.SettingEdit{edit("aa"), edit("bb")},
			[]string{"Masked 1 co-visible alpha skills (*-aa) from toy via permission.skill in toy.json; re-run update after adding or removing alpha skills", "Masked 2 co-visible beta skills (*-bb) from toy via permission.skill in toy.json; re-run update after adding or removing beta skills"}},
		{"zero mask message only", unmasked, skills("one-aa"), []DescriptorRoots{aa}, nil,
			[]string{"Found 1 co-visible alpha skills (*-aa) in roots /shared/skills read by plain; no native mask declared"}},
		{"bare suffix is not a match", masked, skills("-aa", "aa"), []DescriptorRoots{aa}, nil, nil},
		{"self is skipped", masked, skills("one-tz"), []DescriptorRoots{{Descriptor: masked, Roots: shared}}, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			edits, msgs := SkillMaskEdits(tc.target, shared, tc.discovered, tc.others)
			if !reflect.DeepEqual(edits, tc.wantEdits) || !reflect.DeepEqual(msgs, tc.wantMsgs) {
				t.Fatalf("edits %+v msgs %q\nwant %+v %q", edits, msgs, tc.wantEdits, tc.wantMsgs)
			}
		})
	}
}

// Amendment OP-04: the quoted dotted segment serializes to the exact key
// `*-aa` under permission.skill, without quote characters.
func TestSkillMaskEditsExactKey(t *testing.T) {
	target := Descriptor{Name: "toy", ItemSuffix: "tz", SkillMask: SkillMask{File: "toy.json", Format: "json", Dotted: "permission.skill", Value: "deny"}}
	shared := []scan.SkillRoot{{Path: "/shared/skills"}}
	edits, _ := SkillMaskEdits(target, shared, []scan.Resource{{Kind: "skill", Name: "one-aa"}}, []DescriptorRoots{{Descriptor: Descriptor{Name: "alpha", ItemSuffix: "aa"}, Roots: shared}})
	if len(edits) != 1 {
		t.Fatalf("edits = %+v", edits)
	}
	out, err := adapter.MergeSettings([]byte(`{"permission":{"skill":{"*":"allow"}}}`), manifest.FileTarget{File: "toy.json", Format: "json"}, edits[0].Dotted, edits[0].ScalarValue)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"permission\": {\n    \"skill\": {\n      \"*\": \"allow\",\n      \"*-aa\": \"deny\"\n    }\n  }\n}\n"
	if string(out) != want {
		t.Fatalf("serialized\n got: %q\nwant: %q", out, want)
	}
	if strings.Contains(string(out), `"\"*-aa\""`) {
		t.Fatal("quote characters leaked into the key")
	}
}
