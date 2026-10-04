package registry

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/manifest"
)

// These are catalog policy checks, not stronger production decoders: the public
// loaders deliberately allow e.g. absent artifact roles and unknown role names.
func cp00ValidateMeta(m manifest.Meta) error {
	if !manifest.ValidPackageVersion(m.Version) {
		return fmt.Errorf("invalid SemVer version %q", m.Version)
	}
	switch m.Role {
	case manifest.RoleInstruction, manifest.RoleCapability, manifest.RoleMemory,
		manifest.RoleContext, manifest.RoleTools, manifest.RoleSandbox,
		manifest.RoleObservability, manifest.RoleEval, manifest.RoleGuardrail,
		manifest.RoleOrchestration, manifest.RoleLifecycle:
	default:
		return fmt.Errorf("invalid catalog role %q", m.Role)
	}
	if m.Family == manifest.FamilyProfile && m.Role != manifest.RoleLifecycle {
		return fmt.Errorf("profile role must be lifecycle, got %q", m.Role)
	}
	return nil
}

func cp00ValidateRecipe(r *manifest.Recipe) error {
	if err := manifest.ValidateRecipe(r); err != nil {
		return err
	}
	if err := cp00ValidateMeta(r.Header()); err != nil {
		return err
	}
	want := manifest.ShapeWireOnly
	if r.Delivery != nil {
		switch r.Wire.Method {
		case manifest.WireNone:
			want = manifest.ShapeInstall
		case manifest.WireMerge:
			want = manifest.ShapeFetchWire
		case manifest.WireExec:
			want = manifest.ShapeFetchRun
		}
	}
	if r.Shape() != want {
		return fmt.Errorf("recipe shape %q disagrees with delivery/wire: want %q", r.Shape(), want)
	}
	return nil
}

func cp00ValidateAdapter(ad *manifest.Adapter) error {
	if ad.Family != manifest.FamilyAdapter {
		return fmt.Errorf("expected family adapter, got %q", ad.Family)
	}
	if ad.APIVersion != manifest.APIVersion {
		return fmt.Errorf("unsupported adapter apiVersion %q", ad.APIVersion)
	}
	l := ad.Layout
	// Preserve the old tools' required surfaces without imposing MCP or
	// output-style (or hooks/settings) on a tool that does not support them.
	switch ad.Tool {
	case "claude", "codex", "opencode":
		if l.Skill == nil || l.Instruction == nil || l.OutputStyle == nil || l.Mcp == nil {
			return fmt.Errorf("%s: missing required legacy skill/instruction/output-style/mcp layout", ad.Tool)
		}
	}
	var declared int
	checkPaths := func(name string, targets ...manifest.PathTarget) error {
		declared++
		for _, target := range targets {
			if target.OK() {
				return nil
			}
		}
		return fmt.Errorf("%s: declared layout has no usable target", name)
	}
	checkFiles := func(name string, targets ...manifest.FileTarget) error {
		declared++
		usable := false
		for _, target := range targets {
			if !target.OK() {
				if target.Format != "" || target.Path != "" || target.Action != "" {
					return fmt.Errorf("%s: declared target missing file", name)
				}
				continue
			}
			usable = true
			if name == "mcp" || name == "hook" || name == "setting" {
				switch target.Format {
				case "json", "jsonc", "toml":
				default:
					return fmt.Errorf("%s: unsupported target format %q", name, target.Format)
				}
			}
			if name == "instruction" && target.Action != "appendSection" {
				return fmt.Errorf("instruction: expected appendSection action")
			}
			if name == "output-style" && target.Action != "" && target.Action != "appendSection" {
				return fmt.Errorf("output-style: unsupported action %q", target.Action)
			}
			if name == "mcp" && target.Path == "" {
				return fmt.Errorf("mcp: missing merge path")
			}
		}
		if !usable {
			return fmt.Errorf("%s: declared layout has no usable target", name)
		}
		return nil
	}
	if l.Skill != nil {
		if err := checkPaths("skill", l.Skill.Global, l.Skill.Project); err != nil {
			return err
		}
	}
	if l.Agent != nil {
		if err := checkPaths("agent", l.Agent.Global, l.Agent.Project); err != nil {
			return err
		}
	}
	if l.Command != nil {
		if err := checkPaths("command", l.Command.Global, l.Command.Project); err != nil {
			return err
		}
	}
	if l.Instruction != nil {
		if err := checkFiles("instruction", l.Instruction.Global, l.Instruction.Project); err != nil {
			return err
		}
	}
	if l.OutputStyle != nil {
		if err := checkFiles("output-style", l.OutputStyle.Global, l.OutputStyle.Project); err != nil {
			return err
		}
	}
	if l.Mcp != nil {
		if err := checkFiles("mcp", l.Mcp.Global, l.Mcp.Project, l.Mcp.User); err != nil {
			return err
		}
	}
	if l.Hook != nil {
		if err := checkFiles("hook", l.Hook.Global, l.Hook.Project); err != nil {
			return err
		}
	}
	if l.Setting != nil {
		if err := checkFiles("setting", l.Setting.Global, l.Setting.Project); err != nil {
			return err
		}
	}
	if declared == 0 {
		return fmt.Errorf("missing declared layouts")
	}
	return nil
}

