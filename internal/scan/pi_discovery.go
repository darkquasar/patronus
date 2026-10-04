package scan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/darkquasar/patronus/internal/nativepi"
	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// PiReadFile is injected for effective-source inspection; it never executes a loader.
type PiReadFile func(string) ([]byte, bool, error)

// PiSafePath rejects aliases at every existing component, including the root.
// Missing final components are allowed for a prospective destination.
func PiSafePath(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("pi path %q must be canonical and absolute", path)
	}
	for p := path; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("pi path %s: %w", p, err)
		}
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("pi path %s: unapproved symlink/redirection", p)
			}
			if p != path && !info.IsDir() {
				return fmt.Errorf("pi path %s: non-directory ancestor", p)
			}
			if p == path && !info.IsDir() && !info.Mode().IsRegular() {
				return fmt.Errorf("pi path %s: nonregular input", p)
			}
			if info.Mode().Perm()&0444 == 0 {
				return fmt.Errorf("pi path %s: unreadable input", p)
			}
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return nil
}

func PiSafeName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\:\x00\r\n")
}

// PiSourcePath validates declared entry/sidecar containment before any read.
func PiSourcePath(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || filepath.Clean(relative) != relative || relative == ".." || strings.HasPrefix(relative, "../") || strings.Contains(relative, "\\") {
		return "", fmt.Errorf("pi source %q escapes or redirects selected root %s", relative, root)
	}
	path := filepath.Join(root, relative)
	if err := PiSafePath(path); err != nil {
		return "", err
	}
	return path, nil
}

func ReadPiFile(path string) ([]byte, bool, error) {
	if err := PiSafePath(path); err != nil {
		return nil, false, err
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("pi source %s: not a regular file", path)
	}
	b, err := os.ReadFile(path)
	return b, true, err
}

// PiJSONObject deliberately rejects duplicate members at all depths, unlike
// encoding/json's last-wins map decoding. It is shared by native JSON fields
// and read-only configuration inventory.
func PiJSONObject(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	v, err := piJSONValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing JSON data")
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected JSON object")
	}
	return obj, nil
}
func piJSONValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return tok, nil
	}
	switch delim {
	case '{':
		out := map[string]any{}
		for dec.More() {
			token, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, ok := token.(string)
			if !ok {
				return nil, fmt.Errorf("invalid JSON key")
			}
			if _, ok := out[key]; ok {
				return nil, fmt.Errorf("duplicate JSON member %q", key)
			}
			value, err := piJSONValue(dec)
			if err != nil {
				return nil, err
			}
			out[key] = value
		}
		_, err := dec.Token()
		return out, err
	case '[':
		var out []any
		for dec.More() {
			v, err := piJSONValue(dec)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		_, err := dec.Token()
		return out, err
	default:
		return nil, fmt.Errorf("unexpected JSON delimiter")
	}
}

// PiContext is the effective first-readable instruction in one directory.
// Candidates records lower-priority files too, for cross-harness diagnostics.
type PiContext struct {
	Path       string
	Candidates []string
	Mixed      bool
	Warning    string
}

func DiscoverPiContext(dir string, read PiReadFile) (PiContext, error) {
	result := PiContext{Path: filepath.Join(dir, "AGENTS.md")}
	for _, name := range []string{"AGENTS.override.md", "AGENTS.md", "AGENTS.MD", "CLAUDE.md", "CLAUDE.MD"} {
		path := filepath.Join(dir, name)
		if err := PiSafePath(path); err != nil {
			return result, err
		}
		raw, exists, err := read(path)
		if err != nil {
			return result, fmt.Errorf("pi context candidate %s: %w", path, err)
		}
		if !exists {
			continue
		}
		result.Candidates = append(result.Candidates, path)
		if len(result.Candidates) == 1 {
			result.Path = path
			result.Mixed = strings.HasPrefix(name, "CLAUDE") || piOtherSections(raw)
			if name == "AGENTS.override.md" {
				result.Warning = "Pi discovers higher-priority AGENTS.override.md: " + path
			}
		}
	}
	// A separate Claude file is applicable to Claude even when Pi reads AGENTS.
	for _, path := range result.Candidates {
		if strings.HasPrefix(filepath.Base(path), "CLAUDE") {
			result.Mixed = true
		}
	}
	return result, nil
}
func piOtherSections(raw []byte) bool {
	rest := string(raw)
	for {
		_, tail, ok := strings.Cut(rest, "<!-- patronus:start ")
		if !ok {
			return false
		}
		name, after, ok := strings.Cut(tail, " -->")
		if !ok {
			return true
		}
		if !strings.HasPrefix(name, "pi:") {
			return true
		}
		rest = after
	}
}

// PiResource records exact discovered identities and their source, not inferred
// functional equivalence. Order is loader precedence, never permission to shadow.
type PiResource struct {
	Kind, Name, Path, Origin string
	Aliases                  []string
}
type PiResourceRoot struct {
	Kind, Path, Origin string
	AgentsSkills       bool // pinned .agents skill collection differs from Pi's native directory
}

