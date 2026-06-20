package adapter

import (
	"fmt"
	"strings"

	"github.com/darkquasar/patronus/internal/manifest"
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
// at the layout's dotted path. It builds the transport object from the layout's
// ordered key templates, then rides the format-neutral merger in config.go to
// set the dotted path. It is the MCP-specific caller of that deep module — the
// only thing here that is genuinely about MCP is buildTransportObject; the merge
// itself is shared with settings/hook wiring.
func MergeConfig(existing []byte, ft manifest.FileTarget, tr manifest.Transport, spec ServerSpec) ([]byte, error) {
	if tr.Keys == nil {
		return nil, fmt.Errorf("mcp: transport %q has no key template", spec.Transport)
	}
	obj := buildTransportObject(tr, spec.Values)
	dotted := strings.ReplaceAll(ft.Path, "{name}", spec.Name)
	return MergeSettings(existing, ft, dotted, obj)
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
