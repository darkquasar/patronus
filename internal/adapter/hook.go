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
// removal surgical. Codex script hooks must have both placement and registration
// targets. Other tools retain their existing unsupported-surface behavior.
func (e *Engine) transformHook(art *manifest.Artifact, ad *manifest.Adapter, scope, srcDir string, readExisting ReadExisting) ([]diff.FileDiff, error) {
	if ad.Layout.Hook == nil {
		return nil, fmt.Errorf("adapter %q: no Hook layout", ad.Tool)
	}
	spec := art.Hook
	if spec == nil {
		return nil, fmt.Errorf("adapter: hook artifact %q missing hook block", art.Name)
	}

	// A tool without a native hook dispatcher declares its hook surface as
	// data: a script-less gate becomes a declarative permission deny, a nudge
	// has no delivery (its companion instruction carries the advice), and a
	// script hook registers in the Patronus-owned registry the adapter-embedded
	// runner reads. Tools with a native dispatcher fall through unchanged.
	if spec.Intent == manifest.HookGate && spec.Script == "" && ad.Layout.Hook.DeclarativeGate != nil {
		return e.transformDeclarativeGate(art, ad, scope, spec, readExisting)
	}
	if (spec.Intent == manifest.HookNudge || spec.Intent == "") && ad.Layout.Hook.RunnerDialect != nil {
		return nil, nil
	}

	target := ad.Layout.Hook.ForScope(scope)
	scriptDirs := ad.Tool == "codex" || ad.Layout.Hook.RunnerDialect != nil ||
		ad.Layout.Hook.GlobalScriptDir.OK() || ad.Layout.Hook.ProjectScriptDir.OK() ||
		ad.Layout.Hook.GlobalRunnerDir.OK() || ad.Layout.Hook.ProjectRunnerDir.OK()
	if !target.OK() {
		if scriptDirs && spec.Script != "" {
			return nil, fmt.Errorf("adapter %s: script hook %q has no %s registration target", ad.Tool, art.Name, scope)
		}
		return nil, nil
	}

	if spec.Script != "" && !ad.Layout.Hook.ScriptDirFor(scope).OK() {
		if scriptDirs {
			return nil, fmt.Errorf("adapter %s: hook %q ships a script but has no %s hook-script dir", ad.Tool, art.Name, scope)
		}
		return nil, nil
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

	var warning string
	if ad.Tool == "codex" {
		warning = "Codex static hook script placement and structured registration only: event/matcher/payload semantics and native trust remain runtime-unverified; Patronus does not write trust internals"
	}
	diffs = append(diffs, diff.FileDiff{
		Warning: warning,
		Path:    path,
		Action:  diff.Merge,
		Before:  existing,
		After:   after,
		Tool:    ad.Tool,
		Scope:   scope,
		Role:    string(art.Role),
		Note:    "hook " + spec.Event + ": " + art.Name,
		Setting: &diff.SettingEdit{
			Target:      diff.FileTargetRef{File: target.File, Format: target.Format},
			Dotted:      dotted,
			IdentityKey: patronusHookID,
			Identity:    identity,
			Elem:        elem,
		},
	})
	// A tool that reads the registry through the adapter-embedded runner
	// receives the runner next to the registration. Every script hook places
	// the identical bytes; composition records them as one shared placed file.
	if dir := ad.Layout.Hook.RunnerDirFor(scope); dir.OK() && spec.Script != "" {
		if ad.Layout.Hook.RunnerDialect == nil {
			return nil, fmt.Errorf("adapter %s: hook runner for %q: missing runner dialect", ad.Tool, art.Name)
		}
		runner, err := renderHookRunner(*ad.Layout.Hook.RunnerDialect)
		if err != nil {
			return nil, fmt.Errorf("adapter %s: hook runner for %q: %w", ad.Tool, art.Name, err)
		}
		diffs = append(diffs, diff.FileDiff{
			Path:   filepath.Join(e.resolver.ResolveMarker(dir.Path, ad.Tool, scope), hookRunnerFile),
			Action: diff.Create,
			After:  runner,
			Mode:   0o644,
			Tool:   ad.Tool,
			Scope:  scope,
			Role:   string(art.Role),
			Note:   "hook runner: " + art.Name,
		})
	}
	return diffs, nil
}

// transformDeclarativeGate maps a script-less gate hook to the tool's
// declarative permission config: one <gate.Path>.<key> = "deny" per distinct
// key the matcher's tokens map to through the adapter's DeclarativeGate table,
// written into the setting layout file. Such a tool has no hook dispatcher for
// a command-less gate, so the gate is a deny rule; its keys are single names,
// so a Claude alternation like Write|Edit|MultiEdit is split, mapped and
// collapsed. A gate always denies.
//
// A token with no mapping is dropped with a warning rather than wired as a
// bogus key the tool would silently ignore; when EVERY token is unmappable
// there is no honest deny to emit, so that is an error, not an empty plan.
func (e *Engine) transformDeclarativeGate(art *manifest.Artifact, ad *manifest.Adapter, scope string, spec *manifest.HookSpec, readExisting ReadExisting) ([]diff.FileDiff, error) {
	// Declarative denies live in the agent's settings file (the adapter's
	// setting layout), never in a hooks registry.
	if ad.Layout.Setting == nil {
		return nil, fmt.Errorf("adapter %q: declarative gate %q needs a setting layout", ad.Tool, art.Name)
	}
	target := ad.Layout.Setting.ForScope(scope)
	if !target.OK() {
		return nil, nil
	}
	if spec.Matcher == "" {
		return nil, fmt.Errorf("adapter: %s gate hook %q needs a matcher (the permission key to deny)", ad.Tool, art.Name)
	}
	gate := ad.Layout.Hook.DeclarativeGate
	keys, dropped := declarativeGateKeys(gate, spec.Matcher)
	if len(keys) == 0 {
		return nil, fmt.Errorf("adapter: %s gate hook %q has no matcher token that maps to a %s permission key (matcher %q); it cannot be gated on %s", ad.Tool, art.Name, ad.Tool, spec.Matcher, ad.Tool)
	}

	path := e.resolver.ResolveMarker(target.File, ad.Tool, scope)
	existing, _, err := readExisting(path)
	if err != nil {
		return nil, fmt.Errorf("adapter: read settings for gate %q: %w", art.Name, err)
	}

	var warning string
	if len(dropped) > 0 {
		// Partial wire: name the dropped tokens so the blind spot is visible, never
		// a silent skip. The mappable keys below still deny.
		warning = fmt.Sprintf("gate %q: matcher token(s) %s have no %s permission key — denied only %s", art.Name, strings.Join(dropped, ", "), ad.Tool, strings.Join(keys, ", "))
	}

	// Each key is an independent scalar deny, threaded through MergeSettings so the
	// composed file carries every deny, and each gets its own SettingEdit so remove
	// and drift key on the exact permission entry.
	diffs := make([]diff.FileDiff, 0, len(keys))
	cur := existing
	for _, key := range keys {
		dotted := gate.Path + "." + key
		// Read before merging: cur accumulates across keys, so the prior must be
		// captured from the bytes as they stand before THIS key is set.
		prior, priorPresent, err := ReadDotted(cur, target, dotted)
		if err != nil {
			return nil, fmt.Errorf("adapter: wire gate %q: read prior: %w", art.Name, err)
		}
		after, err := MergeSettings(cur, target, dotted, "deny")
		if err != nil {
			return nil, fmt.Errorf("adapter: wire gate %q: %w", art.Name, err)
		}
		diffs = append(diffs, diff.FileDiff{
			Path:    path,
			Action:  diff.Merge,
			Before:  cur,
			After:   after,
			Tool:    ad.Tool,
			Scope:   scope,
			Role:    string(art.Role),
			Note:    "gate " + key + ": " + art.Name,
			Warning: warning,
			Setting: &diff.SettingEdit{
				Target:       diff.FileTargetRef{File: target.File, Format: target.Format},
				Dotted:       dotted,
				ScalarValue:  "deny",
				PriorValue:   prior,
				PriorPresent: priorPresent,
			},
		})
		cur = after
		warning = "" // attach the advisory once, to the first diff only
	}
	return diffs, nil
}

// declarativeGateKeys splits a Claude matcher on "|", maps each token through
// the gate table (case-insensitively), and returns the DISTINCT keys (input
// order preserved) plus the tokens that had no mapping.
func declarativeGateKeys(gate *manifest.DeclarativeGate, matcher string) (keys, dropped []string) {
	seen := map[string]bool{}
	for tok := range strings.SplitSeq(matcher, "|") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		key, ok := gate.Tools[strings.ToLower(tok)]
		if !ok {
			dropped = append(dropped, tok)
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	return keys, dropped
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