// DiscoverPiResources inventories explicit qualified roots. Pi skills/prompts
// use package-before-native first-wins; subagents use builtin/package/user/project
// last-wins (package roots first-wins internally). All duplicates are conflicts.
func DiscoverPiResources(roots []PiResourceRoot) ([]PiResource, error) {
	var out []PiResource
	for _, root := range roots {
		if err := PiSafePath(root.Path); err != nil {
			return nil, err
		}
		err := filepath.WalkDir(root.Path, func(path string, d fs.DirEntry, err error) error {
			if os.IsNotExist(err) && path == root.Path {
				return nil
			}
			if err != nil {
				return err
			}
			if err := PiSafePath(path); err != nil {
				return err
			}
			if d.IsDir() {
				if root.Kind == "command" && path == root.Path {
					for _, ignore := range []string{".gitignore", ".ignore", ".fdignore"} {
						raw, exists, err := ReadPiFile(filepath.Join(path, ignore))
						if err != nil {
							return err
						}
						if exists && len(bytes.TrimSpace(raw)) > 0 {
							return fmt.Errorf("pi prompt source %s: ignore rules require qualified discovery", filepath.Join(path, ignore))
						}
					}
				}
				if path != root.Path {
					if root.Kind == "command" || d.Name() == "node_modules" {
						return filepath.SkipDir
					}
					if root.Kind == "skill" && strings.HasPrefix(d.Name(), ".") {
						return filepath.SkipDir
					}
					if root.Kind == "agent" {
						if d.Name() == ".git" || d.Name() == ".pi" || d.Name() == "sync-backups" || (strings.EqualFold(filepath.Base(filepath.Dir(path)), ".agents") && strings.EqualFold(d.Name(), "skills")) {
							return filepath.SkipDir
						}
						for _, marker := range []string{".git", ".pi", ".agents"} {
							if _, err := os.Stat(filepath.Join(path, marker)); err == nil {
								return filepath.SkipDir
							} else if !os.IsNotExist(err) {
								return err
							}
						}
					}
				}
				if root.Kind == "skill" {
					for _, ignore := range []string{".gitignore", ".ignore", ".fdignore"} {
						raw, exists, err := ReadPiFile(filepath.Join(path, ignore))
						if err != nil {
							return err
						}
						if exists && len(bytes.TrimSpace(raw)) > 0 {
							return fmt.Errorf("pi skill source %s: ignore rules require qualified discovery", filepath.Join(path, ignore))
						}
					}
					file := filepath.Join(path, "SKILL.md")
					if _, exists, err := ReadPiFile(file); err != nil {
						return err
					} else if exists {
						r, err := piResourceFile(root, file)
						if err != nil {
							return err
						}
						if r.Name != "" {
							out = append(out, r)
						}
						return filepath.SkipDir
					}
				}
				return nil
			}
			if root.Kind == "skill" && d.Name() != "SKILL.md" {
				atRoot := filepath.Dir(path) == root.Path
				if !strings.HasSuffix(path, ".md") || (root.AgentsSkills && atRoot) || (!root.AgentsSkills && !atRoot) {
					return nil
				}
			}
			if root.Kind != "skill" && (!strings.HasSuffix(d.Name(), ".md") || (root.Kind == "agent" && strings.HasSuffix(d.Name(), ".chain.md"))) {
				return nil
			}
			r, err := piResourceFile(root, path)
			if err != nil {
				return err
			}
			if r.Name != "" {
				out = append(out, r)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for i, r := range out {
		if err := PiResourceConflict(out[:i], r); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func piResourceFile(root PiResourceRoot, path string) (PiResource, error) {
	raw, _, err := ReadPiFile(path)
	if err != nil {
		return PiResource{}, err
	}
	var fields map[string]any
	if root.Kind == "agent" {
		fields, err = piNativeAgentFields(raw)
	} else {
		fields, err = piMarkdownFields(raw)
	}
	if err != nil {
		return PiResource{}, fmt.Errorf("pi discovery %s: %w", path, err)
	}
	name := strings.TrimSuffix(filepath.Base(path), ".md")
	if root.Kind != "command" {
		description, _ := fields["description"].(string)
		if description == "" {
			return PiResource{}, nil
		}
		declared, _ := fields["name"].(string)
		if root.Kind == "agent" && declared == "" {
			return PiResource{}, nil
		}
		if root.Kind == "skill" {
			name = filepath.Base(filepath.Dir(path))
		}
		if declared != "" {
			name = declared
		}
		if root.Kind == "agent" && nonemptyPiValue(fields["package"]) {
			return PiResource{}, fmt.Errorf("pi agent source %s: package identity requires qualified name resolution", path)
		}
	}
	if !PiSafeName(name) {
		return PiResource{}, fmt.Errorf("pi discovery invalid identity at %s", path)
	}
	var aliases []string
	if root.Kind == "agent" {
		aliases, err = piNativeAgentAliases(fields, name)
		if err != nil {
			return PiResource{}, fmt.Errorf("pi agent source %s: %w", path, err)
		}
	}
	return PiResource{Kind: root.Kind, Name: name, Aliases: aliases, Path: path, Origin: root.Origin}, nil
}

// piNativeAgentAliases mirrors the native loader's alias selection and finite
// list forms without decoding frontmatter as general YAML. Plural metadata has
// precedence; scalar and block-list lines are comma-separated and filter empties.
func piNativeAgentAliases(fields map[string]any, primary string) ([]string, error) {
	value, ok := fields["aliases"]
	if !ok {
		value = fields["alias"]
	}
	if value == nil {
		return nil, nil
	}
	raw, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("aliases must use scalar or finite block-list metadata")
	}
	seen := map[string]bool{primary: true}
	var aliases []string
	for _, line := range strings.Split(raw, "\n") {
		for _, part := range strings.Split(line, ",") {
			alias := strings.TrimSpace(part)
			if alias == "" {
				continue
			}
			if strings.HasPrefix(alias, "[") || strings.HasPrefix(alias, "{") || strings.HasSuffix(alias, "]") || strings.HasSuffix(alias, "}") || !PiSafeName(alias) {
				return nil, fmt.Errorf("invalid alias %q", alias)
			}
			if !seen[alias] {
				seen[alias] = true
				aliases = append(aliases, alias)
			}
		}
	}
	return aliases, nil
}

// piNativeAgentFields reads only the finite metadata needed for static
// identities. Native simple values are strings, not YAML scalars: true, numbers
// and embedded colons must not disappear through YAML coercion. Alias block
// lists use one consistent indentation and list markers; other metadata blocks
// and nested structures fail closed. Assignments match the loader's last-wins
// behavior. Artifact admission and full-document preservation remain adapter-owned.
func piNativeAgentFields(raw []byte) (map[string]any, error) {
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if !strings.HasPrefix(lines[0], "---") {
		return map[string]any{}, nil
	}
	if lines[0] != "---" {
		return nil, fmt.Errorf("unsupported native frontmatter delimiter")
	}
	fields := map[string]any{}
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "---" {
			return fields, nil
		}
		if strings.HasPrefix(line, "---") {
			return nil, fmt.Errorf("unsupported native frontmatter delimiter")
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch key {
		case "name", "description", "package", "alias", "aliases":
		default:
			continue
		}
		value = strings.TrimSpace(value)
		quoted := len(value) >= 2 && ((value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"'))
		if quoted {
			value = value[1 : len(value)-1]
		}
		if value == "" && (key == "alias" || key == "aliases") {
			var err error
			value, i, err = piNativeAliasBlock(lines, i)
			if err != nil {
				return nil, fmt.Errorf("unsupported native metadata block for %s: %w", key, err)
			}
			fields[key] = value
			continue
		}
		if value == "" || (!quoted && (value == ">" || value == ">-" || value == "|" || value == "|-")) {
			return nil, fmt.Errorf("unsupported native metadata block for %s; qualify identity before apply", key)
		}
		fields[key] = value
	}
	return nil, fmt.Errorf("malformed native frontmatter")
}

func piNativeAliasBlock(lines []string, keyLine int) (string, int, error) {
	var (
		indent string
		values []string
	)
	i := keyLine + 1
	for ; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			break
		}
		width := len(line) - len(strings.TrimLeft(line, " \t"))
		if width == 0 {
			break
		}
		currentIndent := line[:width]
		if indent == "" {
			indent = currentIndent
		}
		if currentIndent != indent {
			return "", keyLine, fmt.Errorf("inconsistent or nested indentation")
		}
		item := line[width:]
		if len(item) < 3 || item[0] != '-' || (item[1] != ' ' && item[1] != '\t') || strings.TrimSpace(item[2:]) == "" {
			return "", keyLine, fmt.Errorf("expected nonempty list item")
		}
		values = append(values, strings.TrimSpace(item[2:]))
	}
	return strings.Join(values, "\n"), i - 1, nil
}

// PiMarkdownName only reads metadata; native agent admission is adapter-owned.
func PiMarkdownName(raw []byte) (string, error) {
	fields, err := piMarkdownFields(raw)
	if err != nil {
		return "", err
	}
	if v, ok := fields["name"]; ok {
		s, ok := v.(string)
		if !ok || s == "" {
			return "", fmt.Errorf("name must be nonempty string")
		}
		return s, nil
	}
	return "", nil
}

func piMarkdownFields(raw []byte) (map[string]any, error) {
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return map[string]any{}, nil
	}
	end := 1
	for end < len(lines) && lines[end] != "---" {
		end++
	}
	if end == len(lines) {
		return nil, fmt.Errorf("malformed Markdown frontmatter")
	}
	var fields map[string]any
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func PiResourceConflict(resources []PiResource, selected PiResource) error {
	selectedNames := append([]string{selected.Name}, selected.Aliases...)
	for _, other := range resources {
		if other.Kind != selected.Kind || other.Path == selected.Path {
			continue
		}
		otherNames := append([]string{other.Name}, other.Aliases...)
		for _, selectedName := range selectedNames {
			for _, otherName := range otherNames {
				if otherName == selectedName {
					return fmt.Errorf("pi %s identity %q conflict: %s (%s) and %s (%s); silent shadowing refused", selected.Kind, selectedName, other.Path, other.Origin, selected.Path, selected.Origin)
				}
			}
		}
	}
	return nil
}

// PiMCPSource inventory uses the selected 3.0.0 loader order, low to high.
type PiMCPSource struct {
	Path    string
	Origin  string
	Servers map[string]any
}

func DiscoverPiMCP(home, project, agentRoot, destination string, env EnvLookup, read PiReadFile) ([]PiMCPSource, error) {
	if err := piDefaultBranding(env, read); err != nil {
		return nil, err
	}
	paths := []string{filepath.Join(home, ".config/mcp/mcp.json"), filepath.Join(home, ".agents/mcp.json"), filepath.Join(home, ".agents/mcp/mcp.json"), filepath.Join(agentRoot, "mcp-adapter.json")}
	mode, _ := env("PI_MCP_CONFIG_MODE")
	if mode != "" && strings.ToLower(strings.TrimSpace(mode)) != "exclusive" {
		return nil, fmt.Errorf("unknown PI_MCP_CONFIG_MODE")
	}
	exclusive := strings.ToLower(strings.TrimSpace(mode)) == "exclusive"
	if exclusive {
		paths = paths[3:]
		if destination != paths[0] {
			return nil, fmt.Errorf("pi MCP exclusive path override mismatch: %s versus %s", paths[0], destination)
		}
	}
	var out []PiMCPSource
	seen := map[string]string{}
	addSource := func(source PiMCPSource) error {
		origin := source.Path
		if source.Origin != "" {
			origin += " (" + source.Origin + ")"
		}
		var names []string
		for name := range source.Servers {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			normalized := strings.ReplaceAll(name, "-", "_")
			if old, ok := seen[normalized]; ok {
				return fmt.Errorf("pi MCP duplicate normalized server %q: %s and %s", name, old, origin)
			}
			seen[normalized] = origin
		}
		out = append(out, source)
		return nil
	}
	importedPaths := map[string]bool{}
	inspect := func(path string) (map[string]any, error) {
		raw, exists, err := read(path)
		if err != nil {
			return nil, fmt.Errorf("pi MCP source %s: %w", path, err)
		}
		if !exists {
			return map[string]any{}, nil
		}
		doc, err := piMCPJSONObject(raw)
		if err != nil {
			return nil, fmt.Errorf("pi MCP source %s: %w", path, err)
		}
		if nonemptyPiValue(doc["claudePlugins"]) {
			return nil, fmt.Errorf("pi MCP source %s: active claudePlugins structured schema is unsupported", path)
		}
		if value, declared := doc["imports"]; declared {
			imports, err := piMCPImports(home, project, path, value, read)
			if err != nil {
				return nil, err
			}
			for _, source := range imports {
				if importedPaths[source.Path] {
					continue
				}
				if err := addSource(source); err != nil {
					return nil, err
				}
				importedPaths[source.Path] = true
			}
		}
		settings, _ := doc["settings"].(map[string]any)
		if value, present := doc["settings"]; present && value != nil && settings == nil {
			return nil, fmt.Errorf("pi MCP source %s: settings must be an object", path)
		}
		if !exclusive {
			if nonemptyPiValue(settings["agentPluginPaths"]) {
				return nil, fmt.Errorf("pi MCP source %s: active agentPluginPaths discovery is not statically qualified", path)
			}
			if value := settings["hostConfigDiscovery"]; value != nil && value != "off" && value != "prompt" {
				return nil, fmt.Errorf("pi MCP source %s: active or unknown hostConfigDiscovery needs qualified inspection", path)
			}
		}
		servers, err := piMCPServers(doc, path, "mcpServers", "mcp-servers")
		if err != nil {
			return nil, err
		}
		if err := addSource(PiMCPSource{Path: path, Servers: servers}); err != nil {
			return nil, err
		}
		return settings, nil
	}
	if !exclusive {
		defaults, err := piMCPPackageDefaults(home, project, agentRoot, read)
		if err != nil {
			return nil, err
		}
		for _, source := range defaults {
			if err := addSource(source); err != nil {
				return nil, err
			}
		}
	}
	var ancestorRoots any
	for _, path := range paths {
		settings, err := inspect(path)
		if err != nil {
			return nil, err
		}
		if roots, ok := settings["ancestorConfigRoots"]; ok {
			ancestorRoots = roots
		}
	}
	if exclusive {
		return out, nil
	}
	if ancestorRoots != nil {
		roots, ok := ancestorRoots.([]any)
		if !ok {
			return nil, fmt.Errorf("pi MCP ancestorConfigRoots must be a list")
		}
		closest := ""
		for _, v := range roots {
			root, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("pi MCP ancestor root must be a path")
			}
			if strings.HasPrefix(root, "~/") {
				root = filepath.Join(home, root[2:])
			}
			if err := PiSafePath(root); err != nil {
				return nil, err
			}
			info, err := os.Stat(root)
			if err != nil || !info.IsDir() {
				return nil, fmt.Errorf("pi MCP ancestor root must be an existing directory: %s", root)
			}
			if !piWithin(home, root) {
				return nil, fmt.Errorf("pi MCP ancestor root outside home: %s", root)
			}
			if piWithin(root, project) && len(root) > len(closest) {
				closest = root
			}
		}
		var dirs []string
		if closest != "" {
			for dir := filepath.Dir(project); piWithin(closest, dir); dir = filepath.Dir(dir) {
				dirs = append([]string{dir}, dirs...)
				if dir == closest {
					break
				}
			}
		}
		for _, dir := range dirs {
			for _, path := range []string{filepath.Join(dir, ".mcp.json"), filepath.Join(dir, ".pi/mcp-adapter.json")} {
				if _, err := inspect(path); err != nil {
					return nil, err
				}
			}
		}
	}
	for _, path := range []string{filepath.Join(project, ".mcp.json"), filepath.Join(project, ".pi/mcp-adapter.json")} {
		if _, err := inspect(path); err != nil {
			return nil, err
		}
	}
	return out, nil
}
func piMCPPackageDefaults(home, project, agentRoot string, read PiReadFile) ([]PiMCPSource, error) {
	var out []PiMCPSource
	seenRoots := map[string]bool{}
	for _, base := range []string{filepath.Join(project, ".pi"), agentRoot} {
		settingsPath := filepath.Join(base, "settings.json")
		raw, exists, err := read(settingsPath)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		doc, err := piMCPJSONObject(raw)
		if err != nil {
			return nil, fmt.Errorf("pi MCP settings source %s: %w", settingsPath, err)
		}
		value, declared := doc["packages"]
		if !declared {
			continue
		}
		entries, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("pi MCP source %s: packages must be a list", settingsPath)
		}
		for _, entry := range entries {
			source, ok := entry.(string)
			if !ok {
				if obj, valid := entry.(map[string]any); valid {
					source, ok = obj["source"].(string)
				}
			}
			if !ok || source == "" {
				return nil, fmt.Errorf("pi MCP source %s: invalid package source", settingsPath)
			}
			if strings.HasPrefix(source, "~") || strings.HasPrefix(source, "file:") {
				return nil, fmt.Errorf("pi MCP source %s: package path has unqualified host/MCP spelling", settingsPath)
			}
			root, err := piPackageRoot(home, base, source)
			if err != nil {
				return nil, err
			}
			if seenRoots[root] {
				continue
			}
			seenRoots[root] = true
			manifestPath := filepath.Join(root, "package.json")
			raw, exists, err := read(manifestPath)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, fmt.Errorf("pi MCP package source %s: missing active manifest", manifestPath)
			}
			manifest, err := piMCPJSONObject(raw)
			if err != nil {
				return nil, fmt.Errorf("pi MCP package source %s: %w", manifestPath, err)
			}
			pi, _ := manifest["pi"].(map[string]any)
			if manifest["pi"] != nil && pi == nil {
				return nil, fmt.Errorf("pi MCP package source %s: pi must be an object", manifestPath)
			}
			declaration, declared := pi["mcp"]
			if !declared {
				continue
			}
			name, ok := manifest["name"].(string)
			if !ok || name == "" {
				return nil, fmt.Errorf("pi MCP package source %s: pi.mcp requires a package name", manifestPath)
			}
			paths, ok := declaration.([]any)
			if path, isString := declaration.(string); isString {
				paths = []any{path}
				ok = true
			}
			if !ok {
				return nil, fmt.Errorf("pi MCP package source %s: pi.mcp must be a path or path list", manifestPath)
			}
			for _, value := range paths {
				relative, ok := value.(string)
				if !ok || relative == "" || strings.ContainsAny(relative, "*?![]") {
					return nil, fmt.Errorf("pi MCP package source %s: unsupported pi.mcp path", manifestPath)
				}
				// Package configs are contained data files, never executable imports.
				path, err := PiSourcePath(root, strings.TrimPrefix(relative, "./"))
				if err != nil {
					return nil, err
				}
				raw, exists, err := read(path)
				if err != nil {
					return nil, err
				}
				if !exists {
					return nil, fmt.Errorf("pi MCP package source %s: missing pi.mcp file %s", manifestPath, path)
				}
				doc, err := piMCPJSONObject(raw)
				if err != nil {
					return nil, fmt.Errorf("pi MCP package source %s: %w", path, err)
				}
				if _, exists := doc["mcpServers"]; !exists {
					return nil, fmt.Errorf("pi MCP package source %s: missing mcpServers object", path)
				}
				servers, err := piMCPServers(doc, path, "mcpServers")
				if err != nil {
					return nil, err
				}
				prefixed := map[string]any{}
				for server, value := range servers {
					normalized := piMCPPackageName(name, "package") + "__" + piMCPPackageName(server, "server")
					if _, exists := prefixed[normalized]; exists {
						return nil, fmt.Errorf("pi MCP duplicate normalized package server in %s", path)
					}
					prefixed[normalized] = value
				}
				out = append(out, PiMCPSource{Path: path, Origin: "pi.mcp declared by " + manifestPath, Servers: prefixed})
			}
		}
	}
	return out, nil
}

func piMCPPackageName(name, fallback string) string {
	var out strings.Builder
	invalid := false
	for _, r := range name {
		valid := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-'
		if valid {
			out.WriteRune(r)
		} else if !invalid {
			out.WriteByte('_')
		}
		invalid = !valid
	}
	if name := strings.Trim(out.String(), "_-"); name != "" {
		return name
	}
	return fallback
}

// The pinned MCP loader expands only these host kinds, not arbitrary filenames
// or recursive imports in host documents. Inspect the same first-existing
// candidates; OpenCode alone merges its global and nearest project config.
func piMCPImports(home, project, origin string, value any, read PiReadFile) ([]PiMCPSource, error) {
	entries, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("pi MCP source %s: imports must be a list", origin)
	}
	var out []PiMCPSource
	for _, entry := range entries {
		kind, ok := entry.(string)
		if !ok {
			return nil, fmt.Errorf("pi MCP source %s: import kind must be a string", origin)
		}
		var paths, keys []string
		switch kind {
		case "cursor":
			paths = []string{filepath.Join(home, ".cursor/mcp.json")}
			keys = []string{"mcpServers", "mcp-servers"}
		case "windsurf":
			paths = []string{filepath.Join(home, ".windsurf/mcp.json")}
			keys = []string{"mcpServers", "mcp-servers"}
		case "vscode":
			paths = []string{filepath.Join(project, ".vscode/mcp.json")}
			keys = []string{"mcpServers", "mcp-servers"}
		case "claude-code":
			paths = []string{filepath.Join(home, ".claude/mcp.json"), filepath.Join(home, ".claude.json"), filepath.Join(home, ".claude/claude_desktop_config.json")}
			keys = []string{"mcpServers"}
		case "claude-desktop":
			paths = []string{filepath.Join(home, "Library/Application Support/Claude/claude_desktop_config.json")}
			keys = []string{"mcpServers"}
		case "codex":
			paths = []string{filepath.Join(home, ".codex/config.toml"), filepath.Join(home, ".codex/config.json")}
			keys = []string{"mcp_servers", "mcpServers"}
		case "opencode":
			local, err := piOpenCodeProjectConfig(project)
			if err != nil {
				return nil, err
			}
			paths = []string{filepath.Join(home, ".config/opencode/opencode.json"), local}
		default:
			return nil, fmt.Errorf("pi MCP source %s: unsupported static import kind %q", origin, kind)
		}
		found := false
		for _, path := range paths {
			raw, exists, err := read(path)
			if err != nil {
				return nil, fmt.Errorf("pi MCP import declared by %s: %s: %w", origin, path, err)
			}
			if !exists {
				continue
			}
			var doc map[string]any
			if strings.HasSuffix(path, ".toml") {
				if err := toml.Unmarshal(raw, &doc); err != nil {
					return nil, fmt.Errorf("pi MCP import %s declared by %s: malformed TOML", path, origin)
				}
			} else {
				doc, err = piMCPJSONObject(raw)
				if err != nil {
					return nil, fmt.Errorf("pi MCP import %s declared by %s: %w", path, origin, err)
				}
			}
			var servers map[string]any
			if kind == "opencode" {
				servers, err = piOpenCodeMCPServers(doc, path)
			} else {
				servers, err = piMCPServers(doc, path, keys...)
			}
			if err != nil {
				return nil, err
			}
			out = append(out, PiMCPSource{Path: path, Origin: fmt.Sprintf("import %s declared by %s", kind, origin), Servers: servers})
			found = true
			if kind != "opencode" {
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("pi MCP source %s: import %s has no readable static candidate (%s)", origin, kind, strings.Join(paths, ", "))
		}
	}
	return out, nil
}

func piOpenCodeProjectConfig(project string) (string, error) {
	gitRoot := ""
	for dir := project; ; dir = filepath.Dir(dir) {
		marker := filepath.Join(dir, ".git")
		if err := PiSafePath(marker); err != nil {
			return "", err
		}
		if _, err := os.Stat(marker); err == nil {
			gitRoot = dir
			break
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	if gitRoot == "" {
		return filepath.Join(project, "opencode.json"), nil
	}
	for dir := project; ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, "opencode.json")
		if err := PiSafePath(path); err != nil {
			return "", err
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if dir == gitRoot {
			return path, nil
		}
	}
}

func piMCPServers(doc map[string]any, path string, keys ...string) (map[string]any, error) {
	for _, key := range keys {
		value, exists := doc[key]
		if !exists {
			continue
		}
		servers, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("pi MCP source %s: %s must be an object", path, key)
		}
		for name, entry := range servers {
			if _, ok := entry.(map[string]any); !ok {
				return nil, fmt.Errorf("pi MCP source %s: server %q must be an object", path, name)
			}
		}
		return servers, nil
	}
	return map[string]any{}, nil
}

func piOpenCodeMCPServers(doc map[string]any, path string) (map[string]any, error) {
	value, exists := doc["mcp"]
	if !exists {
		return map[string]any{}, nil
	}
	mcp, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("pi MCP import %s: mcp must be an object", path)
	}
	entries := map[string]any{}
	for name, entry := range mcp {
		if name != "servers" && name != "timeout" {
			entries[name] = entry
		}
	}
	if value, exists := mcp["servers"]; exists {
		servers, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("pi MCP import %s: mcp.servers must be an object", path)
		}
		for name, entry := range servers {
			if _, exists := entries[name]; exists {
				return nil, fmt.Errorf("pi MCP duplicate OpenCode server %q in %s", name, path)
			}
			entries[name] = entry
		}
	}
	out := map[string]any{}
	for name, value := range entries {
		entry, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("pi MCP import %s: server %q must be an object", path, name)
		}
		if entry["enabled"] == false || entry["disabled"] == true {
			continue
		}
		switch entry["type"] {
		case "local":
			command, ok := entry["command"].([]any)
			if !ok || len(command) == 0 {
				return nil, fmt.Errorf("pi MCP import %s: server %q requires a static local command list", path, name)
			}
			for _, value := range command {
				if _, ok := value.(string); !ok {
					return nil, fmt.Errorf("pi MCP import %s: server %q has a malformed command list", path, name)
				}
			}
		case "remote":
			if url, ok := entry["url"].(string); !ok || url == "" {
				return nil, fmt.Errorf("pi MCP import %s: server %q requires a remote URL", path, name)
			}
		default:
			return nil, fmt.Errorf("pi MCP import %s: server %q has an unsupported static type", path, name)
		}
		out[name] = entry
	}
	return out, nil
}