// cp00ArtifactInventory follows the portable-source member contract in
// cmd/patronus.collectArtifactFiles: manifest, entry, attribution NOTICE and
// declared Files trees. WalkDir includes dotfiles; no executable is invoked.
func cp00ArtifactInventory(dir string) (map[string][]byte, error) {
	m, err := manifest.LoadArtifact(filepath.Join(dir, "patronus.yaml"))
	if err != nil {
		return nil, err
	}
	if err := cp00ValidateMeta(m.Header()); err != nil {
		return nil, err
	}
	all := make(map[string][]byte)
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsafe non-regular file %s", p)
		}
		if info.Mode().Perm() != 0o644 && info.Mode().Perm() != 0o755 {
			return fmt.Errorf("unsupported distributable mode %o: %s", info.Mode().Perm(), p)
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		all[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte)
	addFile := func(rel string) error {
		if err := cp00SafeRelativePath(rel); err != nil {
			return err
		}
		data, ok := all[rel]
		if !ok {
			return fmt.Errorf("missing distributed file %q", rel)
		}
		out[rel] = data
		return nil
	}
	if err := addFile("patronus.yaml"); err != nil {
		return nil, err
	}
	if m.Entry == "" && m.Type != manifest.TypeHook && m.Type != manifest.TypeSetting {
		return nil, fmt.Errorf("missing entry for %s", m.Type)
	}
	if m.Entry != "" {
		if err := addFile(m.Entry); err != nil {
			return nil, err
		}
	}
	if m.Attribution != nil {
		if err := addFile("NOTICE"); err != nil {
			return nil, fmt.Errorf("attribution license: %w", err)
		}
		if len(strings.TrimSpace(string(out["NOTICE"]))) == 0 {
			return nil, fmt.Errorf("empty attribution license NOTICE")
		}
	}
	for _, rel := range m.Files {
		rel = strings.TrimSuffix(rel, "/") // directory declarations may end in a slash
		if err := cp00SafeRelativePath(rel); err != nil {
			return nil, err
		}
		found := false
		for name, data := range all {
			if name == rel || strings.HasPrefix(name, rel+"/") {
				if err := cp00SafeRelativePath(name); err != nil {
					return nil, err
				}
				out[name] = data
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("missing declared sidecar %q", rel)
		}
	}
	if m.Hook != nil && m.Hook.Script != "" {
		if _, ok := out[m.Hook.Script]; !ok {
			return nil, fmt.Errorf("hook script %q is not distributed", m.Hook.Script)
		}
	}
	return out, nil
}

func cp00SafeRelativePath(rel string) error {
	if rel == "" || rel == "." || path.IsAbs(rel) || path.Clean(rel) != rel ||
		rel == ".." || strings.HasPrefix(rel, "../") || strings.ContainsAny(rel, "\\:\x00") {
		return fmt.Errorf("unsafe distributable path %q", rel)
	}
	return nil
}

func cp00CatalogInventory(root string) (map[string][]byte, error) {
	members := make(map[string][]byte)
	artifactsRoot := filepath.Join(root, "artifacts")
	err := filepath.WalkDir(artifactsRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "patronus.yaml" {
			// Do not silently lose a whole artifact when its manifest is removed.
			for dir := filepath.Dir(p); dir != artifactsRoot; dir = filepath.Dir(dir) {
				if _, err := os.Stat(filepath.Join(dir, "patronus.yaml")); err == nil {
					return nil
				}
			}
			return fmt.Errorf("missing artifact manifest for %s", p)
		}
		inventory, err := cp00ArtifactInventory(filepath.Dir(p))
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		rel, err := filepath.Rel(root, filepath.Dir(p))
		if err != nil {
			return err
		}
		for name, data := range inventory {
			members[path.Join(filepath.ToSlash(rel), name)] = data
		}
		return nil
	})
	return members, err
}

