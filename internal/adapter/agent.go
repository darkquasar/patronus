package adapter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// transformAgent reshapes a portable Agent (markdown with YAML frontmatter +
// body) into the per-tool format declared by the adapter's agent layout. No
// type: agent artifact ships today, so there is no end-to-end driver; the
// reshape logic and its unit tests live here for the recipe/profile work that
// will introduce agents.
func (e *Engine) transformAgent(art *manifest.Artifact, ad *manifest.Adapter, scope, srcDir string) ([]diff.FileDiff, error) {
	if ad.Layout.Agent == nil {
		return nil, fmt.Errorf("adapter %q: no Agent layout", ad.Tool)
	}
	target := ad.Layout.Agent.ForScope(scope)
	if !target.OK() {
		return nil, fmt.Errorf("adapter %q: Agent has no %s target", ad.Tool, scope)
	}

	// Resolve the destination BEFORE reading the source: the path depends only on
	// the name + layout, never on the body, so a caller that needs to know where an
	// agent WOULD land (drift's shadow hunt) can get it without the source present.
	path := e.resolvePath(target.Path, art.Name, ad.Tool, scope)

	entry := art.Entry
	if entry == "" {
		entry = "agent.md"
	}
	raw, err := os.ReadFile(filepath.Join(srcDir, entry))
	if err != nil {
		return nil, fmt.Errorf("adapter: read agent entry: %w", err)
	}
	fm, body := splitFrontmatter(raw)

	overrides := art.Overrides[ad.Tool]
	content, err := reshapeAgent(ad.Layout.Agent, art, fm, body, overrides)
	if err != nil {
		return nil, err
	}
	return []diff.FileDiff{{
		Path:   path,
		Action: diff.Create,
		After:  content,
		Tool:   ad.Tool,
		Scope:  scope,
		Role:   string(art.Role),
	}}, nil
}

// reshapeAgent produces the per-tool agent file bytes:
//   - format "toml" (Codex): a TOML doc with name/description + the body mapped
//     to the key named by BodyIs (developer_instructions), plus overrides.
//   - frontmatter passthrough (Claude): the original markdown verbatim.
//   - frontmatter allow-list (OpenCode): a markdown doc whose frontmatter is the
//     allowed keys, with the body assigned to the BodyIs key (prompt) when listed.
func reshapeAgent(l *manifest.AgentLayout, art *manifest.Artifact, fm map[string]any, body []byte, overrides map[string]any) ([]byte, error) {
	if l.Format == "toml" {
		return reshapeAgentTOML(l, art, fm, body, overrides)
	}
	if l.Frontmatter.Passthrough {
		return reassembleMarkdown(fm, body), nil
	}
	if len(l.Frontmatter.Allow) > 0 {
		return reshapeAgentMarkdownAllow(l, fm, body), nil
	}
	// No reshape rule: pass the body through unchanged.
	return reassembleMarkdown(fm, body), nil
}

// reshapeAgentTOML builds a TOML agent document.
func reshapeAgentTOML(l *manifest.AgentLayout, art *manifest.Artifact, fm map[string]any, body []byte, overrides map[string]any) ([]byte, error) {
	doc := map[string]any{}
	// Carry name/description from the manifest (authoritative) or frontmatter.
	doc["name"] = firstNonEmpty(stringField(fm, "name"), art.Name)
	doc["description"] = firstNonEmpty(stringField(fm, "description"), art.Description)
	if bodyKey := l.BodyIs; bodyKey != "" {
		doc[bodyKey] = string(bytes.TrimRight(body, "\n"))
	}
	// Per-tool overrides win (e.g. overrides.codex.model).
	for k, v := range overrides {
		doc[k] = v
	}
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("adapter: encode agent toml: %w", err)
	}
	return buf.Bytes(), nil
}

// reshapeAgentMarkdownAllow emits markdown with a frontmatter limited to the
// allow-listed keys. If BodyIs names a frontmatter key (e.g. "prompt"), the body
// is assigned there and the markdown body is left empty; otherwise the body is
// kept below the frontmatter.
func reshapeAgentMarkdownAllow(l *manifest.AgentLayout, fm map[string]any, body []byte) []byte {
	out := map[string]any{}
	bodyKeyListed := false
	for _, k := range l.Frontmatter.Allow {
		if k == l.BodyIs {
			out[k] = string(bytes.TrimRight(body, "\n"))
			bodyKeyListed = true
			continue
		}
		if v, ok := fm[k]; ok {
			out[k] = v
		}
	}
	if bodyKeyListed {
		return reassembleMarkdown(out, nil)
	}
	return reassembleMarkdown(out, body)
}

// splitFrontmatter separates a leading YAML frontmatter block (delimited by ---)
// from the markdown body. When absent, fm is empty and body is the whole input.
func splitFrontmatter(raw []byte) (map[string]any, []byte) {
	fm := map[string]any{}
	s := raw
	if !bytes.HasPrefix(s, []byte("---\n")) && !bytes.HasPrefix(s, []byte("---\r\n")) {
		return fm, raw
	}
	// Find the closing delimiter.
	rest := s[bytes.IndexByte(s, '\n')+1:]
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return fm, raw
	}
	header := rest[:end]
	body := rest[end+1:]
	// Drop the closing "---" line.
	if nl := bytes.IndexByte(body, '\n'); nl >= 0 {
		body = body[nl+1:]
	} else {
		body = nil
	}
	if err := yaml.Unmarshal(header, &fm); err != nil {
		// Treat unparseable frontmatter as no frontmatter.
		return map[string]any{}, raw
	}
	return fm, body
}

// reassembleMarkdown writes a YAML frontmatter block (if any keys) followed by
// the body.
func reassembleMarkdown(fm map[string]any, body []byte) []byte {
	var buf bytes.Buffer
	if len(fm) > 0 {
		buf.WriteString("---\n")
		out, _ := yaml.Marshal(fm)
		buf.Write(out)
		buf.WriteString("---\n")
	}
	if len(body) > 0 {
		if len(fm) > 0 {
			buf.WriteByte('\n')
		}
		buf.Write(body)
	}
	return buf.Bytes()
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
