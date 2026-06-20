package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	toml "github.com/pelletier/go-toml/v2"
)

// This file is the format-neutral config merger: parse existing bytes (json /
// jsonc / toml) into a generic tree, set or list-append a value at a dotted
// path, and re-serialize. It is the deep module under both MCP wiring
// (MergeConfig, mcp.go) and settings/hook wiring (MergeSettings,
// AppendSettingsList). Deletion test: remove mcp.go and this merger still
// stands — the generality is here, not in the MCP naming.
//
// The functions are pure: the planner reads existing bytes and classifies the
// result; the applier writes them.

// MergeSettings sets val at ft's dotted path within existing and re-serializes,
// preserving every sibling key. It is the scalar-merge entry point for a hook
// or native-switch artifact wiring one key into an agent's settings file. dotted
// is the resolved path (placeholders already substituted by the caller).
func MergeSettings(existing []byte, ft manifest.FileTarget, dotted string, val any) ([]byte, error) {
	if !ft.OK() {
		return nil, fmt.Errorf("config: empty file target")
	}
	root, err := parseConfig(existing, ft.Format)
	if err != nil {
		return nil, err
	}
	if err := setDotted(root, dotted, val); err != nil {
		return nil, err
	}
	return serializeConfig(root, ft.Format)
}

// AppendSettingsList appends elem to the array at ft's dotted path within
// existing and re-serializes. The append is idempotent on identity: if an array
// element already carries the same value at identityKey, elem replaces it in
// place rather than duplicating. This is how two hooks on one event coexist —
// each owns one element keyed by its matcher — and re-running an install is a
// no-op. identityKey "" makes every append unconditional (no dedup).
func AppendSettingsList(existing []byte, ft manifest.FileTarget, dotted, identityKey string, elem map[string]any) ([]byte, error) {
	if !ft.OK() {
		return nil, fmt.Errorf("config: empty file target")
	}
	root, err := parseConfig(existing, ft.Format)
	if err != nil {
		return nil, err
	}
	if err := appendDotted(root, dotted, identityKey, elem); err != nil {
		return nil, err
	}
	return serializeConfig(root, ft.Format)
}

// RemoveSettingsList strips from the array at ft's dotted path the single
// element whose identityKey equals identityVal, returning the result and whether
// an element was removed. It is the inverse of AppendSettingsList: surrounding
// elements (user-added or other-artifact) and sibling keys are preserved, so an
// append-then-remove round-trips. A now-empty array key is left in place (an
// empty `hooks.{event}: []` is harmless and keeps the parent object stable).
func RemoveSettingsList(existing []byte, ft manifest.FileTarget, dotted, identityKey, identityVal string) ([]byte, bool, error) {
	if !ft.OK() {
		return nil, false, fmt.Errorf("config: empty file target")
	}
	root, err := parseConfig(existing, ft.Format)
	if err != nil {
		return nil, false, err
	}
	removed := removeDotted(root, dotted, identityKey, identityVal)
	if !removed {
		return existing, false, nil
	}
	out, err := serializeConfig(root, ft.Format)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

// ftOf reconstructs the manifest.FileTarget a SettingEdit needs to merge or
// remove. A SettingEdit carries only file+format (diff must not import manifest),
// so the format-aware merger is reassembled here, at the one seam that owns it.
func ftOf(e *diff.SettingEdit) manifest.FileTarget {
	return manifest.FileTarget{File: e.Target.File, Format: e.Target.Format}
}

// ApplySettingEdit (re-)appends e's element onto existing, idempotently on
// e.Identity. The planner uses it to re-fold a settings list-append onto an
// accumulated config so several hooks into one file all survive.
func ApplySettingEdit(existing []byte, e *diff.SettingEdit) ([]byte, error) {
	return AppendSettingsList(existing, ftOf(e), e.Dotted, e.IdentityKey, e.Elem)
}

// RemoveSettingEdit strips e's element from existing (matched on identity) and
// reports whether one was removed. It is the targeted inverse remove uses, so a
// hook reverts without disturbing sibling hooks on the same event.
func RemoveSettingEdit(existing []byte, e *diff.SettingEdit) ([]byte, bool, error) {
	return RemoveSettingsList(existing, ftOf(e), e.Dotted, e.IdentityKey, e.Identity)
}

// parseConfig decodes existing config bytes into a generic map. Empty input
// yields a fresh map. JSONC is parsed as JSON after stripping comments.
func parseConfig(existing []byte, format string) (map[string]any, error) {
	root := map[string]any{}
	if len(bytes.TrimSpace(existing)) == 0 {
		return root, nil
	}
	switch format {
	case "toml":
		if err := toml.Unmarshal(existing, &root); err != nil {
			return nil, fmt.Errorf("config: parse toml: %w", err)
		}
	case "json", "jsonc", "":
		data := existing
		if format == "jsonc" {
			data = stripJSONComments(existing)
		}
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("config: parse json: %w", err)
		}
	default:
		return nil, fmt.Errorf("config: unsupported format %q", format)
	}
	return root, nil
}

// serializeConfig re-emits root in the target format. JSON map keys are emitted
// in stdlib (alphabetical) order, which is deterministic.
func serializeConfig(root map[string]any, format string) ([]byte, error) {
	switch format {
	case "toml":
		var buf bytes.Buffer
		enc := toml.NewEncoder(&buf)
		enc.SetIndentTables(true)
		if err := enc.Encode(root); err != nil {
			return nil, fmt.Errorf("config: encode toml: %w", err)
		}
		return buf.Bytes(), nil
	case "json", "jsonc", "":
		out, err := json.MarshalIndent(root, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("config: encode json: %w", err)
		}
		return append(out, '\n'), nil
	default:
		return nil, fmt.Errorf("config: unsupported format %q", format)
	}
}