func TestPiContentCatalogInventory(t *testing.T) {
	root := repoRoot(t)
	members, err := cp00CatalogInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) == 0 {
		t.Fatal("no distributed artifact files")
	}
	for name, data := range members {
		if err := cp00DistributedReferences(root, name, data, members); err != nil {
			t.Error(err)
		}
	}
}

// Machine lint checks concrete Markdown links and author-machine path leaks.
// Plain prose invocations, optional capabilities and historical quotations still
// need the C-owned review inventory; this is not a general Markdown parser.
func cp00DistributedReferences(root, name string, data []byte, members map[string][]byte) error {
	authorPath := regexp.MustCompile("(?m)(^|[\\s\"'`(=])(?:/(?:Users|home|root|workspace|workspaces|repo|repos)/|[A-Za-z]:[\\\\/](?:Users|repos)[\\\\/])[^\\s\"'`<>)]*")
	if match := authorPath.Find(data); match != nil {
		return fmt.Errorf("%s: author-home/repository absolute path %q; use /ABSOLUTE/APPROVED/PROJECT in inert templates", name, match)
	}
	if path.Ext(name) != ".md" {
		return nil
	}
	links := regexp.MustCompile(`\]\(([^\s)]+)(?:\s+"[^"]*")?\)`)
	var fence byte
	var fenceLen int
	var literalMarkdown bool
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		// A shorter inner fence does not close a literal Markdown example.
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
		for _, match := range links.FindAllStringSubmatch(line, -1) {
			target := strings.Trim(match[1], "<>")
			if strings.HasPrefix(target, "#") || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			// Explicit angle-bracket template parameters are not file claims.
			if strings.Contains(target, "<") || match[1] == "<relative-path>" {
				continue
			}
			target, _, _ = strings.Cut(target, "#")
			if cp00UpstreamOnlyLink(name, target, string(data)) {
				continue
			}
			resolved := path.Join(path.Dir(name), target)
			if _, ok := members[resolved]; !ok {
				allowed, err := cp00LegacyDistributionDebt(root, name, target, data)
				if err != nil {
					return err
				}
				if !allowed {
					return fmt.Errorf("%s: missing distributed reference %q", name, target)
				}
			}
		}
	}
	return nil
}

func cp00UpstreamOnlyLink(source, target, body string) bool {
	// Reviewed pre-existing classification, not inherited by new Pi siblings.
	if source != "artifacts/skills/writing-skills/SKILL.md" ||
		!strings.Contains(body, "`../using-superpowers/references/*-tools.md` runtime tool-mapping files are\n> upstream-only") {
		return false
	}
	switch target {
	case "../using-superpowers/references/claude-code-tools.md",
		"../using-superpowers/references/codex-tools.md",
		"../using-superpowers/references/copilot-tools.md",
		"../using-superpowers/references/gemini-tools.md":
		return true
	}
	return false
}

// Known baseline delivery defects, explicitly accepted for CP-00 by the
// coordinator. These are NOT claims that the sidecars ship. Bind both manifest
// (including version) and entry bytes; no changed artifact or Pi sibling inherits
// this debt. Remediation needs separately versioned legacy content, not this gate.
func cp00LegacyDistributionDebt(root, source, target string, body []byte) (bool, error) {
	rules := []cp00DistributionDebt{
		{"artifacts/skills/research-team/SKILL.md", "053e8927d68e6a4827c1ef1c6e06b9f6d1a9e9a6f150f466968b7faddfd98525", []string{"RESEARCHER-TEMPLATE.md", "DELIVERABLE-TEMPLATES.md", "LESSONS-FORMAT.md"}},
		{"artifacts/skills/plan-execute-parallel/SKILL.md", "41f296d52c6d2cdfe8723894d1b021850b578e832fa743d6d96ded0038e4420b", []string{"PROVENANCE-GUIDE.md", "TEAMMATE-TEMPLATE.md"}},
	}
	for _, rule := range rules {
		ok, err := rule.check(root, source, target, body)
		if ok || err != nil {
			return ok, err
		}
	}
	return false, nil
}