// Preserve string bytes while removing only JSONC comments/trailing commas.
// Duplicate-member rejection remains shared with the strict native JSON reader.
func piMCPJSONObject(raw []byte) (map[string]any, error) {
	clean := append([]byte(nil), raw...)
	quoted, escaped := false, false
	for i := 0; i < len(clean); i++ {
		c := clean[i]
		if quoted {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				quoted = false
			}
			continue
		}
		if c == '"' {
			quoted = true
			continue
		}
		if c != '/' || i+1 >= len(clean) {
			continue
		}
		switch clean[i+1] {
		case '/':
			for i < len(clean) && clean[i] != '\n' {
				clean[i] = ' '
				i++
			}
		case '*':
			clean[i], clean[i+1] = ' ', ' '
			i += 2
			closed := false
			for i < len(clean) {
				if clean[i] == '*' && i+1 < len(clean) && clean[i+1] == '/' {
					clean[i], clean[i+1] = ' ', ' '
					i++
					closed = true
					break
				}
				if clean[i] != '\n' && clean[i] != '\r' {
					clean[i] = ' '
				}
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated JSON comment")
			}
		}
	}
	quoted, escaped = false, false
	for i, c := range clean {
		if quoted {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				quoted = false
			}
			continue
		}
		if c == '"' {
			quoted = true
			continue
		}
		if c != ',' {
			continue
		}
		j := i + 1
		for j < len(clean) && strings.ContainsRune(" \t\r\n", rune(clean[j])) {
			j++
		}
		if j < len(clean) && (clean[j] == '}' || clean[j] == ']') {
			clean[i] = ' '
		}
	}
	return PiJSONObject(clean)
}

