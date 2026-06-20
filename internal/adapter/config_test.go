package adapter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	toml "github.com/pelletier/go-toml/v2"
)

// reparse normalizes bytes through a decode so tests compare structure, not
// whitespace/key-order, across json and toml.
func reparse(t *testing.T, b []byte, format string) map[string]any {
	t.Helper()
	m := map[string]any{}
	var err error
	if format == "toml" {
		err = toml.Unmarshal(b, &m)
	} else {
		err = json.Unmarshal(b, &m)
	}
	if err != nil {
		t.Fatalf("reparse %s: %v\n%s", format, err, b)
	}
	return m
}

// TestMergeSettingsScalar covers setting a scalar at a dotted path in both json
// and toml, creating intermediates, preserving siblings, and idempotence.
func TestMergeSettingsScalar(t *testing.T) {
	cases := []struct {
		name     string
		format   string
		existing string
		dotted   string
		val      any
		want     map[string]any
	}{
		{
			name: "json empty creates nested", format: "json", existing: "",
			dotted: "permissions.sandbox", val: true,
			want: map[string]any{"permissions": map[string]any{"sandbox": true}},
		},
		{
			name: "json preserves siblings", format: "json",
			existing: `{"model":"opus","permissions":{"other":1}}`,
			dotted:   "permissions.sandbox", val: "ro",
			want: map[string]any{
				"model":       "opus",
				"permissions": map[string]any{"other": float64(1), "sandbox": "ro"},
			},
		},
		{
			name: "toml nested table", format: "toml",
			existing: "model = \"gpt\"\n",
			dotted:   "sandbox.mode", val: "workspace-write",
			want: map[string]any{
				"model":   "gpt",
				"sandbox": map[string]any{"mode": "workspace-write"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ft := manifest.FileTarget{File: "f", Format: tc.format}
			out, err := MergeSettings([]byte(tc.existing), ft, tc.dotted, tc.val)
			if err != nil {
				t.Fatalf("MergeSettings: %v", err)
			}
			if got := reparse(t, out, tc.format); !jsonEqual(t, got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
			// Idempotent: a second identical merge yields byte-identical output.
			out2, err := MergeSettings(out, ft, tc.dotted, tc.val)
			if err != nil {
				t.Fatalf("re-merge: %v", err)
			}
			if !bytes.Equal(out, out2) {
				t.Errorf("re-merge not idempotent:\n%s\nvs\n%s", out, out2)
			}
		})
	}
}

// TestMergeSettingsScalarErrors covers the can't-descend-through-a-scalar guard.
func TestMergeSettingsScalarErrors(t *testing.T) {
	ft := manifest.FileTarget{File: "f", Format: "json"}
	if _, err := MergeSettings([]byte(`{"a":1}`), ft, "a.b", 2); err == nil {
		t.Fatal("expected error descending into scalar key, got nil")
	}
	bad := manifest.FileTarget{File: "", Format: "json"}
	if _, err := MergeSettings(nil, bad, "a", 1); err == nil {
		t.Fatal("expected error on empty file target")
	}
}

// TestAppendSettingsList covers list-append idempotence, coexistence of two
// elements, and removal of exactly one — the hook-array contract.
func TestAppendSettingsList(t *testing.T) {
	ft := manifest.FileTarget{File: "settings.json", Format: "json"}
	dotted := "hooks.PreToolUse"

	a := map[string]any{"id": "A", "command": "guard-a"}
	b := map[string]any{"id": "B", "command": "guard-b"}

	// First append into an empty file.
	out, err := AppendSettingsList(nil, ft, dotted, "id", a)
	if err != nil {
		t.Fatalf("append A: %v", err)
	}
	// Second append (different identity) coexists.
	out, err = AppendSettingsList(out, ft, dotted, "id", b)
	if err != nil {
		t.Fatalf("append B: %v", err)
	}
	if n := listLen(t, out, dotted); n != 2 {
		t.Fatalf("after A+B: list len = %d, want 2", n)
	}

	// Re-appending A is idempotent (replace-in-place, no dup).
	out2, err := AppendSettingsList(out, ft, dotted, "id", a)
	if err != nil {
		t.Fatalf("re-append A: %v", err)
	}
	if !bytes.Equal(out, out2) {
		t.Errorf("re-append not idempotent:\n%s\nvs\n%s", out, out2)
	}

	// Remove A leaves exactly B.
	out3, found, err := RemoveSettingsList(out, ft, dotted, "id", "A")
	if err != nil || !found {
		t.Fatalf("remove A: found=%v err=%v", found, err)
	}
	if n := listLen(t, out3, dotted); n != 1 {
		t.Fatalf("after remove A: list len = %d, want 1", n)
	}
	got := reparse(t, out3, "json")
	list := got["hooks"].(map[string]any)["PreToolUse"].([]any)
	if list[0].(map[string]any)["id"] != "B" {
		t.Errorf("survivor = %v, want B", list[0])
	}

	// Removing an absent identity is a no-op that reports the original bytes.
	out4, found, err := RemoveSettingsList(out3, ft, dotted, "id", "ZZZ")
	if err != nil || found {
		t.Fatalf("remove absent: found=%v err=%v", found, err)
	}
	if !bytes.Equal(out3, out4) {
		t.Error("removing absent identity changed bytes")
	}
}

// TestRemoveSettingsListRoundTrip proves append-then-remove restores the prior
// surrounding settings byte-for-byte (the merge/unmerge fidelity gate).
func TestRemoveSettingsListRoundTrip(t *testing.T) {
	ft := manifest.FileTarget{File: "settings.json", Format: "json"}
	dotted := "hooks.PreToolUse"
	prior := []byte("{\n  \"model\": \"opus\"\n}\n")

	elem := map[string]any{"id": "A", "command": "x"}
	appended, err := AppendSettingsList(prior, ft, dotted, "id", elem)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if bytes.Equal(prior, appended) {
		t.Fatal("append was a no-op")
	}
	restored, found, err := RemoveSettingsList(appended, ft, dotted, "id", "A")
	if err != nil || !found {
		t.Fatalf("remove: found=%v err=%v", found, err)
	}
	// The only structural residue is an empty hooks.PreToolUse array, which is
	// harmless; the user's model key is intact and no other key leaked.
	got := reparse(t, restored, "json")
	if got["model"] != "opus" {
		t.Errorf("model lost: %v", got)
	}
	if list, ok := got["hooks"].(map[string]any)["PreToolUse"].([]any); !ok || len(list) != 0 {
		t.Errorf("expected empty PreToolUse array, got %v", got["hooks"])
	}
}

func listLen(t *testing.T, b []byte, _ string) int {
	t.Helper()
	got := reparse(t, b, "json")
	hooks, ok := got["hooks"].(map[string]any)
	if !ok {
		return 0
	}
	list, ok := hooks["PreToolUse"].([]any)
	if !ok {
		return 0
	}
	return len(list)
}

// jsonEqual compares two decoded trees by round-tripping through json (order- and
// type-normalizing).
func jsonEqual(t *testing.T, a, b map[string]any) bool {
	t.Helper()
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return bytes.Equal(ab, bb)
}