type cp00DistributionDebt struct {
	source  string
	digest  string // SHA-256(manifest bytes || entry bytes)
	targets []string
}

func (d cp00DistributionDebt) check(root, source, target string, body []byte) (bool, error) {
	if source != d.source {
		return false, nil
	}
	matched := false
	for _, allowed := range d.targets {
		matched = matched || target == allowed
	}
	if !matched {
		return false, nil
	}
	manifestBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path.Dir(source)), "patronus.yaml"))
	if err != nil {
		return false, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(append(manifestBytes, body...))) != d.digest {
		return false, fmt.Errorf("%s: legacy distribution debt no longer matches immutable manifest/content", source)
	}
	info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path.Dir(source)), target))
	if err != nil {
		return false, fmt.Errorf("%s: missing legacy reference source %q: %w", source, target, err)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("%s: unsafe legacy reference source %q", source, target)
	}
	return true, nil
}

func cp00WriteFixture(t *testing.T, root, name, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func cp00ArtifactFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cp00WriteFixture(t, root, "patronus.yaml", "apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: invented-guide\nversion: 1.2.3\ndescription: invented fixture\nentry: SKILL.md\nfiles: [references/, run.sh]\n")
	cp00WriteFixture(t, root, "SKILL.md", "# Invented guide\nRead [reference](references/.fixture).\n")
	cp00WriteFixture(t, root, "references/.fixture", "invented hidden bytes\n")
	cp00WriteFixture(t, root, "run.sh", "#!/bin/sh\necho invented\n")
	if err := os.Chmod(filepath.Join(root, "run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPiContentInventedInventory(t *testing.T) {
	root := cp00ArtifactFixture(t)
	members, err := cp00ArtifactInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if string(members["references/.fixture"]) != "invented hidden bytes\n" {
		t.Fatal("declared directory must inventory dotfile bytes")
	}
	if err := cp00DistributedReferences(root, "SKILL.md", members["SKILL.md"], members); err != nil {
		t.Fatal(err)
	}
}

func TestPiContentInvalidArtifact(t *testing.T) {
	cases := []struct{ name, old, replacement, diagnostic string }{
		{"family", "family: artifact", "family: recipe", "expected family"},
		{"type", "type: skill", "type: widget", "invalid artifact type"},
		{"role", "role: capability", "role: invented-role", "invalid catalog role"},
		{"missing-role", "role: capability", "role: ''", "invalid catalog role"},
		{"version", "version: 1.2.3", "version: not-semver", "invalid SemVer"},
		{"api-version", "patronus/v2", "patronus/v999", "apiVersion"},
		{"entry", "entry: SKILL.md", "entry: missing.md", "missing distributed file"},
		{"missing-entry", "entry: SKILL.md", "entry: ''", "missing entry"},
		{"undeclared-hook-script", "type: skill", "type: hook\nhook: {event: SessionStart, command: 'sh {script}', script: missing.sh}", "is not distributed"},
		{"sidecar", "references/", "absent/", "missing declared sidecar"},
		{"traversal", "references/", "../outside", "unsafe distributable path"},
		{"absolute", "references/", "/home/invented/file", "unsafe distributable path"},
		{"windows-path", "references/", "C:\\Users\\invented", "unsafe distributable path"},
		{"mode", "type: skill", "type: skill\nmode: pointer", "mode is only valid"},
		{"license", "type: skill", "type: skill\nattribution: {upstream: example.invalid/upstream, license: MIT, copyright: Fixture}", "attribution license"},
		{"incomplete-attribution", "type: skill", "type: skill\nattribution: {upstream: example.invalid/upstream}", "attribution requires"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := cp00ArtifactFixture(t)
			p := filepath.Join(root, "patronus.yaml")
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			cp00WriteFixture(t, root, "patronus.yaml", strings.Replace(string(data), tc.old, tc.replacement, 1))
			_, err = cp00ArtifactInventory(root)
			cp00WantDiagnostic(t, err, tc.diagnostic)
		})
	}
}

func TestPiContentUnsafeDirectoryMembers(t *testing.T) {
	if filepath.Separator != '/' {
		t.Skip("requires POSIX filenames containing colon or backslash")
	}
	for _, name := range []string{"references/bad:name.md", `references/bad\name.md`} {
		t.Run(name, func(t *testing.T) {
			root := cp00ArtifactFixture(t)
			cp00WriteFixture(t, root, name, "unsafe directory member\n")
			_, err := cp00ArtifactInventory(root)
			cp00WantDiagnostic(t, err, fmt.Sprintf("unsafe distributable path %q", name))
		})
	}
}

func TestPiContentUnsafeInventory(t *testing.T) {
	t.Run("mode", func(t *testing.T) {
		root := cp00ArtifactFixture(t)
		if err := os.Chmod(filepath.Join(root, "references/.fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := cp00ArtifactInventory(root)
		cp00WantDiagnostic(t, err, "unsupported distributable mode")
	})
	t.Run("symlink", func(t *testing.T) {
		root := cp00ArtifactFixture(t)
		if err := os.Symlink("../outside", filepath.Join(root, "references/leak")); err != nil {
			t.Fatal(err)
		}
		_, err := cp00ArtifactInventory(root)
		cp00WantDiagnostic(t, err, "unsafe non-regular file")
	})
	t.Run("manifest", func(t *testing.T) {
		_, err := cp00ArtifactInventory(t.TempDir())
		cp00WantDiagnostic(t, err, "patronus.yaml")
	})
}

func cp00WantDiagnostic(t *testing.T, err error, diagnostic string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), diagnostic) {
		t.Fatalf("error = %v, want diagnostic %q", err, diagnostic)
	}
}

func TestPiContentRecipeShapes(t *testing.T) {
	base := "apiVersion: patronus/v2\nfamily: recipe\nrole: tools\nname: invented-tool\nversion: 2.3.4\n"
	merge := "wire: {method: merge, actor: patronus, mcp: {transport: stdio, command: invented}}\n"
	delivery := "deliver: {via: script}\n"
	cases := []struct {
		name, body string
		shape      manifest.RecipeShape
	}{
		{"wire-only", merge, manifest.ShapeWireOnly},
		{"install", delivery, manifest.ShapeInstall},
		{"fetch-wire", delivery + merge, manifest.ShapeFetchWire},
		{"fetch-run", delivery + "wire: {method: exec, actor: external, run: [invented]}\n", manifest.ShapeFetchRun},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := manifest.DecodeRecipe([]byte(base + tc.body))
			if err != nil {
				t.Fatal(err)
			}
			if err := cp00ValidateRecipe(r); err != nil {
				t.Fatal(err)
			}
			if r.Shape() != tc.shape {
				t.Fatalf("shape = %s, want %s", r.Shape(), tc.shape)
			}
		})
	}
	for _, tc := range []struct{ name, body, diagnostic string }{
		{"empty-shape", "", "recipe does nothing"},
		{"invalid-method", "wire: {method: teleport}", "invalid wire.method"},
		{"missing-run", delivery + "wire: {method: exec, actor: external}", "wire.run"},
		{"missing-merge", "wire: {method: merge, actor: patronus}", "wire.mcp"},
		{"wrong-actor", "wire: {method: merge, actor: external}", "requires actor: patronus"},
		{"bad-delivery", "deliver: {via: invented}", "invalid deliver.via"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := manifest.DecodeRecipe([]byte(base + tc.body))
			cp00WantDiagnostic(t, err, tc.diagnostic)
		})
	}
}

