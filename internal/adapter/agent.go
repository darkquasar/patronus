package adapter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

	if ad.Tool == "pi" || ad.Layout.Agent.Format == "pi-subagents-markdown" {
		return e.transformPiAgent(art, ad, scope, srcDir, path)
	}

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
	content, dropped, err := reshapeAgent(ad.Layout.Agent, art, fm, body, overrides)
	if err != nil {
		return nil, err
	}
	var warning string
	if len(dropped) > 0 {
		warning = fmt.Sprintf("agent %q: frontmatter key(s) %s are not in the %s allow-list and were dropped", art.Name, strings.Join(dropped, ", "), ad.Tool)
	}
	return []diff.FileDiff{{
		Path:    path,
		Action:  diff.Create,
		After:   content,
		Tool:    ad.Tool,
		Scope:   scope,
		Role:    string(art.Role),
		Warning: warning,
	}}, nil
}

// reshapeAgent produces the per-tool agent file bytes:
//   - format "toml" (Codex): a TOML doc with name/description + the body mapped
//     to the key named by BodyIs (developer_instructions), plus overrides.
//   - frontmatter passthrough (Claude): the original markdown verbatim.
//   - frontmatter allow-list (OpenCode): a markdown doc whose frontmatter is the
//     allowed keys, with the body assigned to the BodyIs key (prompt) when listed.
//
// dropped names the frontmatter keys an allow-list reshape discarded (sorted;
// `name` excluded, since the file name is the agent identity), so the caller
// can surface them as an advisory.
func reshapeAgent(l *manifest.AgentLayout, art *manifest.Artifact, fm map[string]any, body []byte, overrides map[string]any) (content []byte, dropped []string, err error) {
	if l.Format == "toml" {
		content, err = reshapeAgentTOML(l, art, fm, body, overrides)
		return content, nil, err
	}
	if l.Frontmatter.Passthrough {
		return reassembleMarkdown(fm, body), nil, nil
	}
	if len(l.Frontmatter.Allow) > 0 {
		content, dropped = reshapeAgentMarkdownAllow(l, fm, body)
		return content, dropped, nil
	}
	// No reshape rule: pass the body through unchanged.
	return reassembleMarkdown(fm, body), nil, nil
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
func reshapeAgentMarkdownAllow(l *manifest.AgentLayout, fm map[string]any, body []byte) (content []byte, dropped []string) {
	out := map[string]any{}
	allowed := map[string]bool{}
	bodyKeyListed := false
	for _, k := range l.Frontmatter.Allow {
		allowed[k] = true
		if k == l.BodyIs {
			out[k] = string(bytes.TrimRight(body, "\n"))
			bodyKeyListed = true
			continue
		}
		if v, ok := fm[k]; ok {
			out[k] = v
		}
	}
	for k := range fm {
		if !allowed[k] && k != "name" {
			dropped = append(dropped, k)
		}
	}
	sort.Strings(dropped)
	if bodyKeyListed {
		return reassembleMarkdown(out, nil), dropped
	}
	return reassembleMarkdown(out, body), dropped
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