func piDiscoveryWarning(warnings []func(string), message string) {
	for _, warn := range warnings {
		if warn != nil {
			warn(message)
		}
	}
}

// Extension files are inspected only as static paths/declarations. Their code,
// callbacks, imports and runtime registrations are intentionally never evaluated.
func piDeclaredExtensions(value any, base, home, origin string, contained bool, warnings []func(string), depth int) error {
	if depth > 32 {
		return fmt.Errorf("pi extension source %s: cyclic or overly nested static directory declarations", origin)
	}
	entries, ok := value.([]any)
	if !ok {
		return fmt.Errorf("pi extension source %s: extensions must be a list", origin)
	}
	for _, entry := range entries {
		name, ok := entry.(string)
		if !ok || name == "" || strings.ContainsAny(name, "*?![]") {
			return fmt.Errorf("pi extension source %s: invalid or unsupported filtered path", origin)
		}
		path := name
		if strings.HasPrefix(path, "~/") {
			path = filepath.Join(home, path[2:])
		} else if !filepath.IsAbs(path) {
			path = filepath.Join(base, path)
		}
		if contained && !piWithin(base, path) {
			return fmt.Errorf("pi extension source %s escapes package root: %s", origin, path)
		}
		if err := PiSafePath(path); err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("pi extension source %s: required entry %s: %w", origin, path, err)
		}
		if info.IsDir() {
			piDiscoveryWarning(warnings, fmt.Sprintf("Pi static inventory: extension directory %s declared by %s is not executed; runtime registrations are runtime-unverified", path, origin))
			if err := piNativeExtensions(path, home, warnings, depth+1); err != nil {
				return err
			}
			continue
		}
		if _, _, err := ReadPiFile(path); err != nil {
			return err
		}
		piDiscoveryWarning(warnings, fmt.Sprintf("Pi static inventory: extension %s declared by %s is not executed; runtime registrations are runtime-unverified", path, origin))
	}
	return nil
}