func TestPiContentProfilePolicy(t *testing.T) {
	for _, tc := range []struct{ name, field, diagnostic string }{
		{"family", "family: artifact\nrole: lifecycle", "expected family"},
		{"role", "family: profile\nrole: capability", "profile role must be lifecycle"},
		{"invalid-role", "family: profile\nrole: invented", "invalid catalog role"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			cp00WriteFixture(t, root, "bundle.yaml", "apiVersion: patronus/v2\nname: invented-bundle\nversion: 1.0.0\n"+tc.field+"\n")
			p, err := manifest.LoadProfile(filepath.Join(root, "bundle.yaml"))
			if err == nil {
				err = cp00ValidateMeta(p.Header())
			}
			cp00WantDiagnostic(t, err, tc.diagnostic)
		})
	}
}

func TestPiContentDeclaredAdapterLayouts(t *testing.T) {
	base := "apiVersion: patronus/v2\nfamily: adapter\ntool: invented\n"
	// New tools can omit unsupported hooks/output-style/MCP; absence is not an
	// empty declaration. No production Pi YAML is needed to exercise this rule.
	for _, layout := range []string{
		"skill: {project: '.invented/skills/{name}/SKILL.md'}",
		"command: {global: '~/.invented/prompts/{name}.md', project: null}",
		"instruction: {project: {file: AGENTS.md, action: appendSection}}",
		"output-style: {project: {file: '.invented/output-styles/{name}.md'}}",
		"output-style: {project: {file: AGENTS.md, action: appendSection}}",
	} {
		ad, err := manifest.DecodeAdapter([]byte(base + "layout: {" + layout + "}\n"))
		if err != nil {
			t.Fatal(err)
		}
		if err := cp00ValidateAdapter(ad); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct{ name, body, diagnostic string }{
		{"missing-layout", "layout: {}", "missing declared layouts"},
		{"empty-skill", "layout: {skill: {}}", "skill: declared layout"},
		{"empty-agent", "layout: {agent: {}}", "agent: declared layout"},
		{"missing-file", "layout: {instruction: {project: {action: appendSection}}}", "missing file"},
		{"missing-action", "layout: {instruction: {project: {file: AGENTS.md}}}", "appendSection"},
		{"invalid-output-action", "layout: {output-style: {project: {file: AGENTS.md, action: appendSectoin}}}", "output-style: unsupported action"},
		{"bad-format", "layout: {mcp: {project: {file: config, format: invented}}}", "unsupported target format"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ad, err := manifest.DecodeAdapter([]byte(base + tc.body))
			if err != nil {
				t.Fatal(err)
			}
			cp00WantDiagnostic(t, cp00ValidateAdapter(ad), tc.diagnostic)
		})
	}
	t.Run("legacy-required-layout", func(t *testing.T) {
		ad, err := manifest.DecodeAdapter([]byte(strings.Replace(base, "tool: invented", "tool: claude", 1) + "layout: {skill: {project: skill.md}}"))
		if err != nil {
			t.Fatal(err)
		}
		cp00WantDiagnostic(t, cp00ValidateAdapter(ad), "missing required legacy")
	})
}

