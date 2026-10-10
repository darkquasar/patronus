package adapter

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// TestRedactForDisplay: the display copy of a diff whose Redact globs match
// replaces every matching value with the placeholder in Before and After, the
// original diff's bytes are untouched, a glob with no match returns the bytes
// unchanged, and unparseable bytes are replaced wholesale rather than shown.
func TestRedactForDisplay(t *testing.T) {
	before := []byte(`{"provider":{"inv":{"options":{"apiKey":"fixture-not-a-secret"}}},"user":1}` + "\n")
	after := []byte(`{"provider":{"inv":{"options":{"apiKey":"fixture-not-a-secret"}},"two":{"options":{"token":"second-fixture"}}},"user":2}` + "\n")
	d := diff.FileDiff{Path: "/invented/opencode.json", Action: diff.Merge, Before: before, After: after, Redact: []string{"provider.*.options"}}
	shown := RedactForDisplay(d)
	for _, leak := range []string{"fixture-not-a-secret", "second-fixture"} {
		if strings.Contains(shown.Unified(), leak) || bytes.Contains(shown.Before, []byte(leak)) || bytes.Contains(shown.After, []byte(leak)) {
			t.Fatalf("display copy leaks %q:\n%s", leak, shown.Unified())
		}
	}
	if !strings.Contains(string(shown.After), `"options": "[redacted]"`) || !strings.Contains(string(shown.After), `"user": 2`) {
		t.Fatalf("display copy shape:\n%s", shown.After)
	}
	if !bytes.Equal(d.Before, before) || !bytes.Equal(d.After, after) {
		t.Fatal("original diff bytes were altered")
	}

	noMatch := diff.FileDiff{Path: "/invented/opencode.json", Action: diff.Merge, Before: before, After: after, Redact: []string{"nothing.*.here"}}
	if s := RedactForDisplay(noMatch); !bytes.Equal(s.Before, before) || !bytes.Equal(s.After, after) {
		t.Fatalf("no-match redaction changed bytes:\n%s", s.After)
	}

	broken := diff.FileDiff{Path: "/invented/opencode.json", Action: diff.Merge, Before: []byte(`{"provider": {"inv": {"options": {"apiKey": "fixture-not-a-secret"`), After: after, Redact: []string{"provider.*.options"}}
	if s := RedactForDisplay(broken); bytes.Contains(s.Before, []byte("fixture-not-a-secret")) || !bytes.Contains(s.Before, []byte("[redacted")) {
		t.Fatalf("unparseable bytes shown: %s", s.Before)
	}

	none := diff.FileDiff{Path: "/invented/opencode.json", Action: diff.Merge, Before: before, After: after}
	if s := RedactForDisplay(none); !bytes.Equal(s.After, after) {
		t.Fatal("a diff without Redact must be returned as is")
	}
}

// TestStampRedactMarksSettingFileDiffsOnly: diffs whose path is the adapter's
// setting layout file (either scope) receive the layout's redact globs as a
// copy; every other path is untouched; an adapter without redact data stamps
// nothing.
func TestStampRedactMarksSettingFileDiffsOnly(t *testing.T) {
	home, proj := t.TempDir(), t.TempDir()
	res := toolpath.New(testEnv(home), home, proj)
	ad := loadAdapter(t, "opencode")
	diffs := []diff.FileDiff{
		{Path: filepath.Join(proj, "opencode.json"), Action: diff.Merge},
		{Path: filepath.Join(home, ".config/opencode/opencode.json"), Action: diff.Merge},
		{Path: filepath.Join(proj, ".opencode/skills/x/SKILL.md"), Action: diff.Create},
		{Path: filepath.Join(proj, ".opencode/patronus/hooks.json"), Action: diff.Merge},
	}
	StampRedact(diffs, ad, res)
	for i, want := range []string{"provider.*.options", "provider.*.options", "", ""} {
		if got := strings.Join(diffs[i].Redact, ","); got != want {
			t.Fatalf("diff %d (%s) redact = %q, want %q", i, diffs[i].Path, got, want)
		}
	}
	diffs[0].Redact[0] = "mutated"
	if ad.Layout.Setting.Redact[0] != "provider.*.options" {
		t.Fatal("stamp shared the adapter's slice")
	}
	plain := []diff.FileDiff{{Path: filepath.Join(proj, ".claude/settings.json"), Action: diff.Merge}}
	StampRedact(plain, loadAdapter(t, "claude"), res)
	if plain[0].Redact != nil {
		t.Fatalf("claude stamped: %v", plain[0].Redact)
	}
}
