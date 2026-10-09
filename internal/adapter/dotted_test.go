package adapter

import (
	"reflect"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

// Characterization: unquoted dotted paths produce exactly these bytes through
// MergeSettings, RemoveSettingScalar and ReadDotted (JSON and TOML). Recorded
// at the base revision before the quoted-segment splitter landed (HS-T2).
func TestDottedUnquotedPathsRoundTripBytes(t *testing.T) {
	for _, tc := range []struct {
		name, format, existing, dotted string
		val                            any
		wantMerged, wantRemoved        string
	}{
		{"json nested", "json", "{\n  \"model\": \"x\"\n}\n", "permission.edit", "deny",
			"{\n  \"model\": \"x\",\n  \"permission\": {\n    \"edit\": \"deny\"\n  }\n}\n",
			"{\n  \"model\": \"x\",\n  \"permission\": {}\n}\n"},
		{"json hyphen key", "json", "", "mcp.fixture-name.url", "https://fixture.invalid",
			"{\n  \"mcp\": {\n    \"fixture-name\": {\n      \"url\": \"https://fixture.invalid\"\n    }\n  }\n}\n",
			"{\n  \"mcp\": {\n    \"fixture-name\": {}\n  }\n}\n"},
		{"toml nested", "toml", "model = 'x'\n", "sandbox.mode", "ro",
			"model = 'x'\n\n[sandbox]\n  mode = 'ro'\n",
			"model = 'x'\n\n[sandbox]\n"},
		{"toml mcp leaf prunes empty server", "toml", "", "mcp_servers.fixture.url", "https://fixture.invalid",
			"[mcp_servers]\n  [mcp_servers.fixture]\n    url = 'https://fixture.invalid'\n",
			"[mcp_servers]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ft := manifest.FileTarget{File: "settings", Format: tc.format}
			merged, err := MergeSettings([]byte(tc.existing), ft, tc.dotted, tc.val)
			if err != nil {
				t.Fatal(err)
			}
			if string(merged) != tc.wantMerged {
				t.Fatalf("merged bytes\n got: %q\nwant: %q", merged, tc.wantMerged)
			}
			got, present, err := ReadDotted(merged, ft, tc.dotted)
			if err != nil || !present || !reflect.DeepEqual(got, tc.val) {
				t.Fatalf("ReadDotted = %v %v %v", got, present, err)
			}
			removed, ok, err := RemoveSettingScalar(merged, ft, tc.dotted, nil, false)
			if err != nil || !ok {
				t.Fatalf("RemoveSettingScalar = %v %v", ok, err)
			}
			if string(removed) != tc.wantRemoved {
				t.Fatalf("removed bytes\n got: %q\nwant: %q", removed, tc.wantRemoved)
			}
		})
	}
}

func TestSplitDotted(t *testing.T) {
	for _, tc := range []struct {
		dotted string
		want   []string
		err    string
	}{
		{"a", []string{"a"}, ""},
		{"a.b.c", []string{"a", "b", "c"}, ""},
		{"mcp_servers.fixture-name.url", []string{"mcp_servers", "fixture-name", "url"}, ""},
		{`permission.edit."*.env"`, []string{"permission", "edit", "*.env"}, ""},
		{`permission.edit.".env.*"`, []string{"permission", "edit", ".env.*"}, ""},
		{`"a.b".c`, []string{"a.b", "c"}, ""},
		{`a."".c`, []string{"a", "", "c"}, ""},
		{`a.b"c`, []string{"a", `b"c`}, ""}, // a quote inside an unquoted segment stays literal, as before
		{`a."b.c`, nil, `config: unterminated quoted segment in dotted path "a.\"b.c"`},
		{`a."b"c.d`, nil, `config: quoted segment must end at a dot in dotted path "a.\"b\"c.d"`},
	} {
		t.Run(tc.dotted, func(t *testing.T) {
			got, err := splitDotted(tc.dotted)
			if tc.err != "" {
				if err == nil || err.Error() != tc.err {
					t.Fatalf("error = %v want %s", err, tc.err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("splitDotted(%q) = %q, %v want %q", tc.dotted, got, err, tc.want)
			}
		})
	}
}

// A quoted segment writes and reads the exact key without quote characters,
// in both formats, and a malformed path is refused by every entry point.
func TestDottedQuotedSegmentWritesExactKey(t *testing.T) {
	for _, tc := range []struct {
		format, want string
	}{
		{"json", "{\n  \"permission\": {\n    \"edit\": {\n      \"*.env\": \"deny\"\n    }\n  }\n}\n"},
		{"toml", "[permission]\n  [permission.edit]\n    '*.env' = 'deny'\n"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			ft := manifest.FileTarget{File: "settings", Format: tc.format}
			merged, err := MergeSettings(nil, ft, `permission.edit."*.env"`, "deny")
			if err != nil {
				t.Fatal(err)
			}
			if string(merged) != tc.want {
				t.Fatalf("merged bytes\n got: %q\nwant: %q", merged, tc.want)
			}
			got, present, err := ReadDotted(merged, ft, `permission.edit."*.env"`)
			if err != nil || !present || got != "deny" {
				t.Fatalf("ReadDotted = %v %v %v", got, present, err)
			}
			if _, present, err := ReadDotted(merged, ft, "permission.edit.*.env"); err != nil || present {
				t.Fatalf("unquoted spelling must not reach the quoted key: %v %v", present, err)
			}
			removed, ok, err := RemoveSettingScalar(merged, ft, `permission.edit."*.env"`, nil, false)
			if err != nil || !ok {
				t.Fatalf("RemoveSettingScalar = %v %v", ok, err)
			}
			if _, present, _ := ReadDotted(removed, ft, `permission.edit."*.env"`); present {
				t.Fatal("quoted key survived removal")
			}
			if _, err := MergeSettings(nil, ft, `permission."edit`, "deny"); err == nil {
				t.Fatal("malformed quoting accepted by MergeSettings")
			}
			if _, _, err := ReadDotted(nil, ft, `permission."edit`); err == nil {
				t.Fatal("malformed quoting accepted by ReadDotted")
			}
		})
	}
}