func TestPiContentReferencePolicy(t *testing.T) {
	members := map[string][]byte{"references/.fixture": []byte("invented")}
	for _, tc := range []struct{ name, body string }{
		{"declared-dotfile", "Read [reference](references/.fixture)."},
		{"external", "See [upstream](https://example.invalid/reference) and [section](#local)."},
		{"inert-approved-project", "root: /ABSOLUTE/APPROVED/PROJECT"},
		{"variable-not-author-home", `HOME="$SANDBOX/home"`},
		{"nested-markdown-example", "````markdown theme={null}\n[example](absent.md)\n```python\nx = 1\n```\n[still example](also-absent.md)\n````\n[actual](references/.fixture)"},
		{"template-parameter", "[findings](<relative-path>)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := cp00DistributedReferences(t.TempDir(), "SKILL.md", []byte(tc.body), members); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, tc := range []struct{ name, body, diagnostic string }{
		{"missing-active-link", "Read [required](missing.md).", "missing distributed reference"},
		{"angle-link", "Read [required](<missing.md>).", "missing distributed reference"},
		{"missing-after-example", "````markdown\n[example](example.md)\n```\n````\n[required](missing.md)", "missing distributed reference"},
		{"shell-not-markdown-example", "```sh\n# Read [required](missing.md)\n```", "missing distributed reference"},
		{"author-home", "Read /home/invented/project/guide.md", "author-home/repository absolute path"},
		{"author-macos", "cd /Users/invented/repository", "author-home/repository absolute path"},
		{"author-windows", `cd C:\Users\invented\repository`, "author-home/repository absolute path"},
		{"repository", "root: /workspace/invented", "author-home/repository absolute path"},
		{"template-real-home", "```json\n{\"root\": \"/home/invented/repository\"}\n```", "author-home/repository absolute path"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := cp00DistributedReferences(t.TempDir(), "SKILL.md", []byte(tc.body), members)
			cp00WantDiagnostic(t, err, tc.diagnostic)
		})
	}
}

