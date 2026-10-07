package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

var _distributedLinks = regexp.MustCompile(`\]\(([^\s)]+)(?:\s+"[^"]*")?\)`)
var _authorPaths = regexp.MustCompile("(?m)(^|[\\s\"'`(=])(?:/(?:Users|home|root|workspace|workspaces|repo|repos)/|[A-Za-z]:[\\\\/](?:Users|repos)[\\\\/])[^\\s\"'`<>)]*")

type referenceException struct {
	Source       string   `yaml:"source"`
	Digest       string   `yaml:"manifest_content_sha256"`
	References   []string `yaml:"references"`
	Reason       string   `yaml:"reason"`
	UpstreamOnly bool     `yaml:"upstream_only"`
}

// Compatibility debt is explicit source data bound to exact manifest/body bytes.
// A changed artifact cannot inherit an old exception or silently suppress new links.
func (c *catalog) referenceExceptions(members map[string][]byte) (map[string]bool, error) {
	const file = "docs/compatibility/distributed-reference-exceptions.yaml"
	allowed := map[string]bool{}
	if _, err := os.Lstat(filepath.Join(c.root, file)); os.IsNotExist(err) {
		return allowed, nil
	}
	data, err := readRegular(c.root, file)
	if err != nil {
		return nil, err
	}
	var policy struct {
		Version    int                  `yaml:"schema_version"`
		Exceptions []referenceException `yaml:"exceptions"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&policy); err != nil {
		return nil, err
	}
	if policy.Version != 1 {
		return nil, fmt.Errorf("reference exceptions: unsupported schema")
	}
	for _, rule := range policy.Exceptions {
		body, ok := members[rule.Source]
		if !ok || rule.Reason == "" || len(rule.References) == 0 {
			return nil, fmt.Errorf("reference exception: unknown source or incomplete reason/references: %s", rule.Source)
		}
		manifest, err := readRegular(c.root, path.Join(path.Dir(rule.Source), "patronus.yaml"))
		if err != nil {
			return nil, err
		}
		digest := sha256.New()
		_, _ = digest.Write(manifest)
		_, _ = digest.Write(body)
		if fmt.Sprintf("%x", digest.Sum(nil)) != rule.Digest {
			return nil, fmt.Errorf("%s: stale reference exception; manifest/content changed", rule.Source)
		}
		for _, ref := range rule.References {
			key := rule.Source + "\x00" + ref
			if ref == "" || allowed[key] {
				return nil, fmt.Errorf("%s: empty/duplicate reference exception", rule.Source)
			}
			if !rule.UpstreamOnly {
				if _, err := regularPath(c.root, path.Join(path.Dir(rule.Source), ref), false); err != nil {
					return nil, fmt.Errorf("%s: legacy reference source: %w", rule.Source, err)
				}
			}
			allowed[key] = true
		}
	}
	return allowed, nil
}

func (c *catalog) checkDistributedResources() error {
	members := map[string][]byte{}
	for _, it := range c.items {
		if text(it.manifest, "family") != "artifact" {
			continue
		}
		rel, err := filepath.Rel(c.root, filepath.Dir(it.file))
		if err != nil {
			return err
		}
		for name, body := range it.payload {
			members[path.Join(filepath.ToSlash(rel), name)] = body
		}
	}
	allowed, err := c.referenceExceptions(members)
	if err != nil {
		return err
	}
	for name, body := range members {
		if match := _authorPaths.Find(body); match != nil {
			return fmt.Errorf("%s: author-machine absolute path %q", name, match)
		}
		if path.Ext(name) != ".md" {
			continue
		}
		var fence byte
		fenceLen := 0
		literalMarkdown := false
		for _, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
				n := 0
				for n < len(trimmed) && trimmed[n] == trimmed[0] {
					n++
				}
				if n >= 3 {
					if fenceLen == 0 {
						fence, fenceLen = trimmed[0], n
						info := strings.Fields(trimmed[n:])
						literalMarkdown = len(info) > 0 && (info[0] == "markdown" || info[0] == "md")
					} else if trimmed[0] == fence && n >= fenceLen && strings.TrimSpace(trimmed[n:]) == "" {
						fenceLen = 0
					}
					continue
				}
			}
			if fenceLen != 0 && literalMarkdown {
				continue
			}
			for _, match := range _distributedLinks.FindAllStringSubmatch(line, -1) {
				ref := strings.Trim(match[1], "<>")
				if strings.HasPrefix(ref, "#") || strings.Contains(ref, "://") || strings.HasPrefix(ref, "mailto:") || strings.Contains(ref, "<") || match[1] == "<relative-path>" {
					continue
				}
				ref, _, _ = strings.Cut(ref, "#")
				resolved := path.Join(path.Dir(name), ref)
				if relative, ok := strings.CutPrefix(name, "artifacts/skills/"); ok {
					skill, _, _ := strings.Cut(relative, "/")
					if local, ok := strings.CutPrefix(ref, "{skillDir}/"); ok {
						resolved = path.Join("artifacts/skills", skill, local)
					} else if sibling, ok := strings.CutPrefix(ref, "{skillsDir}/"); ok {
						resolved = path.Join("artifacts/skills", sibling)
					}
				}
				if _, ok := members[resolved]; !ok && !allowed[name+"\x00"+ref] {
					return fmt.Errorf("%s: missing distributed reference %q", name, ref)
				}
			}
		}
	}
	if err := c.checkAgentSelections(); err != nil {
		return err
	}
	return c.checkAgentOverlays()
}

// Native role overrides replace entire lists. Preserve the base role's ordered
// selection and require added skills to be declared Pi-compatible dependencies.
func (c *catalog) checkAgentOverlays() error {
	for name, it := range c.items {
		if text(it.manifest, "type") != "setting" {
			continue
		}
		targets, err := stringList(it.manifest["targets"])
		if err != nil {
			return err
		}
		setting := object(it.manifest, "setting")
		parts := strings.Split(text(setting, "path"), ".")
		if !contains(targets, "pi") || len(parts) != 4 || parts[0] != "subagents" || parts[1] != "agentOverrides" || (parts[3] != "skills" && parts[3] != "tools") {
			continue
		}
		role := c.items[parts[2]]
		if role == nil || text(role.manifest, "type") != "agent" {
			return fmt.Errorf("%s: override refers to missing native agent %s", name, parts[2])
		}
		raw := bytes.ReplaceAll(role.payload[text(role.manifest, "entry")], []byte("\r\n"), []byte("\n"))
		if !bytes.HasPrefix(raw, []byte("---\n")) {
			return fmt.Errorf("%s: missing agent frontmatter", parts[2])
		}
		end := bytes.Index(raw[4:], []byte("\n---"))
		if end < 0 {
			return fmt.Errorf("%s: unterminated agent frontmatter", parts[2])
		}
		front, err := decodeYAML(raw[4 : 4+end])
		if err != nil {
			return err
		}
		var base []string
		for _, value := range strings.Split(text(front, parts[3]), ",") {
			if value = strings.TrimSpace(value); value != "" {
				base = append(base, value)
			}
		}
		values, err := stringList(setting["value"])
		if err != nil {
			return fmt.Errorf("%s: override list: %w", name, err)
		}
		if len(values) < len(base) || !slices.Equal(values[:len(base)], base) {
			return fmt.Errorf("%s: override must preserve the complete ordered %s list of %s", name, parts[3], parts[2])
		}
		if parts[3] == "tools" {
			continue
		}
		requires, err := stringList(it.manifest["requires"])
		if err != nil {
			return err
		}
		for _, selected := range values[len(base):] {
			skill := c.items[selected]
			if !contains(requires, selected) || skill == nil || text(skill.manifest, "type") != "skill" {
				return fmt.Errorf("%s: added skill %q must be a declared skill dependency", name, selected)
			}
			skillTargets, err := stringList(skill.manifest["targets"])
			if err != nil {
				return err
			}
			if len(skillTargets) != 0 && !contains(skillTargets, "pi") {
				return fmt.Errorf("%s: added skill %q is not Pi-compatible", name, selected)
			}
		}
	}
	return nil
}

func (c *catalog) checkAgentSelections() error {
	for name, it := range c.items {
		m := it.manifest
		if text(m, "type") != "agent" {
			continue
		}
		targets, err := stringList(m["targets"])
		if err != nil {
			return err
		}
		if !contains(targets, "pi") {
			continue
		}
		files, err := stringList(m["files"])
		if err != nil {
			return err
		}
		if len(targets) != 1 || len(files) != 0 || len(object(m, "overrides")) != 0 {
			return fmt.Errorf("%s: native agent requires sole Pi target and no sidecars/overrides", name)
		}
		raw := bytes.ReplaceAll(it.payload[text(m, "entry")], []byte("\r\n"), []byte("\n"))
		if !bytes.HasPrefix(raw, []byte("---\n")) {
			return fmt.Errorf("%s: missing agent frontmatter", name)
		}
		end := bytes.Index(raw[4:], []byte("\n---"))
		if end < 0 {
			return fmt.Errorf("%s: unterminated agent frontmatter", name)
		}
		front, err := decodeYAML(raw[4 : 4+end])
		if err != nil {
			return err
		}
		if text(front, "name") != name {
			return fmt.Errorf("%s: agent frontmatter identity mismatch", name)
		}
		requires, err := stringList(m["requires"])
		if err != nil {
			return err
		}
		selection, ok := front["skills"].(string)
		if front["skills"] != nil && !ok {
			return fmt.Errorf("%s: native agent skills must be a literal comma-separated string", name)
		}
		for _, selected := range strings.Split(selection, ",") {
			selected = strings.TrimSpace(selected)
			if selected == "" && selection == "" {
				continue
			}
			if !safeComponent(selected) || !contains(requires, selected) {
				return fmt.Errorf("%s: selected skill %q not declared in requires", name, selected)
			}
			skill := c.items[selected]
			if skill == nil || text(skill.manifest, "family") != "artifact" || text(skill.manifest, "type") != "skill" {
				return fmt.Errorf("%s: selected resource %q is not a Pi-compatible skill", name, selected)
			}
			skillTargets, err := stringList(skill.manifest["targets"])
			if err != nil {
				return err
			}
			if len(skillTargets) != 0 && !contains(skillTargets, "pi") {
				return fmt.Errorf("%s: selected resource %q is not a Pi-compatible skill", name, selected)
			}
		}
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
