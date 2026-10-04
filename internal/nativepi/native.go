// Package nativepi observes the accepted Pi 0.87.1 managed npm layout. It
// reads only scoped settings and the selected package.json, never npm files'
// integrity or the legacy global fallback. Pi/npm alone mutate package files.
package nativepi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const EditWarning = "This operation may discard manual edits inside the installed plugin. Patronus does not check those files for modifications. Pi/npm may also change dependencies; installation may use network and run dependency scripts."
const RuntimeVersion = "0.87.1"

var sourcePattern = regexp.MustCompile(`^npm:((?:@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*)@((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?)$`)

func ParseSource(source string) (name, version string, err error) {
	m := sourcePattern.FindStringSubmatch(source)
	if m == nil || len(m[1]) > 214 {
		return "", "", fmt.Errorf("pi requires an exact npm:name@version source, got %q", source)
	}
	core, _, _ := strings.Cut(m[2], "+")
	_, pre, _ := strings.Cut(core, "-")
	for _, p := range strings.Split(pre, ".") {
		if len(p) > 1 && p[0] == '0' && strings.Trim(p, "0123456789") == "" {
			return "", "", fmt.Errorf("invalid exact npm version %q", m[2])
		}
	}
	return m[1], m[2], nil
}

type Identity struct {
	Name      string `json:"name"`
	Scope     string `json:"scope"`
	Root      string `json:"root"`
	Project   string `json:"project"`
	AgentRoot string `json:"agentRoot"`
}

type Operation struct {
	Identity Identity `json:"identity"`
	Kind     string   `json:"operation"`
	Source   string   `json:"source"`
}

// Validate rejects ambiguous aliases, including symlinks at existing components.
// Missing managed directories are valid install destinations, not evidence of
// payload presence. Rechecked immediately before and after manager execution.
func (id Identity) Validate() error {
	name, _, err := ParseSource("npm:" + id.Name + "@0.0.0")
	if err != nil || name != id.Name {
		return fmt.Errorf("invalid native npm identity %q", id.Name)
	}
	if id.Scope != "global" && id.Scope != "local" {
		return errors.New("invalid native scope")
	}
	if id.Scope == "global" && id.Root != id.AgentRoot {
		return errors.New("global native root differs from agent root")
	}
	if id.Scope == "local" && id.Root != filepath.Join(id.Project, ".pi") {
		return errors.New("local native root differs from project/.pi")
	}
	for _, p := range []string{id.Root, id.AgentRoot, id.Project} {
		if err := SafePath(p); err != nil {
			return err
		}
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return fmt.Errorf("native root is not a directory: %s", p)
		}
	}
	return nil
}

func SafePath(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == string(filepath.Separator) {
		return fmt.Errorf("noncanonical native path %q", path)
	}
	for p := path; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && (info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && p != path) || (!info.IsDir() && !info.Mode().IsRegular())) {
			return fmt.Errorf("unsafe native path %s", p)
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return nil
}

func (op Operation) Validate() error {
	if err := op.Identity.Validate(); err != nil {
		return err
	}
	name, _, err := ParseSource(op.Source)
	if err != nil || name != op.Identity.Name {
		return fmt.Errorf("native source does not match selected identity: %q", op.Source)
	}
	if op.Kind != "install" && op.Kind != "remove" {
		return fmt.Errorf("unsupported native operation %q", op.Kind)
	}
	return nil
}

func (op Operation) Argv() []string {
	source := op.Source
	if op.Kind == "remove" {
		source = "npm:" + op.Identity.Name
	}
	argv := []string{"pi", op.Kind, source}
	if op.Identity.Scope == "local" {
		return append(argv, "-l", "--approve")
	}
	return append(argv, "--no-approve")
}

func (id Identity) SettingsPath() string { return filepath.Join(id.Root, "settings.json") }
func (id Identity) MetadataPath() string {
	return filepath.Join(id.Root, "npm", "node_modules", filepath.FromSlash(id.Name), "package.json")
}

type Observation struct {
	Status  string `json:"status"`
	Source  string `json:"source,omitempty"`
	Version string `json:"version,omitempty"`
}

