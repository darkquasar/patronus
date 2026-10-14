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
		{"non-skill resources are not counted", masked, []scan.Resource{{Kind: "agent", Name: "worker-aa"}, {Kind: "command", Name: "run-aa"}}, []DescriptorRoots{aa}, nil, nil},
		{"mixed kinds count skills only", masked, []scan.Resource{{Kind: "agent", Name: "worker-aa"}, {Kind: "skill", Name: "one-aa", Path: "/shared/skills/one-aa/SKILL.md"}}, []DescriptorRoots{aa}, []diff.SettingEdit{edit("aa")},
			[]string{"Masked 1 co-visible alpha skills (*-aa) from toy via permission.skill in toy.json; re-run update after adding or removing alpha skills"}},
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

func TestSkillMaskClaudeConditions(t *testing.T) {
	shared := []scan.SkillRoot{{Path: "/invented/.claude/skills"}}
	target := Descriptor{Name: "opencode", ItemSuffix: "oc", SkillMask: SkillMask{File: "opencode.json", Format: "jsonc", Dotted: "permission.skill", Value: "deny"}}
	claude := DescriptorRoots{Descriptor: Descriptor{Name: "claude", ItemSuffix: "cl"}, Roots: shared}
	for _, tc := range []struct {
		name   string
		roots  []scan.SkillRoot
		skills []scan.Resource
		count  int
	}{
		{"matching", shared, []scan.Resource{{Kind: "skill", Name: "workflow-cl"}}, 1},
		{"user suffix also matches", shared, []scan.Resource{{Kind: "skill", Name: "user-cl"}}, 1},
		{"neutral references not masked", shared, []scan.Resource{{Kind: "skill", Name: "decision-clarity-reference"}}, 0},
		{"bare suffix not masked", shared, []scan.Resource{{Kind: "skill", Name: "-cl"}}, 0},
		{"no skills", shared, nil, 0},
		{"no shared roots", []scan.SkillRoot{{Path: "/elsewhere"}}, []scan.Resource{{Kind: "skill", Name: "workflow-cl"}}, 0},
		{"empty roots", nil, []scan.Resource{{Kind: "skill", Name: "workflow-cl"}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			edits, messages := SkillMaskEdits(target, tc.roots, tc.skills, []DescriptorRoots{claude})
			if len(edits) != tc.count || len(messages) != tc.count {
				t.Fatalf("edits=%+v messages=%v", edits, messages)
			}
			if tc.count == 1 && (edits[0].Dotted != `permission.skill."*-cl"` || edits[0].ScalarValue != "deny") {
				t.Fatalf("Claude mask: %+v", edits)
			}
		})
	}
}
