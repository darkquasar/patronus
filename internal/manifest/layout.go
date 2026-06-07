package manifest

import "gopkg.in/yaml.v3"

// Layout is an adapter's per-kind on-disk transform rules (§5b). Each field is a
// pointer so "the tool does not handle this kind" (absent) is distinguishable
// from a present-but-empty block. The YAML is polymorphic — string|object|null
// per field — so several entry types implement custom UnmarshalYAML.
type Layout struct {
	Skill       *SkillLayout       `yaml:"Skill,omitempty"`
	Agent       *AgentLayout       `yaml:"Agent,omitempty"`
	Command     *CommandLayout     `yaml:"Command,omitempty"`
	Mcp         *McpLayout         `yaml:"Mcp,omitempty"`
	Hook        *HookLayout        `yaml:"Hook,omitempty"`
	Instruction *InstructionLayout `yaml:"Instruction,omitempty"`
}

// PathTarget is a layout entry that is EITHER a bare path string
// ("~/.claude/skills/{name}/SKILL.md") OR null (e.g. Codex Command.project).
// Set is true only when a non-null scalar was present; yaml.v3 leaves the field
// zero for both an explicit null and a missing key, so absent == null here
// (both mean "no usable target" — see OK).
type PathTarget struct {
	Path string
	Set  bool
}

// UnmarshalYAML accepts a scalar string. (yaml.v3 does not invoke this for an
// explicit null value, leaving the zero PathTarget.)
func (p *PathTarget) UnmarshalYAML(value *yaml.Node) error {
	if value.Tag == "!!null" {
		return nil
	}
	p.Set = true
	return value.Decode(&p.Path)
}

// OK reports whether this target carries a usable path.
func (p PathTarget) OK() bool { return p.Path != "" }

// FileTarget is a layout entry that is EITHER an object {file, format, path,
// action} OR null (e.g. Codex/OpenCode Hook). Used by Mcp/Hook/Instruction.
type FileTarget struct {
	File   string `yaml:"file"`
	Format string `yaml:"format,omitempty"` // json | jsonc | toml
	Path   string `yaml:"path,omitempty"`   // dotted, e.g. "mcp_servers.{name}"
	Action string `yaml:"action,omitempty"` // merge | appendSection
	Null   bool   `yaml:"-"`                // true when the entry was explicit null
}

// UnmarshalYAML accepts an object. (yaml.v3 does not invoke this for an explicit
// null value, leaving the zero FileTarget; check OK to test usability.)
func (t *FileTarget) UnmarshalYAML(value *yaml.Node) error {
	if value.Tag == "!!null" {
		t.Null = true
		return nil
	}
	type raw FileTarget // avoid recursion
	return value.Decode((*raw)(t))
}

// OK reports whether this target carries a usable file path.
func (t FileTarget) OK() bool { return t.File != "" }

// Frontmatter is either "passthrough" (scalar) or an allow-list of keys to keep
// (sequence), e.g. [mode, model, prompt, permission] for OpenCode agents.
type Frontmatter struct {
	Passthrough bool
	Allow       []string
}

// UnmarshalYAML accepts the scalar "passthrough" or a sequence of key names.
func (f *Frontmatter) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		var s string
		if err := value.Decode(&s); err != nil {
			return err
		}
		f.Passthrough = s == "passthrough"
		return nil
	}
	return value.Decode(&f.Allow)
}

// OrderedMap preserves declaration order and arbitrary key names for a YAML
// mapping decoded as string->string. This is the §9.9 fix: MCP transport key
// templates differ structurally per tool (Claude carries a literal
// type:"stdio"; Codex omits type entirely; OpenCode uses type:"local"), so the
// transport shape must be data, not a fixed field.
type OrderedMap struct {
	Keys []string
	Vals map[string]string
}

// UnmarshalYAML decodes the mapping node's content pairs in order.
func (m *OrderedMap) UnmarshalYAML(value *yaml.Node) error {
	m.Vals = make(map[string]string, len(value.Content)/2)
	for i := 0; i+1 < len(value.Content); i += 2 {
		k := value.Content[i].Value
		v := value.Content[i+1].Value
		m.Keys = append(m.Keys, k)
		m.Vals[k] = v
	}
	return nil
}

// SkillLayout describes where a Skill is written (SKILL.md passthrough).
type SkillLayout struct {
	Global      PathTarget  `yaml:"global"`
	Project     PathTarget  `yaml:"project"`
	NameSource  string      `yaml:"nameSource,omitempty"` // e.g. "dir"
	Frontmatter Frontmatter `yaml:"frontmatter,omitempty"`
	Required    []string    `yaml:"required,omitempty"`
}

// AgentLayout describes how a portable Agent is reshaped per tool.
type AgentLayout struct {
	Global      PathTarget  `yaml:"global"`
	Project     PathTarget  `yaml:"project"`
	BodyIs      string      `yaml:"bodyIs,omitempty"` // systemPrompt | developer_instructions | prompt
	Format      string      `yaml:"format,omitempty"` // toml (codex); markdown otherwise
	Frontmatter Frontmatter `yaml:"frontmatter,omitempty"`
	Required    []string    `yaml:"required,omitempty"`
}

// CommandLayout describes where a Command is written. project may be null.
type CommandLayout struct {
	Global  PathTarget `yaml:"global"`
	Project PathTarget `yaml:"project"`
}

// Transport is one MCP transport's key template (stdio or http).
type Transport struct {
	Keys *OrderedMap `yaml:"keys"`
}

// McpLayout describes the MERGE primitive for wiring MCP servers per tool.
type McpLayout struct {
	Global     FileTarget           `yaml:"global"`
	Project    FileTarget           `yaml:"project"`
	User       FileTarget           `yaml:"user"` // Claude only
	Transports map[string]Transport `yaml:"transports,omitempty"`
}

// HookLayout describes the MERGE primitive for hooks. Null for tools whose hook
// surface is not yet modeled (Codex, OpenCode today).
type HookLayout struct {
	Global  FileTarget `yaml:"global"`
	Project FileTarget `yaml:"project"`
}

// InstructionLayout describes the APPEND-section target for instructions.
type InstructionLayout struct {
	Global  FileTarget `yaml:"global"`
	Project FileTarget `yaml:"project"`
}

// ForScope returns the file/path target for the given scope ("global"|"local").
// Scope "local" maps to the Project field.
func (l *InstructionLayout) ForScope(scope string) FileTarget {
	if scope == "global" {
		return l.Global
	}
	return l.Project
}

// ForScope returns the path string for the given scope ("global"|"local").
func (l *SkillLayout) ForScope(scope string) PathTarget {
	if scope == "global" {
		return l.Global
	}
	return l.Project
}

// ForScope returns the path string for the given scope ("global"|"local").
func (l *AgentLayout) ForScope(scope string) PathTarget {
	if scope == "global" {
		return l.Global
	}
	return l.Project
}

// ForScope returns the path string for the given scope ("global"|"local").
func (l *CommandLayout) ForScope(scope string) PathTarget {
	if scope == "global" {
		return l.Global
	}
	return l.Project
}

// ForScope returns the file target for the given scope ("global"|"local").
func (l *McpLayout) ForScope(scope string) FileTarget {
	if scope == "global" {
		return l.Global
	}
	return l.Project
}