func piExtensionDirectory(dir, home string, warnings []func(string), depth int) (bool, error) {
	manifestPath := filepath.Join(dir, "package.json")
	raw, exists, err := ReadPiFile(manifestPath)
	if err != nil {
		return false, err
	}
	if exists {
		doc, err := PiJSONObject(raw)
		if err != nil {
			return false, fmt.Errorf("pi extension manifest %s: %w", manifestPath, err)
		}
		pi, _ := doc["pi"].(map[string]any)
		if doc["pi"] != nil && pi == nil {
			return false, fmt.Errorf("pi extension manifest %s: pi must be an object", manifestPath)
		}
		if value, declared := pi["extensions"]; declared {
			if err := piDeclaredExtensions(value, dir, home, manifestPath, true, warnings, depth+1); err != nil {
				return false, err
			}
			if nonemptyPiValue(value) {
				return true, nil
			}
		}
	}
	for _, entry := range []string{"index.ts", "index.js"} {
		path := filepath.Join(dir, entry)
		_, exists, err := ReadPiFile(path)
		if err != nil {
			return false, err
		}
		if exists {
			return true, piDeclaredExtensions([]any{entry}, dir, home, dir, true, warnings, depth+1)
		}
	}
	return false, nil
}

func piNativeExtensions(dir, home string, warnings []func(string), depth int) error {
	if depth > 32 {
		return fmt.Errorf("pi extension source %s: cyclic or overly nested static directory declarations", dir)
	}
	if err := PiSafePath(dir); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	found, err := piExtensionDirectory(dir, home, warnings, depth+1)
	if err != nil || found {
		return err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := PiSafePath(path); err != nil {
			return err
		}
		if entry.IsDir() {
			if _, err := piExtensionDirectory(path, home, warnings, depth+1); err != nil {
				return err
			}
			continue
		}
		if strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".js") {
			if err := piDeclaredExtensions([]any{entry.Name()}, dir, home, dir, true, warnings, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func nonemptyPiValue(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	case string:
		return x != ""
	case bool:
		return x
	default:
		return true
	}
}
func piWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, "../") && !filepath.IsAbs(rel)
}

