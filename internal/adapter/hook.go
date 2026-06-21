package adapter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// patronusHookID is the marker field Patronus stamps on every hook array element
// it writes. It is both the idempotence key (re-install replaces in place) and
// the handle remove uses to strip exactly our element, leaving user-added and
// other-artifact hooks on the same event untouched.
const patronusHookID = "patronusId"

// transformHook registers a hook artifact into the agent's settings file. The
// hook is one element appended to the array at hooks.{event}; its identity (a
// digest of artifact+matcher+command) makes the append idempotent and the
// removal surgical. Tools whose hook surface is unmodeled (Codex, OpenCode today)
// carry a null Hook layout target — for them a hook artifact is a no-op rather
// than an error, so a cross-tool profile installs cleanly and only the tools
// that support hooks get them.
func (e *Engine) transformHook(art *manifest.Artifact, ad *manifest.Adapter, scope, srcDir string, readExisting ReadExisting) ([]diff.FileDiff, error) {
	if ad.Layout.Hook == nil {
		return nil, fmt.Errorf("adapter %q: no Hook layout", ad.Tool)
	}
	target := ad.Layout.Hook.ForScope(scope)
	if !target.OK() {
		return nil, nil // tool models no hook surface at this scope — honest skip
	}
	spec := art.Hook
	if spec == nil {
		return nil, fmt.Errorf("adapter: hook artifact %q missing hook block", art.Name)
	}

	// A hook may ship a helper script: place it (CREATE) and resolve the command's
	// {script} token to its installed path before the registration is built, so the
	// settings entry invokes exactly the placed file.
	var diffs []diff.FileDiff
	command := spec.Command
	if spec.Script != "" {
		place, scriptPath, err := e.placeHookScript(art, ad, scope, srcDir, spec.Script)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, place)
		command = strings.ReplaceAll(command, "{script}", scriptPath)
	}

	identity := hookIdentity(art.Name, spec)
	elem := hookElement(spec, command, identity)
	dotted := strings.ReplaceAll(target.Path, "{event}", spec.Event)
	path := e.resolver.ResolveMarker(target.File, ad.Tool, scope)

	existing, _, err := readExisting(path)
	if err != nil {
		return nil, fmt.Errorf("adapter: read settings for hook %q: %w", art.Name, err)
	}
	after, err := AppendSettingsList(existing, target, dotted, patronusHookID, elem)
	if err != nil {
		return nil, fmt.Errorf("adapter: wire hook %q: %w", art.Name, err)
	}

	return append(diffs, diff.FileDiff{
		Path:   path,
		Action: diff.Merge,
		Before: existing,
		After:  after,
		Tool:   ad.Tool,
		Scope:  scope,
		Role:   string(art.Role),
		Note:   "hook " + spec.Event + ": " + art.Name,
		Setting: &diff.SettingEdit{
			Target:      diff.FileTargetRef{File: target.File, Format: target.Format},
			Dotted:      dotted,
			IdentityKey: patronusHookID,
			Identity:    identity,
			Elem:        elem,
		},
	}), nil
}

// placeHookScript emits the CREATE diff that writes the hook's bundled helper
// script into the tool's hook-script dir, and returns that diff plus the absolute
// installed path (for {script} substitution in the command). It errors if the
// tool models a hook surface but no script dir — a hook artifact that ships a
// script can only target a tool that knows where to put it.
func (e *Engine) placeHookScript(art *manifest.Artifact, ad *manifest.Adapter, scope, srcDir, script string) (diff.FileDiff, string, error) {
	dir := ad.Layout.Hook.ScriptDirFor(scope)
	if !dir.OK() {
		return diff.FileDiff{}, "", fmt.Errorf("adapter %q: hook %q ships a script but the tool has no %s hook-script dir", ad.Tool, art.Name, scope)
	}
	body, err := os.ReadFile(filepath.Join(srcDir, script))
	if err != nil {
		return diff.FileDiff{}, "", fmt.Errorf("adapter: read hook script %q: %w", script, err)
	}
	scriptDir := e.resolver.ResolveMarker(dir.Path, ad.Tool, scope)
	scriptPath := filepath.Join(scriptDir, art.Name+filepath.Ext(script))
	return diff.FileDiff{
		Path:   scriptPath,
		Action: diff.Create,
		After:  body,
		Mode:   0o755, // a hook script must be executable
		Tool:   ad.Tool,
		Scope:  scope,
		Role:   string(art.Role),
		Note:   "hook script: " + art.Name,
	}, scriptPath, nil
}

// hookElement renders one Claude-shaped hook matcher-group:
//
//	{ "matcher": "...", "patronusId": "...",
//	  "hooks": [ { "type": "command", "command": "...", "timeout": N } ] }
//
// The matcher key is omitted when empty (an "all tools" hook), mirroring how the
// agent itself treats an absent matcher. The handler type defaults to "command".
// command is the resolved command (with any {script} token already substituted
// to the placed script path), not spec.Command verbatim.
func hookElement(spec *manifest.HookSpec, command, identity string) map[string]any {
	handler := map[string]any{
		"type":    hookType(spec.Type),
		"command": command,
	}
	if spec.Timeout > 0 {
		handler["timeout"] = spec.Timeout
	}
	elem := map[string]any{
		patronusHookID: identity,
		"hooks":        []any{handler},
	}
	if spec.Matcher != "" {
		elem["matcher"] = spec.Matcher
	}
	return elem
}

// hookType returns the handler type, defaulting to "command".
func hookType(t string) string {
	if t == "" {
		return "command"
	}
	return t
}

// hookIdentity is a stable per-artifact-per-hook id: a short digest over the
// artifact name and the hook's event/matcher/command. It is stable across
// re-installs (so the append is idempotent) and unique per hook (so two hooks on
// one event don't collide), with the artifact name making it human-traceable.
func hookIdentity(name string, spec *manifest.HookSpec) string {
	sum := sha256.Sum256([]byte(name + "\x00" + spec.Event + "\x00" + spec.Matcher + "\x00" + spec.Command))
	return "patronus:" + name + ":" + hex.EncodeToString(sum[:])[:8]
}