// setDotted sets a value at a dotted path within root, creating intermediate
// maps as needed. Existing sibling keys are preserved (structural MERGE).
func setDotted(root map[string]any, dotted string, val any) error {
	parent, leaf, err := descend(root, dotted)
	if err != nil {
		return err
	}
	parent[leaf] = val
	return nil
}

// appendDotted appends elem to the array at dotted (creating it if absent),
// replacing an existing element that matches on identityKey so the operation is
// idempotent. Intermediate maps are created as needed.
func appendDotted(root map[string]any, dotted, identityKey string, elem map[string]any) error {
	parent, leaf, err := descend(root, dotted)
	if err != nil {
		return err
	}
	list := asAnyList(parent[leaf])
	if i := indexByIdentity(list, identityKey, elem); i >= 0 {
		list[i] = elem // idempotent replace-in-place
	} else {
		list = append(list, elem)
	}
	parent[leaf] = list
	return nil
}

// removeDotted strips the element at dotted whose identityKey equals identityVal
// and reports whether one was removed. A missing path or non-list value simply
// reports false (not an error), so remove is safe against a settings file the
// user has since edited.
func removeDotted(root map[string]any, dotted, identityKey, identityVal string) bool {
	parent, leaf := descendExisting(root, dotted)
	if parent == nil {
		return false
	}
	list := asAnyList(parent[leaf])
	if list == nil {
		return false
	}
	kept := make([]any, 0, len(list))
	removed := false
	for _, e := range list {
		if !removed && identityOf(e, identityKey) == identityVal {
			removed = true
			continue
		}
		kept = append(kept, e)
	}
	if removed {
		parent[leaf] = kept
	}
	return removed
}

// descend walks dotted to its leaf, creating intermediate maps as needed, and
// returns the parent map plus the final key. It errors only when a path segment
// names an existing non-object value (can't descend through a scalar).
func descend(root map[string]any, dotted string) (parent map[string]any, leaf string, err error) {
	parts := strings.Split(dotted, ".")
	cur := root
	for i, p := range parts {
		if i == len(parts)-1 {
			return cur, p, nil
		}
		next, ok := cur[p]
		if !ok {
			m := map[string]any{}
			cur[p] = m
			cur = m
			continue
		}
		m, ok := asStringMap(next)
		if !ok {
			return nil, "", fmt.Errorf("config: cannot descend into non-object key %q", p)
		}
		cur[p] = m // normalize (toml may decode as map[any]any)
		cur = m
	}
	return cur, "", nil // unreachable: dotted is never empty
}

// descendExisting walks dotted without creating intermediates, returning the
// parent map + final key. A missing segment yields a nil parent, so callers treat
// "path absent" as "no work" rather than failing.
func descendExisting(root map[string]any, dotted string) (parent map[string]any, leaf string) {
	parts := strings.Split(dotted, ".")
	cur := root
	for i, p := range parts {
		if i == len(parts)-1 {
			return cur, p
		}
		m, ok := asStringMap(cur[p])
		if !ok {
			return nil, "" // path doesn't reach a leaf: nothing to remove
		}
		cur[p] = m
		cur = m
	}
	return nil, "" // unreachable
}

// indexByIdentity returns the index of the first list element matching elem on
// identityKey, or -1. identityKey "" disables matching (always -1 → append).
func indexByIdentity(list []any, identityKey string, elem map[string]any) int {
	if identityKey == "" {
		return -1
	}
	want := fmt.Sprint(elem[identityKey])
	for i, e := range list {
		if identityOf(e, identityKey) == want {
			return i
		}
	}
	return -1
}

// identityOf reads the identityKey field of a list element as a string, or ""
// when the element is not an object or lacks the key.
func identityOf(e any, identityKey string) string {
	m, ok := asStringMap(e)
	if !ok {
		return ""
	}
	v, ok := m[identityKey]
	if !ok {
		return ""
	}
	return fmt.Sprint(v)
}

// asAnyList coerces a decoded value into a []any, returning nil for absent or
// non-list values (a nil slice is a usable empty list for append).
func asAnyList(v any) []any {
	if l, ok := v.([]any); ok {
		return l
	}
	return nil
}

// asStringMap coerces a decoded value into a map[string]any when possible.
func asStringMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, vv := range m {
			out[fmt.Sprint(k)] = vv
		}
		return out, true
	default:
		return nil, false
	}
}

// stripJSONComments removes // line comments and /* */ block comments so a JSONC
// document can be parsed by encoding/json. String contents are preserved. The
// merger does not round-trip comments; it only needs correct merged bytes for
// conflict classification.
func stripJSONComments(in []byte) []byte {
	var out bytes.Buffer
	inString := false
	escaped := false
	for i := 0; i < len(in); i++ {
		c := in[i]
		if inString {
			out.WriteByte(c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch {
		case c == '"':
			inString = true
			out.WriteByte(c)
		case c == '/' && i+1 < len(in) && in[i+1] == '/':
			for i < len(in) && in[i] != '\n' {
				i++
			}
			if i < len(in) {
				out.WriteByte('\n')
			}
		case c == '/' && i+1 < len(in) && in[i+1] == '*':
			i += 2
			for i+1 < len(in) && !(in[i] == '*' && in[i+1] == '/') {
				i++
			}
			i++ // land on '/'
		default:
			out.WriteByte(c)
		}
	}
	return out.Bytes()
}
