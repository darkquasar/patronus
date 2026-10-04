package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/scan"
)

func (e *Engine) transformPiAgent(art *manifest.Artifact, ad *manifest.Adapter, scope, srcDir, path string) ([]diff.FileDiff, error) {
	l := ad.Layout.Agent
	fail := func(reason string) ([]diff.FileDiff, error) {
		return nil, fmt.Errorf("artifact %q target %q field layout line 1: %s; use the finite pi-subagents-markdown contract", art.Name, ad.Tool, reason)
	}
	if ad.Tool != "pi" || l.Format != "pi-subagents-markdown" {
		return fail("native format requires Pi; Pi requires explicit native format")
	}
	if len(art.Targets) != 1 || art.Targets[0] != "pi" || l.BodyIs != "" || l.Frontmatter.Passthrough || l.Frontmatter.Allow != nil {
		return fail("incompatible targets, BodyIs or Frontmatter")
	}
	if len(art.Files) > 0 || len(art.Overrides) > 0 {
		return fail("Files and artifact Overrides are unsupported")
	}
	if art.Entry == "" || strings.ToLower(filepath.Ext(art.Entry)) != ".md" {
		return fail("explicit Markdown entry required")
	}
	entry, err := scan.PiSourcePath(srcDir, art.Entry)
	if err != nil {
		return fail(err.Error())
	}
	if err := scan.PiSafePath(path); err != nil {
		return fail(err.Error())
	}
	if filepath.Base(path) != art.Name+".md" || !scan.PiSafeName(art.Name) {
		return fail("destination basename and catalog identity disagree")
	}
	raw, err := os.ReadFile(entry)
	if err != nil {
		return nil, err
	}
	if _, err := ValidatePiAgent(art.Name, raw); err != nil {
		return nil, err
	}
	return []diff.FileDiff{{Path: path, Action: diff.Create, After: raw, Tool: ad.Tool, Scope: scope, Role: string(art.Role)}}, nil
}