func read(path string) ([]byte, error) {
	if err := SafePath(path); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err == nil {
		err = uniqueJSON(b)
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return b, err
}

func Observe(op Operation) (Observation, error) {
	var o Observation
	if err := op.Validate(); err != nil {
		return o, err
	}
	settings, err := read(op.Identity.SettingsPath())
	if err != nil {
		return o, err
	}
	if settings != nil {
		var root map[string]json.RawMessage
		if err = json.Unmarshal(settings, &root); err != nil || root == nil {
			return o, fmt.Errorf("malformed Pi settings at %s", op.Identity.SettingsPath())
		}
		if raw, ok := root["packages"]; ok {
			var packages []json.RawMessage
			if string(raw) == "null" || json.Unmarshal(raw, &packages) != nil {
				return o, errors.New("malformed Pi packages array")
			}
			for _, entry := range packages {
				var source string
				if json.Unmarshal(entry, &source) != nil {
					var filtered struct {
						Source string `json:"source"`
					}
					if json.Unmarshal(entry, &filtered) != nil || filtered.Source == "" {
						return o, errors.New("malformed Pi package declaration")
					}
					source = filtered.Source
					if source == "npm:"+op.Identity.Name || strings.HasPrefix(source, "npm:"+op.Identity.Name+"@") {
						return o, errors.New("filtered native declaration is unsupported; cannot infer whole-package ownership")
					}
				}
				// Other package source kinds are not ours. Same-name unpinned/ranged
				// declarations are drift, not missing or silently adoptable state.
				bare := "npm:" + op.Identity.Name
				if source == bare || strings.HasPrefix(source, bare+"@") {
					if o.Source != "" {
						return o, errors.New("ambiguous duplicate native declarations")
					}
					o.Source = source
				}
			}
		}
	}
	metadata, err := read(op.Identity.MetadataPath())
	if err != nil {
		return o, err
	}
	if metadata != nil {
		var pkg struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if json.Unmarshal(metadata, &pkg) != nil || pkg.Name != op.Identity.Name {
			return o, fmt.Errorf("malformed/mismatched selected package metadata at %s", op.Identity.MetadataPath())
		}
		_, _, err := ParseSource("npm:" + pkg.Name + "@" + pkg.Version)
		if err != nil {
			return o, err
		}
		o.Version = pkg.Version
	}
	_, version, _ := ParseSource(op.Source)
	switch {
	case o.Source == "" && o.Version == "":
		o.Status = "missing"
	case o.Source != "" && o.Version == "":
		o.Status = "declaration-only"
	case o.Source == "" && o.Version != "":
		o.Status = "payload-only"
	case o.Source == op.Source && o.Version == version:
		o.Status = "present"
	default:
		o.Status = "drifted"
	}
	return o, nil
}

type Invocation struct {
	Argv []string
	Cwd  string
	Env  []string
}
type Runner interface {
	Run(context.Context, Invocation) (string, error)
}

func InvocationFor(op Operation, argv []string) Invocation {
	env := []string{}
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "PI_CODING_AGENT_DIR=") {
			env = append(env, v)
		}
	}
	env = append(env, "PI_CODING_AGENT_DIR="+op.Identity.AgentRoot)
	return Invocation{Argv: argv, Cwd: op.Identity.Project, Env: env}
}

// Probe is apply-only. Success identifies the supported CLI, not plugin loading.
func Probe(ctx context.Context, r Runner, op Operation) error {
	if err := op.Validate(); err != nil {
		return err
	}
	out, err := r.Run(ctx, InvocationFor(op, []string{"pi", "--version"}))
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != RuntimeVersion {
		return fmt.Errorf("unsupported Pi runtime %q; observer requires %s", strings.TrimSpace(out), RuntimeVersion)
	}
	return nil
}

// Reject duplicate keys rather than silently trusting the last declaration or
// metadata identity in malformed JSON. This inspects only the two chosen files.
func uniqueJSON(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	var value func() error
	value = func() error {
		token, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return errors.New("duplicate or invalid JSON key")
				}
				seen[name] = true
				if err := value(); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := value(); err != nil {
					return err
				}
			}
		default:
			return errors.New("unexpected JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := value(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON input")
	}
	return nil
}

// Same is root-qualified npm identity; global execution cwd is not another owner.
func (id Identity) Same(other Identity) bool {
	return id.Name == other.Name && id.Scope == other.Scope && id.Root == other.Root && (id.Scope != "local" || id.Project == other.Project)
}
