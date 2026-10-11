package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// codexAuthPrerequisites reports explicitly configured Codex MCP auth and
// environment references whose environment variable is not present. It reads
// only the planned config bytes Patronus already holds and checks variable
// presence through lookup; values are never read into the report, copied into
// ownership or logged. Servers without such references are not assumed to need
// credentials. The result is advisory: a missing reference is an actionable
// prerequisite for the user, never a silent pass and never a write.
func codexAuthPrerequisites(cs *diff.ChangeSet, lookup func(string) (string, bool)) []string {
	if cs == nil || lookup == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for i := range cs.Diffs {
		d := &cs.Diffs[i]
		if d.Tool != "codex" || d.Setting == nil || d.Setting.Target.Format != "toml" {
			continue
		}
		parts := strings.Split(d.Setting.Dotted, ".")
		if len(parts) < 2 || parts[0] != "mcp_servers" {
			continue
		}
		name := parts[1]
		raw := d.After
		if len(raw) == 0 {
			raw = d.Before
		}
		ft := manifest.FileTarget{File: d.Setting.Target.File, Format: d.Setting.Target.Format}
		server, present, err := adapter.ReadDotted(raw, ft, "mcp_servers."+name)
		if err != nil || !present {
			continue
		}
		table, ok := server.(map[string]any)
		if !ok {
			continue
		}
		for _, ref := range codexAuthRefs(table) {
			if _, set := lookup(ref.env); set {
				continue
			}
			msg := fmt.Sprintf("Codex MCP server %q prerequisite: %s references environment variable %s, which is not set; export %s in the environment Codex starts from (Patronus checks presence only and does not read or copy credentials)", name, ref.field, ref.env, ref.env)
			key := d.Path + "\x00" + msg
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, msg)
		}
	}
	return out
}

type codexAuthRef struct{ field, env string }

// codexAuthRefs lists the environment-variable references a Codex MCP server
// table explicitly configures: bearer_token_env_var, env_http_headers values
// and env_vars entries. Literal env/http_headers values are not references.
func codexAuthRefs(table map[string]any) []codexAuthRef {
	var refs []codexAuthRef
	if v, ok := table["bearer_token_env_var"].(string); ok && v != "" {
		refs = append(refs, codexAuthRef{"bearer_token_env_var", v})
	}
	if m, ok := table["env_http_headers"].(map[string]any); ok {
		headers := make([]string, 0, len(m))
		for h := range m {
			headers = append(headers, h)
		}
		sort.Strings(headers)
		for _, h := range headers {
			if v, ok := m[h].(string); ok && v != "" {
				refs = append(refs, codexAuthRef{"env_http_headers." + h, v})
			}
		}
	}
	if l, ok := table["env_vars"].([]any); ok {
		for _, e := range l {
			if v, ok := e.(string); ok && v != "" {
				refs = append(refs, codexAuthRef{"env_vars", v})
			}
		}
	}
	return refs
}