// ValidatePiAgent admits the deliberately finite native grammar without YAML
// coercion or rewriting. The returned skill names allow selection-closure checks.
func ValidatePiAgent(name string, raw []byte) ([]string, error) {
	fail := func(field string, line int, why string) error {
		return fmt.Errorf("artifact %q target pi field %s line %d: %s; use the supported native shape", name, field, line, why)
	}
	if !utf8.Valid(raw) {
		return nil, fail("document", 1, "invalid UTF-8")
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return nil, fail("document", 1, "exact opening --- required")
	}
	seen := map[string]string{}
	var skills []string
	closed := false
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "---" {
			if strings.TrimSpace(strings.Join(lines[i+1:], "\n")) == "" {
				return nil, fail("body", i+2, "nonempty prompt required")
			}
			closed = true
			break
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || key == "" || strings.TrimSpace(key) != key || strings.ContainsAny(key, " \t\r") {
			return nil, fail("document", i+1, "malformed top-level entry or resumed list")
		}
		if _, ok := seen[key]; ok {
			return nil, fail(key, i+1, "duplicate key")
		}
		value = strings.TrimSpace(value)
		seen[key] = value
		fieldLine := i + 1
		switch key {
		case "name", "description", "model", "output":
			s, err := piNativeString(value)
			if err != nil {
				return nil, fail(key, fieldLine, err.Error())
			}
			if key == "name" && s != name {
				return nil, fail(key, fieldLine, "native name and catalog identity disagree")
			}
		case "tools", "excludeTools", "skills", "extensions", "subagentOnlyExtensions", "skillPath", "defaultReads":
			var items []string
			if value != "" {
				items = strings.Split(value, ",")
			}
			indent := -1
			if value == "" {
				for i+1 < len(lines) && len(lines[i+1]) > 0 && (lines[i+1][0] == ' ' || lines[i+1][0] == '\t') {
					i++
					itemLine := lines[i]
					trimmed := strings.TrimLeft(itemLine, " ")
					n := len(itemLine) - len(trimmed)
					if n == 0 || !strings.HasPrefix(trimmed, "- ") || (indent >= 0 && indent != n) {
						return nil, fail(key, i+1, "only contiguous consistently space-indented list items are supported")
					}
					indent = n
					items = append(items, strings.TrimPrefix(trimmed, "- "))
				}
			}
			for _, item := range items {
				item = strings.TrimSpace(item)
				if item == "" || strings.EqualFold(item, "null") || item == "~" || strings.ContainsAny(item, "\"'[]{}#&*!|>,\t\r\n ") {
					return nil, fail(key, fieldLine, "invalid unquoted list token")
				}
				switch key {
				case "tools", "excludeTools":
					if strings.ContainsAny(item, "/\\") || strings.HasSuffix(item, ".js") || strings.HasSuffix(item, ".ts") || strings.HasPrefix(item, "mcp:") {
						return nil, fail(key, fieldLine, "executable module/path or static mcp: selector is unsupported")
					}
				case "skills":
					skills = append(skills, item)
				default:
					if !filepath.IsAbs(item) || filepath.Clean(item) != item || strings.ContainsAny(item, "~$\\") {
						return nil, fail(key, fieldLine, "resource reference must be an absolute path without expansion or dot segments")
					}
				}
			}
		case "inheritProjectContext", "inheritGlobalContext", "inheritSkills", "allowNestedSubagents", "async", "defaultProgress":
			if value != "true" && value != "false" {
				return nil, fail(key, fieldLine, "literal true or false required")
			}
		case "systemPromptMode", "defaultContext", "outputMode", "acceptanceRole":
			allowed := map[string]string{"systemPromptMode": "replace|append", "defaultContext": "fresh|fork", "outputMode": "inline|file-only", "acceptanceRole": "read-only|writer"}
			if !strings.Contains("|"+allowed[key]+"|", "|"+value+"|") || value == "" {
				return nil, fail(key, fieldLine, "unsupported enum value")
			}
		case "timeoutMs", "toolTimeoutMs", "maxSubagentDepth":
			for _, c := range value {
				if c < '0' || c > '9' {
					return nil, fail(key, fieldLine, "unquoted decimal integer required")
				}
			}
			n, err := strconv.ParseUint(value, 10, 31)
			if err != nil || (n == 0 && key != "maxSubagentDepth") {
				return nil, fail(key, fieldLine, "integer out of admitted range")
			}
		case "outputSchema", "toolBudget":
			if _, err := scan.PiJSONObject([]byte(value)); err != nil {
				return nil, fail(key, fieldLine, "single-line JSON object required: "+err.Error())
			}
		default:
			return nil, fail(key, fieldLine, "unsupported field")
		}
	}
	if !closed {
		return nil, fail("document", len(lines), "exact closing --- required")
	}
	for _, key := range []string{"name", "description"} {
		if _, ok := seen[key]; !ok {
			return nil, fail(key, 1, "required field absent")
		}
	}
	if seen["inheritGlobalContext"] == "true" && seen["inheritProjectContext"] != "true" {
		return nil, fail("inheritGlobalContext", 1, "global inheritance requires project inheritance")
	}
	return skills, nil
}

func piNativeString(s string) (string, error) {
	bad := func() (string, error) {
		return "", fmt.Errorf("nonempty single-line string without escapes, comments or YAML controls required")
	}
	if s == "" || strings.ContainsAny(s, "\\\r\n\t") {
		return bad()
	}
	if s[0] == '\'' || s[0] == '"' {
		q := s[0]
		if len(s) < 2 || s[len(s)-1] != q {
			return bad()
		}
		s = s[1 : len(s)-1]
		if strings.ContainsRune(s, rune(q)) {
			return bad()
		}
	} else if strings.ContainsAny(s, "\"'#[]{}&*!|>") || s == "~" || strings.EqualFold(s, "null") {
		return bad()
	}
	if strings.TrimSpace(s) == "" {
		return bad()
	}
	return s, nil
}