func TestPiContentUpstreamLinkClassification(t *testing.T) {
	source := "artifacts/skills/writing-skills/SKILL.md"
	target := "../using-superpowers/references/claude-code-tools.md"
	note := "`../using-superpowers/references/*-tools.md` runtime tool-mapping files are\n> upstream-only"
	if !cp00UpstreamOnlyLink(source, target, note) {
		t.Fatal("documented upstream-only link rejected")
	}
	for _, tc := range []struct{ source, target, body string }{
		{source, target, "classification removed"},
		{source + "-pi", target, note},
		{source, "../using-superpowers/references/invented-tools.md", note},
	} {
		if cp00UpstreamOnlyLink(tc.source, tc.target, tc.body) {
			t.Fatal("classification escaped its exact source/target/note")
		}
	}
}

func TestPiContentLegacyDistributionDebt(t *testing.T) {
	root := t.TempDir()
	source := "artifacts/skills/invented/SKILL.md"
	metadata := "version: 1.0.0\nfiles: []\n"
	body := []byte("Read [sidecar](sidecar.md).")
	cp00WriteFixture(t, root, "artifacts/skills/invented/patronus.yaml", metadata)
	cp00WriteFixture(t, root, "artifacts/skills/invented/sidecar.md", "invented sidecar")
	rule := cp00DistributionDebt{source: source, digest: fmt.Sprintf("%x", sha256.Sum256(append([]byte(metadata), body...))), targets: []string{"sidecar.md"}}
	ok, err := rule.check(root, source, "sidecar.md", body)
	if err != nil || !ok {
		t.Fatalf("unchanged debt: ok=%v err=%v", ok, err)
	}
	t.Run("changed-source", func(t *testing.T) {
		ok, err := rule.check(root, "artifacts/skills/invented-pi/SKILL.md", "sidecar.md", body)
		if err != nil || ok {
			t.Fatalf("Pi sibling inherited legacy debt: ok=%v err=%v", ok, err)
		}
	})
	t.Run("changed-content", func(t *testing.T) {
		_, err := rule.check(root, source, "sidecar.md", []byte("changed"))
		cp00WantDiagnostic(t, err, "no longer matches")
	})
	t.Run("changed-version", func(t *testing.T) {
		cp00WriteFixture(t, root, "artifacts/skills/invented/patronus.yaml", "version: 1.0.1\nfiles: []\n")
		_, err := rule.check(root, source, "sidecar.md", body)
		cp00WantDiagnostic(t, err, "no longer matches")
		cp00WriteFixture(t, root, "artifacts/skills/invented/patronus.yaml", metadata)
	})
	t.Run("missing-source", func(t *testing.T) {
		if err := os.Remove(filepath.Join(root, "artifacts/skills/invented/sidecar.md")); err != nil {
			t.Fatal(err)
		}
		_, err := rule.check(root, source, "sidecar.md", body)
		cp00WantDiagnostic(t, err, "missing legacy reference source")
	})
}