// PiResourceRoots derives only read-only, locally resolvable sources. It never
// runs the upstream npm-root resolver or extension code. Warnings identify
// executable sources whose runtime registrations are outside this inventory.
func PiResourceRoots(home, project, agentRoot string, agents bool, env EnvLookup, warnings ...func(string)) ([]PiResourceRoot, error) {
	if agents {
		v, _ := env("PI_OFFLINE")
		switch strings.ToLower(v) {
		case "1", "true", "yes":
		default:
			piDiscoveryWarning(warnings, "Pi static inventory: global npm discovery is runtime-unverified (npm root -g was not executed); local/cached declared sources are still checked")
		}
	}
	if err := piDefaultBranding(env, ReadPiFile); err != nil {
		return nil, err
	}
	var packages, configured, builtin, native, userAgentDirs, projectAgentDirs, extraAgentDirs []PiResourceRoot
	seenPackages := map[string]bool{}
	seenNpmSources := map[string]bool{}
	activeSubagentsRoot := ""
	addPaths := func(value any, base, kind, origin string, into *[]PiResourceRoot) error {
		if value == nil {
			return nil
		}
		list, ok := value.([]any)
		if !ok {
			return fmt.Errorf("pi source %s: %s paths must be a list", origin, kind)
		}
		for _, v := range list {
			rel, ok := v.(string)
			if !ok || rel == "" || strings.ContainsAny(rel, "*?![]") {
				return fmt.Errorf("pi source %s: unsupported filtered/glob %s path %v", origin, kind, v)
			}
			path := rel
			if !filepath.IsAbs(path) {
				path = filepath.Join(base, path)
			}
			if strings.HasPrefix(rel, "~/") {
				path = filepath.Join(home, rel[2:])
			}
			if err := PiSafePath(path); err != nil {
				return err
			}
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("pi source %s: active %s path %s: %w", origin, kind, path, err)
			}
			*into = append(*into, PiResourceRoot{Kind: kind, Path: path, Origin: origin})
		}
		return nil
	}
	inspectPackage := func(root string, required, piResources bool) error {
		key := fmt.Sprintf("%s:%t", root, piResources)
		if seenPackages[key] {
			return nil
		}
		seenPackages[key] = true
		path := filepath.Join(root, "package.json")
		raw, exists, err := ReadPiFile(path)
		if err != nil {
			return err
		}
		if !exists {
			if required {
				return fmt.Errorf("pi package source %s: missing package.json; no resolver execution permitted", root)
			}
			return nil
		}
		doc, err := PiJSONObject(raw)
		if err != nil {
			return fmt.Errorf("pi package source %s: %w", path, err)
		}
		pi, _ := doc["pi"].(map[string]any)
		if doc["pi"] != nil && pi == nil {
			return fmt.Errorf("pi package source %s: pi metadata must be an object", path)
		}
		if value, declared := pi["extensions"]; piResources && declared {
			if err := piDeclaredExtensions(value, root, home, path, true, warnings, 0); err != nil {
				return err
			}
		}
		if piResources && pi == nil {
			if err := piNativeExtensions(filepath.Join(root, "extensions"), home, warnings, 0); err != nil {
				return err
			}
		}
		for _, pair := range [][2]string{{"skills", "skill"}, {"prompts", "command"}} {
			if !piResources {
				continue
			}
			value, declared := pi[pair[0]]
			if !declared && pi == nil {
				if _, err := os.Stat(filepath.Join(root, pair[0])); os.IsNotExist(err) {
					continue
				} else if err != nil {
					return err
				}
				value = []any{pair[0]}
			}
			var roots []PiResourceRoot
			if err := addPaths(value, root, pair[1], "package: "+path, &roots); err != nil {
				return err
			}
			for _, r := range roots {
				if !piWithin(root, r.Path) {
					return fmt.Errorf("pi package source %s escapes package root: %s", path, r.Path)
				}
			}
			packages = append(packages, roots...)
		}
		if agents {
			sub, _ := pi["subagents"].(map[string]any)
			legacy, _ := doc["pi-subagents"].(map[string]any)
			if pi["subagents"] != nil && sub == nil || doc["pi-subagents"] != nil && legacy == nil {
				return fmt.Errorf("pi package source %s: subagents metadata must be an object", path)
			}
			for _, config := range []map[string]any{legacy, sub} {
				var roots []PiResourceRoot
				if err := addPaths(config["agents"], root, "agent", "package: "+path, &roots); err != nil {
					return err
				}
				for _, r := range roots {
					if !piWithin(root, r.Path) {
						return fmt.Errorf("pi package agents escape %s: %s", root, r.Path)
					}
				}
				packages = append(packages, roots...)
			}
			// This is the selected native consumer's builtin storage contract, not a
			// catalog recipe dispatch. Other packages use their declared agent roots.
			if doc["name"] == "pi-subagents" {
				if doc["version"] != "0.72.1" {
					return fmt.Errorf("pi builtin source %s: unqualified subagents version", path)
				}
				builtin = append(builtin, PiResourceRoot{Kind: "agent", Path: filepath.Join(root, "agents"), Origin: "subagents builtin: " + path})
				if piResources && activeSubagentsRoot == "" {
					activeSubagentsRoot = root
				}
			}
		}
		return nil
	}
	// Project package comes first in subagents' package tier.
	if agents {
		if err := inspectPackage(project, false, false); err != nil {
			return nil, err
		}
	}
	projectAgentRoot := project
	if agents {
		for dir := project; dir != home; dir = filepath.Dir(dir) {
			found := false
			for _, marker := range []string{".pi", ".agents"} {
				info, err := os.Stat(filepath.Join(dir, marker))
				if err == nil && info.IsDir() {
					found = true
				}
			}
			if found {
				projectAgentRoot = dir
				break
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	for _, base := range []string{filepath.Join(project, ".pi"), agentRoot} {
		path := filepath.Join(base, "settings.json")
		raw, exists, err := ReadPiFile(path)
		if err != nil {
			return nil, err
		}
		doc := map[string]any{}
		if exists {
			doc, err = PiJSONObject(raw)
			if err != nil {
				return nil, fmt.Errorf("pi settings source %s: %w", path, err)
			}
		}
		for _, pair := range [][2]string{{"skills", "skill"}, {"prompts", "command"}} {
			if err := addPaths(doc[pair[0]], base, pair[1], path, &configured); err != nil {
				return nil, err
			}
		}
		if agents {
			sub, _ := doc["subagents"].(map[string]any)
			if doc["subagents"] != nil && sub == nil {
				return nil, fmt.Errorf("pi agent source %s: subagents settings must be an object", path)
			}
			for _, key := range []string{"agentExcludeDirs", "projectRootResolution"} {
				if nonemptyPiValue(sub[key]) && sub[key] != "nearest" {
					return nil, fmt.Errorf("pi agent source %s: %s needs qualified discovery", path, key)
				}
			}
			agentDirs := &projectAgentDirs
			if base == agentRoot {
				agentDirs = &userAgentDirs
			}
			if err := addPaths(sub["agentScanDirs"], project, "agent", path, agentDirs); err != nil {
				return nil, err
			}
		}
		if value, declared := doc["extensions"]; declared {
			if err := piDeclaredExtensions(value, base, home, path, false, warnings, 0); err != nil {
				return nil, err
			}
		}
		if err := piNativeExtensions(filepath.Join(base, "extensions"), home, warnings, 0); err != nil {
			return nil, err
		}
		var configuredPackageRoots []string
		if value, ok := doc["packages"]; ok {
			entries, ok := value.([]any)
			if !ok {
				return nil, fmt.Errorf("pi source %s: packages must be a list", path)
			}
			for _, entry := range entries {
				source, ok := entry.(string)
				if !ok {
					obj, valid := entry.(map[string]any)
					if !valid || len(obj) != 1 {
						return nil, fmt.Errorf("pi source %s: package filters are not qualified", path)
					}
					source, ok = obj["source"].(string)
				}
				if !ok || source == "" {
					return nil, fmt.Errorf("pi source %s: invalid package source", path)
				}
				// Pi 0.87.1 resolves a project package before the identical global
				// npm source. Do not inventory its inactive copy as a second skill
				// or prompt source. Different refs still retain conflict checks.
				if _, _, err := nativepi.ParseSource(source); err == nil {
					if seenNpmSources[source] {
						continue
					}
					seenNpmSources[source] = true
				}
				root, err := piPackageRoot(home, base, source)
				if err != nil {
					return nil, fmt.Errorf("pi source %s: %w", path, err)
				}
				configuredPackageRoots = append(configuredPackageRoots, root)
			}
		}
		if agents {
			modules := filepath.Join(base, "npm/node_modules")
			if err := PiSafePath(modules); err != nil {
				return nil, err
			}
			entries, err := os.ReadDir(modules)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".") {
					continue
				}
				root := filepath.Join(modules, entry.Name())
				if strings.HasPrefix(entry.Name(), "@") {
					children, err := os.ReadDir(root)
					if err != nil {
						return nil, err
					}
					for _, child := range children {
						if err := inspectPackage(filepath.Join(root, child.Name()), true, false); err != nil {
							return nil, err
						}
					}
				} else {
					if err := inspectPackage(root, true, false); err != nil {
						return nil, err
					}
				}
			}
		}
		for _, root := range configuredPackageRoots {
			if err := inspectPackage(root, true, true); err != nil {
				return nil, err
			}
		}
	}
	if agents && projectAgentRoot != project {
		// Creating a nearer .pi changes subagents discovery. Never silently replace
		// the previously effective ancestor agent source with a new local directory.
		return nil, fmt.Errorf("pi agent project root %s differs from selected workspace %s; refuse discovery shadowing", projectAgentRoot, project)
	}
	if agents {
		if extra, _ := env("PI_SUBAGENT_EXTRA_AGENT_DIRS"); extra != "" {
			for _, dir := range filepath.SplitList(extra) {
				if err := PiSafePath(dir); err != nil {
					return nil, err
				}
				extraAgentDirs = append(extraAgentDirs, PiResourceRoot{Kind: "agent", Path: dir, Origin: "PI_SUBAGENT_EXTRA_AGENT_DIRS"})
			}
		}
	}
	// Pi's package manager inserts project native resources before user native
	// resources. Legacy .agents skills walk ancestors only up to a git root.
	native = append(native, PiResourceRoot{Kind: "skill", Path: filepath.Join(project, ".pi/skills"), Origin: "native project"})
	for dir := project; ; dir = filepath.Dir(dir) {
		if dir != home {
			native = append(native, PiResourceRoot{Kind: "skill", Path: filepath.Join(dir, ".agents/skills"), Origin: "native ancestor: " + dir, AgentsSkills: true})
		}
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	native = append(native, PiResourceRoot{Kind: "command", Path: filepath.Join(project, ".pi/prompts"), Origin: "native project"}, PiResourceRoot{Kind: "skill", Path: filepath.Join(agentRoot, "skills"), Origin: "native user"}, PiResourceRoot{Kind: "skill", Path: filepath.Join(home, ".agents/skills"), Origin: "native user .agents", AgentsSkills: true}, PiResourceRoot{Kind: "command", Path: filepath.Join(agentRoot, "prompts"), Origin: "native user"})
	if agents {
		native = append(native, extraAgentDirs...)
		native = append(native, userAgentDirs...)
		for _, dir := range []string{filepath.Join(agentRoot, "agents"), filepath.Join(home, ".agents")} {
			native = append(native, PiResourceRoot{Kind: "agent", Path: dir, Origin: "native subagents user: " + dir})
		}
		native = append(native, projectAgentDirs...)
		for _, dir := range []string{filepath.Join(project, ".agents"), filepath.Join(project, ".pi/agents")} {
			native = append(native, PiResourceRoot{Kind: "agent", Path: dir, Origin: "native subagents project: " + dir})
		}
	}
	if activeSubagentsRoot != "" {
		// Builtins belong to the extension Pi loads, not every installed copy
		// discovered in npm storage. Keep the selected consumer's source only.
		selected := filepath.Join(activeSubagentsRoot, "agents")
		var active []PiResourceRoot
		for _, root := range builtin {
			if root.Path == selected && len(active) == 0 {
				active = append(active, root)
			}
		}
		builtin = active
	}
	return append(append(append(builtin, packages...), configured...), native...), nil
}

// The pinned MCP loader reads piConfig statically from PI_PACKAGE_DIR. An
// ordinary host manifest that retains default branding does not redirect roots.
// PiRootOverride refuses forms whose host and MCP interpretations diverge.
// Tilde and relative roots are normalized by toolpath using its explicit base.
func PiRootOverride(env EnvLookup) error {
	value, _ := env("PI_CODING_AGENT_DIR")
	if value != strings.TrimSpace(value) || strings.Contains(value, "://") {
		return fmt.Errorf("pi root override: whitespace or URI spelling has unqualified host/MCP path semantics")
	}
	return nil
}

func piDefaultBranding(env EnvLookup, read PiReadFile) error {
	if err := PiRootOverride(env); err != nil {
		return err
	}
	root, _ := env("PI_PACKAGE_DIR")
	if strings.TrimSpace(root) == "" {
		return nil
	}
	path := filepath.Join(strings.TrimSpace(root), "package.json")
	raw, exists, err := read(path)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("pi discovery PI_PACKAGE_DIR %s: unreadable active host manifest", path)
	}
	doc, err := PiJSONObject(raw)
	if err != nil {
		return fmt.Errorf("pi discovery PI_PACKAGE_DIR %s: %w", path, err)
	}
	config, _ := doc["piConfig"].(map[string]any)
	if value, ok := doc["piConfig"]; ok && value != nil && config == nil {
		return fmt.Errorf("pi discovery PI_PACKAGE_DIR %s: invalid piConfig", path)
	}
	for key, want := range map[string]string{"name": "pi", "configDir": ".pi"} {
		value, ok := config[key]
		if !ok {
			continue
		}
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("pi discovery PI_PACKAGE_DIR %s: invalid piConfig.%s", path, key)
		}
		if strings.TrimSpace(s) != "" && strings.TrimSpace(s) != want {
			return fmt.Errorf("pi discovery PI_PACKAGE_DIR %s: piConfig.%s override disagrees with selected Pi paths", path, key)
		}
	}
	return nil
}

