package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type document map[string]any
type item struct {
	manifest document
	file     string
	payload  map[string][]byte
}
type catalog struct {
	root    string
	items   map[string]*item
	targets map[string]bool
}
type indexEntry struct {
	Manifest document `json:"manifest"`
	Tarball  struct {
		SHA256 string `json:"sha256"`
	} `json:"tarball"`
}
type publicIndex struct {
	SchemaVersion int          `json:"schemaVersion"`
	Artifacts     []indexEntry `json:"artifacts"`
	Recipes       []indexEntry `json:"recipes"`
	Profiles      []indexEntry `json:"profiles"`
	Plugins       []indexEntry `json:"plugins"`
}

func text(m document, key string) string { s, _ := m[key].(string); return s }
func asDocument(value any) (document, bool) {
	switch v := value.(type) {
	case map[string]any:
		return v, true
	case document:
		return v, true
	}
	return nil, false
}
func object(m document, key string) document {
	v, _ := asDocument(m[key])
	return v
}
func stringList(v any) ([]string, error) {
	if v == nil {
		return nil, nil
	}
	if s, ok := v.(string); ok {
		return []string{s}, nil
	}
	a, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected string list, got %T", v)
	}
	out := make([]string, 0, len(a))
	for _, x := range a {
		s, ok := x.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("empty/non-string list item")
		}
		out = append(out, s)
	}
	return out, nil
}
func decodeYAML(data []byte) (document, error) {
	var m document
	err := yaml.Unmarshal(data, &m)
	if err == nil && len(m) == 0 {
		err = fmt.Errorf("empty manifest")
	}
	return m, err
}
func safeRelative(name string) bool {
	return name != "" && name != "." && path.Clean(name) == name && !path.IsAbs(name) && !strings.ContainsAny(name, "\\:\x00") && name != ".." && !strings.HasPrefix(name, "../")
}
func safeComponent(name string) bool { return safeRelative(name) && !strings.Contains(name, "/") }
func regularPath(root, rel string, allowDir bool) (string, error) {
	// Public manifests may spell a declared directory with one trailing slash.
	if allowDir {
		rel = strings.TrimSuffix(rel, "/")
	}
	if !safeRelative(rel) {
		return "", fmt.Errorf("unsafe path %q", rel)
	}
	current := root
	info, err := os.Lstat(current)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("non-directory/symlink root %s", root)
	}
	parts := strings.Split(rel, "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err = os.Lstat(current)
		if err != nil {
			return "", fmt.Errorf("%s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink path %s", current)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return "", fmt.Errorf("non-directory %s", current)
		}
	}
	if !info.Mode().IsRegular() && !(allowDir && info.IsDir()) {
		return "", fmt.Errorf("nonregular path %s", current)
	}
	return current, nil
}
func readRegular(root, rel string) ([]byte, error) {
	p, err := regularPath(root, rel, false)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 64<<20+1))
	if len(b) > 64<<20 {
		return nil, fmt.Errorf("%s: file exceeds 64MiB validation limit", p)
	}
	return b, err
}
func readDocument(root, rel string) (document, error) {
	b, err := readRegular(root, rel)
	if err != nil {
		return nil, err
	}
	m, err := decodeYAML(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	return m, nil
}

func checkCatalog(source, indexPath string) (*catalog, error) {
	root, err := filepath.Abs(source)
	if err != nil {
		return nil, err
	}
	c := &catalog{root: root, items: map[string]*item{}, targets: map[string]bool{"all": true}}
	for _, group := range []struct {
		dir, family string
		recursive   bool
	}{{"artifacts", "artifact", true}, {"recipes", "recipe", false}, {"profiles", "profile", false}, {"plugins", "plugin", false}, {"adapters", "adapter", false}} {
		base := filepath.Join(root, group.dir)
		if _, err := os.Lstat(base); os.IsNotExist(err) {
			continue
		}
		if _, err := regularPath(root, group.dir, true); err != nil {
			return nil, err
		}
		err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink source %s", p)
			}
			if d.IsDir() {
				if p != base && !group.recursive {
					return fs.SkipDir
				}
				return nil
			}
			if !d.Type().IsRegular() {
				return fmt.Errorf("nonregular source %s", p)
			}
			if group.recursive {
				if d.Name() != "patronus.yaml" {
					return nil
				}
			} else if filepath.Ext(p) != ".yaml" && filepath.Ext(p) != ".yml" {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			m, err := readDocument(root, filepath.ToSlash(rel))
			if err != nil {
				return err
			}
			if group.family == "adapter" {
				target := text(m, "tool")
				if !safeComponent(target) || c.targets[target] {
					return fmt.Errorf("%s: missing/duplicate adapter tool", rel)
				}
				c.targets[target] = true
				return nil
			}
			if err := validateIdentity(m, group.family); err != nil {
				return fmt.Errorf("%s: %w", rel, err)
			}
			name := text(m, "name")
			if c.items[name] != nil {
				return fmt.Errorf("%s: duplicate source name %s", rel, name)
			}
			it := &item{manifest: m, file: p}
			c.items[name] = it
			if group.family == "artifact" {
				if filepath.Base(filepath.Dir(p)) != name {
					return fmt.Errorf("%s: directory/name mismatch", rel)
				}
				it.payload, err = sourcePayload(it)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if len(c.items) == 0 {
		return nil, fmt.Errorf("empty source catalog")
	}
	if err := c.checkReferences(); err != nil {
		return nil, err
	}
	if err := c.checkDistributedResources(); err != nil {
		return nil, err
	}
	indexAbs, err := filepath.Abs(indexPath)
	if err != nil {
		return nil, err
	}
	indexRoot := filepath.Dir(indexAbs)
	data, err := readRegular(indexRoot, filepath.Base(indexAbs))
	if err != nil {
		return nil, err
	}
	digest, err := readRegular(indexRoot, filepath.Base(indexAbs)+".sha256")
	if err != nil {
		return nil, fmt.Errorf("index sidecar: %w", err)
	}
	if err := verifyDigest(data, string(bytes.TrimSpace(digest))); err != nil {
		return nil, fmt.Errorf("index digest: %w", err)
	}
	var ix publicIndex
	if err := json.Unmarshal(data, &ix); err != nil {
		return nil, fmt.Errorf("index: %w", err)
	}
	if ix.SchemaVersion != 1 && ix.SchemaVersion != 2 {
		return nil, fmt.Errorf("index: unsupported schemaVersion %d", ix.SchemaVersion)
	}
	if len(ix.Plugins) != 0 {
		return nil, fmt.Errorf("index plugins not emitted by public build; source-only plugin checks required")
	}
	seen := map[string]bool{}
	for _, group := range []struct {
		family  string
		entries []indexEntry
	}{{"artifact", ix.Artifacts}, {"recipe", ix.Recipes}, {"profile", ix.Profiles}} {
		for _, entry := range group.entries {
			m := entry.Manifest
			if err := validateIdentity(m, group.family); err != nil {
				return nil, fmt.Errorf("index: %w", err)
			}
			name := text(m, "name")
			if seen[name] {
				return nil, fmt.Errorf("index duplicate name %s", name)
			}
			seen[name] = true
			it := c.items[name]
			if it == nil || text(it.manifest, "family") != group.family {
				return nil, fmt.Errorf("index/source mismatch: %s", name)
			}
			if !manifestEqual(it.manifest, m) {
				return nil, fmt.Errorf("%s: source/index manifest mismatch", name)
			}
			if group.family == "artifact" {
				if err := checkBundle(indexRoot, it, entry); err != nil {
					return nil, fmt.Errorf("%s: %w", name, err)
				}
			}
		}
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("empty index")
	}
	for name, it := range c.items {
		if text(it.manifest, "family") != "plugin" && !seen[name] {
			return nil, fmt.Errorf("source/index set mismatch: missing %s", name)
		}
	}
	return c, nil
}
func validateIdentity(m document, family string) error {
	if text(m, "family") != family {
		return fmt.Errorf("expected family %s", family)
	}
	api := text(m, "apiVersion")
	if api != "patronus/v2" && !(family == "recipe" && api == "patronus/v3") {
		return fmt.Errorf("unsupported apiVersion %q", api)
	}
	for _, key := range []string{"name", "version"} {
		if !safeComponent(text(m, key)) {
			return fmt.Errorf("unsafe/missing %s %q", key, text(m, key))
		}
	}
	if text(m, "role") == "" {
		return fmt.Errorf("missing role")
	}
	return nil
}
func sourcePayload(it *item) (map[string][]byte, error) {
	m := it.manifest
	root := filepath.Dir(it.file)
	out := map[string][]byte{}
	add := func(rel string) error {
		b, err := readRegular(root, rel)
		if err != nil {
			return err
		}
		out[rel] = b
		return nil
	}
	entry := text(m, "entry")
	if entry != "" {
		if err := add(entry); err != nil {
			return nil, err
		}
	} else if text(m, "type") != "hook" && text(m, "type") != "setting" {
		return nil, fmt.Errorf("%s: missing entry", it.file)
	}
	files, err := stringList(m["files"])
	if err != nil {
		return nil, err
	}
	if object(m, "attribution") != nil {
		files = append(files, "NOTICE")
	}
	for _, rel := range files {
		p, err := regularPath(root, rel, true)
		if err != nil {
			return nil, err
		}
		err = filepath.WalkDir(p, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink sidecar %s", p)
			}
			if d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			return add(filepath.ToSlash(rel))
		})
		if err != nil {
			return nil, err
		}
	}
	if script := text(object(m, "hook"), "script"); script != "" {
		if _, ok := out[script]; !ok {
			return nil, fmt.Errorf("%s: hook script %s not declared in payload", it.file, script)
		}
	}
	if text(m, "type") == "skill" {
		body := out[entry]
		body = bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))
		if !bytes.HasPrefix(body, []byte("---\n")) {
			return nil, fmt.Errorf("%s: missing skill frontmatter", it.file)
		}
		end := bytes.Index(body[4:], []byte("\n---"))
		if end < 0 {
			return nil, fmt.Errorf("%s: unterminated skill frontmatter", it.file)
		}
		front, err := decodeYAML(body[4 : 4+end])
		if err != nil {
			return nil, err
		}
		if text(front, "name") != text(m, "name") {
			return nil, fmt.Errorf("%s: skill frontmatter name mismatch", it.file)
		}
	}
	return out, nil
}
func (c *catalog) checkReferences() error {
	for name, it := range c.items {
		m := it.manifest
		refs, err := stringList(m["requires"])
		if err != nil {
			return fmt.Errorf("%s requires: %w", name, err)
		}
		if text(m, "family") == "profile" {
			if base := text(m, "extends"); base != "" {
				if p := c.items[base]; p == nil || text(p.manifest, "family") != "profile" {
					return fmt.Errorf("%s: missing extends profile %s", name, base)
				}
			}
			for _, v := range object(m, "layers") {
				list, err := stringList(v)
				if err != nil {
					return fmt.Errorf("%s layers: %w", name, err)
				}
				refs = append(refs, list...)
			}
		}
		for _, ref := range refs {
			base, qualifier, qualified := strings.Cut(ref, "@")
			if qualified && !c.targets[qualifier] {
				return fmt.Errorf("%s: unknown reference target %s", name, ref)
			}
			if c.items[base] == nil {
				return fmt.Errorf("%s: missing declared reference %s", name, ref)
			}
		}
		targets, err := stringList(m["targets"])
		if err != nil {
			return err
		}
		for _, target := range targets {
			if !c.targets[target] {
				return fmt.Errorf("%s: undeclared target %s", name, target)
			}
		}
	}
	return nil
}