func TestPiContentAttributionNotice(t *testing.T) {
	root := cp00ArtifactFixture(t)
	data, err := os.ReadFile(filepath.Join(root, "patronus.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cp00WriteFixture(t, root, "patronus.yaml", string(data)+"attribution: {upstream: example.invalid/upstream, license: MIT, copyright: Fixture}\n")
	cp00WriteFixture(t, root, "NOTICE", "Invented attribution and license\n")
	members, err := cp00ArtifactInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if string(members["NOTICE"]) != "Invented attribution and license\n" {
		t.Fatal("NOTICE missing from distribution")
	}
	cp00WriteFixture(t, root, "NOTICE", "\n")
	_, err = cp00ArtifactInventory(root)
	cp00WantDiagnostic(t, err, "empty attribution license NOTICE")
}

func TestPiContentDanglingRequires(t *testing.T) {
	root := t.TempDir()
	cp00WriteFixture(t, root, "artifacts/skills/invented/patronus.yaml", "apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: invented\nversion: 1.0.0\ndescription: invented fixture\nrequires: [missing-dependency]\n")
	_, err := NewLocalRegistry(root).Catalog(context.Background())
	cp00WantDiagnostic(t, err, `requires unknown item "missing-dependency"`)
}

func TestPiContentMissingCatalogManifest(t *testing.T) {
	root := t.TempDir()
	cp00WriteFixture(t, root, "artifacts/skills/invented/SKILL.md", "Invented orphan body")
	_, err := cp00CatalogInventory(root)
	cp00WantDiagnostic(t, err, "missing artifact manifest")
}

// Native role syntax and selected skill references are catalog data. These
// checks discover subjects rather than encoding any shipped role membership.
func TestPiContentNativeAgents(t *testing.T) {
	cat, err := NewLocalRegistry(repoRoot(t)).Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range cat.Artifacts {
		art := entry.Manifest
		if art.Type != manifest.TypeAgent || !cp03Contains(art.Targets, "pi") {
			continue
		}
		t.Run(art.Name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(entry.Source.LocalDir, art.Entry))
			if err != nil {
				t.Fatal(err)
			}
			if err := cp03NativeAgentReferences(cat, art, raw); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func cp03Contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func cp03NativeAgentReferences(cat *Catalog, art *manifest.Artifact, raw []byte) error {
	if len(art.Targets) != 1 || art.Targets[0] != "pi" {
		return fmt.Errorf("%s: native agent requires sole Pi target", art.Name)
	}
	if len(art.Files) != 0 || len(art.Overrides) != 0 {
		return fmt.Errorf("%s: native agent requires no sidecars or Overrides", art.Name)
	}
	skills, err := adapter.ValidatePiAgent(art.Name, raw)
	if err != nil {
		return err
	}
	for _, name := range skills {
		if !cp03Contains(art.Requires, name) {
			return fmt.Errorf("%s: selected skill %q not declared in requires", art.Name, name)
		}
		found := false
		for _, entry := range cat.Artifacts {
			skill := entry.Manifest
			if skill.Name == name && skill.Type == manifest.TypeSkill &&
				(len(skill.Targets) == 0 || cp03Contains(skill.Targets, "pi")) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%s: selected resource %q is not a Pi-compatible skill", art.Name, name)
		}
	}
	return nil
}

func TestPiContentNativeAgentReferences(t *testing.T) {
	const body = "---\nname: invented-reader\ndescription: Invented fixture\ntools: read, bash\nskills: invented-guide\n---\nRead the assigned source.\n"
	const agent = "apiVersion: patronus/v2\nfamily: artifact\ntype: agent\nrole: orchestration\nname: invented-reader\nversion: 1.0.0\ndescription: Invented fixture\nentry: agent.md\ntargets: [pi]\nrequires: [invented-guide]\n"
	const skill = "apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: invented-guide\nversion: 1.0.0\ndescription: Invented fixture\nentry: SKILL.md\ntargets: [pi]\n"
	check := func(t *testing.T, agentData, skillData, raw string) error {
		t.Helper()
		a, err := manifest.DecodeArtifact([]byte(agentData))
		if err != nil {
			t.Fatal(err)
		}
		s, err := manifest.DecodeArtifact([]byte(skillData))
		if err != nil {
			t.Fatal(err)
		}
		cat := &Catalog{Artifacts: []ArtifactEntry{{Manifest: a}, {Manifest: s}}}
		return cp03NativeAgentReferences(cat, a, []byte(raw))
	}
	t.Run("valid", func(t *testing.T) {
		if err := check(t, agent, skill, body); err != nil {
			t.Fatal(err)
		}
	})
	for _, tc := range []struct{ name, agent, skill, body, diagnostic string }{
		{"identity", agent, skill, strings.Replace(body, "name: invented-reader", "name: different", 1), "identity disagree"},
		{"native-flow-list", agent, skill, strings.Replace(body, "skills: invented-guide", "skills: [invented-guide]", 1), "invalid unquoted list token"},
		{"undeclared-skill", strings.Replace(agent, "requires: [invented-guide]", "requires: []", 1), skill, body, "not declared in requires"},
		{"absent-skill", agent, strings.Replace(skill, "name: invented-guide", "name: other-guide", 1), body, "not a Pi-compatible skill"},
		{"wrong-type", agent, strings.Replace(skill, "type: skill", "type: command", 1), body, "not a Pi-compatible skill"},
		{"wrong-target", agent, strings.Replace(skill, "targets: [pi]", "targets: [claude]", 1), body, "not a Pi-compatible skill"},
		{"mixed-targets", strings.Replace(agent, "targets: [pi]", "targets: [pi, claude]", 1), skill, body, "sole Pi target"},
		{"sidecars", agent + "files: [extra.md]\n", skill, body, "no sidecars or Overrides"},
		{"overrides", agent + "overrides: {pi: {model: invented}}\n", skill, body, "no sidecars or Overrides"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cp00WantDiagnostic(t, check(t, tc.agent, tc.skill, tc.body), tc.diagnostic)
		})
	}
}