func piPackageRoot(home, base, source string) (string, error) {
	root := source
	switch {
	case strings.HasPrefix(source, "npm:"):
		name := strings.TrimPrefix(source, "npm:")
		start := 0
		if strings.HasPrefix(name, "@") {
			start = 1
		}
		if i := strings.Index(name[start:], "@"); i >= 0 {
			name = name[:start+i]
		}
		var err error
		root, err = PiSourcePath(filepath.Join(base, "npm/node_modules"), name)
		if err != nil {
			return "", err
		}
	case strings.HasPrefix(source, "git:") || strings.HasPrefix(source, "https://"):
		// Finite common cache spelling shared by the pinned Pi/subagents/MCP
		// loaders. No clone, fetch, git query or npm-root command is performed.
		cache := strings.TrimPrefix(source, "git:")
		cache = strings.TrimPrefix(cache, "https://")
		if strings.ContainsAny(cache, "#?:\\\\ ") {
			return "", fmt.Errorf("package %s: unqualified git cache spelling", source)
		}
		if i := strings.LastIndex(cache, "@"); i >= 0 {
			if i < strings.LastIndex(cache, "/") || i == len(cache)-1 {
				return "", fmt.Errorf("package %s: unqualified git ref", source)
			}
			cache = cache[:i]
		}
		cache = strings.TrimSuffix(cache, ".git")
		if len(strings.Split(cache, "/")) < 3 {
			return "", fmt.Errorf("package %s: expected host/owner/repository", source)
		}
		var err error
		root, err = PiSourcePath(filepath.Join(base, "git"), cache)
		if err != nil {
			return "", err
		}
	case strings.Contains(source, "://") || strings.HasPrefix(source, "git@"):
		return "", fmt.Errorf("package %s: unqualified remote cache spelling", source)
	default:
		root = strings.TrimPrefix(root, "file:")
		if strings.HasPrefix(root, "~/") {
			root = filepath.Join(home, root[2:])
		} else if !filepath.IsAbs(root) {
			root = filepath.Join(base, root)
		}
	}
	if err := PiSafePath(root); err != nil {
		return "", err
	}
	return root, nil
}
