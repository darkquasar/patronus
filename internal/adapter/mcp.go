package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/darkquasar/patronus/internal/manifest"
	toml "github.com/pelletier/go-toml/v2"
)

// ServerSpec is a recipe-supplied MCP server description. Recipes are Phase 4, so
// there is no end-to-end driver yet; this type and MergeConfig are exercised by
// unit tests and ready for the recipe engine to call.
type ServerSpec struct {
	Name      string         // server id; substituted into the dotted path {name}
	Transport string         // "stdio" | "http" — selects the layout transport
	Values    map[string]any // placeholder values: command, args, env, url, headers, commandArray, bearerTokenEnvVar
}

// MergeConfig computes the new bytes of a tool config after wiring spec into it
// at the layout's dotted path. It parses existing per ft.Format (json/jsonc/toml),
// builds the transport object from the layout's ordered key templates, sets the
// dotted path, and re-serializes. It is pure — the planner reads existing bytes
// and classifies the result; the Phase-3 applier writes them.
func MergeConfig(existing []byte, ft manifest.FileTarget, tr manifest.Transport, spec ServerSpec) ([]byte, error) {
	if !ft.OK() {
		return nil, fmt.Errorf("mcp: empty file target")
	}
	if tr.Keys == nil {
		return nil, fmt.Errorf("mcp: transport %q has no key template", spec.Transport)
	}

	root, err := parseConfig(existing, ft.Format)
	if err != nil {
		return nil, err
	}

	obj := buildTransportObject(tr, spec.Values)
	dotted := strings.ReplaceAll(ft.Path, "{name}", spec.Name)
	if err := setDotted(root, dotted, obj); err != nil {
		return nil, err
	}

	return serializeConfig(root, ft.Format)
}

// buildTransportObject renders the transport's key template into a concrete
// object. Literal templates (no braces, e.g. "stdio"/"local"/"remote") emit
// verbatim; placeholder templates ("{command}", "{args}", …) are looked up in
// vals and inserted with their native type. A missing value drops the key — this
// is exactly how Codex omits the absent `type` key (§9.9): it simply is not in
// the template, so nothing is emitted.
func buildTransportObject(tr manifest.Transport, vals map[string]any) map[string]any {
	out := map[string]any{}
	for _, k := range tr.Keys.Keys {
		tmpl := tr.Keys.Vals[k]
		ph, isPlaceholder := placeholder(tmpl)
		if !isPlaceholder {
			out[k] = tmpl // literal value, e.g. type: "stdio"
			continue
		}
		if v, ok := vals[ph]; ok && v != nil {
			out[k] = v
		}
		// absent placeholder value -> key omitted
	}
	return out
}

// placeholder reports whether tmpl is a single "{name}" placeholder and returns
// the inner name.
func placeholder(tmpl string) (string, bool) {
	if len(tmpl) >= 2 && tmpl[0] == '{' && tmpl[len(tmpl)-1] == '}' {
		return tmpl[1 : len(tmpl)-1], true
	}
	return "", false
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
			return nil, fmt.Errorf("mcp: parse toml: %w", err)
		}
	case "json", "jsonc", "":
		data := existing
		if format == "jsonc" {
			data = stripJSONComments(existing)
		}
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, fmt.Errorf("mcp: parse json: %w", err)
		}
	default:
		return nil, fmt.Errorf("mcp: unsupported format %q", format)
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
			return nil, fmt.Errorf("mcp: encode toml: %w", err)
		}
		return buf.Bytes(), nil
	case "json", "jsonc", "":
		out, err := json.MarshalIndent(root, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("mcp: encode json: %w", err)
		}
		return append(out, '\n'), nil
	default:
		return nil, fmt.Errorf("mcp: unsupported format %q", format)
	}
}

// setDotted sets a value at a dotted path within root, creating intermediate
// maps as needed. Existing sibling keys are preserved (structural MERGE).
func setDotted(root map[string]any, dotted string, val any) error {
	parts := strings.Split(dotted, ".")
	cur := root
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = val
			return nil
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
			return fmt.Errorf("mcp: cannot descend into non-object key %q", p)
		}
		cur[p] = m // normalize (toml may decode as map[string]any already)
		cur = m
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
// document can be parsed by encoding/json. String contents are preserved. Phase 2
// does not attempt to round-trip comments (DESIGN: deferred to the Phase-3
// writer); it only needs correct merged bytes for conflict classification.
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