// Compare metadata, not prose or resolver results. Public writers omit empty fields,
// normalize scalar profile layers to lists, and materialize hook intent's default.
func canonical(v any) any {
	switch x := v.(type) {
	case document:
		return canonical(map[string]any(x))
	case map[string]any:
		out := map[string]any{}
		for k, v := range x {
			value := canonical(v)
			if value != nil {
				out[k] = value
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case []any:
		if len(x) == 0 {
			return nil
		}
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = canonical(v)
		}
		return out
	case string:
		if x == "" {
			return nil
		}
	case int:
		if x == 0 {
			return nil
		}
		return float64(x)
	case float64:
		if x == 0 {
			return nil
		}
	case bool:
		if !x {
			return nil
		}
	}
	return v
}
func normalized(m document) any {
	b, _ := json.Marshal(m)
	var copy document
	_ = json.Unmarshal(b, &copy)
	// The public recipe writer omits the legacy source-only Kind header.
	if text(copy, "family") == "recipe" && text(copy, "kind") == "Recipe" {
		delete(copy, "kind")
	}
	if text(copy, "family") == "profile" {
		for k, v := range object(copy, "layers") {
			if s, ok := v.(string); ok {
				object(copy, "layers")[k] = []any{s}
			}
		}
	}
	if h := object(copy, "hook"); h != nil && text(h, "intent") == "" {
		h["intent"] = "nudge"
	}
	return canonical(copy)
}
func manifestEqual(a, b document) bool { return reflect.DeepEqual(normalized(a), normalized(b)) }
func verifyDigest(data []byte, digest string) error {
	want := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	if digest != want {
		return fmt.Errorf("SHA-256 mismatch (want %s)", want)
	}
	return nil
}
func checkBundle(root string, it *item, entry indexEntry) error {
	name, version := text(entry.Manifest, "name"), text(entry.Manifest, "version")
	rel := fmt.Sprintf("%s/%s/%s-%s.tar.gz", name, version, name, version)
	b, err := readRegular(root, rel)
	if err != nil {
		return err
	}
	if err := verifyDigest(b, entry.Tarball.SHA256); err != nil {
		return err
	}
	gz, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	members := map[string][]byte{}
	var total int64
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("archive: %w", err)
		}
		if !safeRelative(h.Name) || (h.Typeflag != tar.TypeReg && h.Typeflag != '\x00') {
			return fmt.Errorf("archive unsafe/nonregular member %q", h.Name)
		}
		if _, exists := members[h.Name]; exists {
			return fmt.Errorf("archive duplicate member %q", h.Name)
		}
		total += h.Size
		if h.Size < 0 || h.Size > 64<<20 || total > 256<<20 || len(members) >= 10000 {
			return fmt.Errorf("archive validation size/member limit exceeded")
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return fmt.Errorf("archive: %w", err)
		}
		members[h.Name] = data
	}
	// Read to gzip EOF as well, so its trailer checksum is validated.
	if _, err := io.Copy(io.Discard, gz); err != nil {
		return fmt.Errorf("archive gzip checksum: %w", err)
	}
	archiveManifest, err := decodeYAML(members["patronus.yaml"])
	if err != nil {
		return fmt.Errorf("archive manifest: %w", err)
	}
	if !manifestEqual(archiveManifest, entry.Manifest) {
		return fmt.Errorf("archive/index manifest mismatch")
	}
	if len(members) != len(it.payload)+1 {
		return fmt.Errorf("archive/source payload set mismatch")
	}
	for rel, want := range it.payload {
		got, ok := members[rel]
		if !ok || !bytes.Equal(got, want) {
			return fmt.Errorf("archive missing/different declared payload %s", rel)
		}
	}
	return nil
}

var _componentPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.@+-]*$`)

func caseComponent(s string) bool {
	return safeComponent(s) && _componentPattern.MatchString(s) && !strings.Contains(s, "--")
}
func sortedNames(items map[string]*item) []string {
	out := make([]string, 0, len(items))
	for name := range items {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
