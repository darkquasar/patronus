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
	spec := art.Hook
	if spec == nil {
		return nil, fmt.Errorf("adapter: hook artifact %q missing hook block", art.Name)
	}

	// OpenCode has no declarative hooks block: a gate maps to its `permission`
	// config; a nudge is delivered by a paired instruction (AGENTS.md), so it emits
	// no hook diff here. Other tools fall through to the settings-list merge below.
	if ad.Tool == "opencode" {
		if spec.Intent == manifest.HookGate {
			return e.transformGateOpenCode(art, ad, scope, spec, readExisting)
		}
		return nil, nil // nudge: the paired instruction artifact carries it on OpenCode
	}

	target := ad.Layout.Hook.ForScope(scope)
	if !target.OK() {
		return nil, nil // tool models no hook surface at this scope — honest skip
	}

	// A script-bearing hook can only wire on a tool that has somewhere to put the
	// script. A tool with a hook surface but NO hook-script dir (Codex references an
	// absolute command; it places no bundled script) is an honest skip for such a
	// hook, not an error — a cross-tool profile installs cleanly and the hook lands
	// only where its script can. A hook that inlines its command (no script) still
	// wires everywhere.
	if spec.Script != "" && !ad.Layout.Hook.ScriptDirFor(scope).OK() {
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

// _claudeToOpenCode maps a Claude PreToolUse matcher token to the OpenCode
// permission key that gates the same capability. OpenCode keys are single
// lowercase tool names (not pipe-alternations), and its `edit` permission covers
// write/edit/patch — so Write, Edit, and MultiEdit all deny under `edit`. Tokens
// with no OpenCode tool (TodoWrite is Claude-only) are absent: they cannot be
// honestly gated on OpenCode and are dropped with a warning. Keys are matched
// case-insensitively so both "Bash" and "bash" resolve.
var _claudeToOpenCode = map[string]string{
	"write":     "edit",
	"edit":      "edit",
	"multiedit": "edit",
	"bash":      "bash",
	"read":      "read",
	"grep":      "grep",
	"glob":      "glob",
	"webfetch":  "webfetch",
	"websearch": "websearch",
	"task":      "task",
}

// transformGateOpenCode maps a gate hook to OpenCode's declarative permission
// config: one permission.<tool> = "deny" per distinct OpenCode tool the matcher
// names. OpenCode has no hooks block, so a gate is a deny rule, not a PreToolUse
// handler; and its keys are single lowercase tool names, so a Claude alternation
// like Write|Edit|MultiEdit must be split and mapped (here, collapsing to the one
// `edit` key). A gate always denies — ask/allow are not a gate's job.
//
// A token with no OpenCode equivalent (Claude-only, e.g. TodoWrite) is dropped
// with a warning rather than wired as a bogus key that OpenCode would silently
// ignore — the pat-8jow no-op. When EVERY token is unmappable there is no honest
// deny to emit, so that is an error, not a silent empty plan.
func (e *Engine) transformGateOpenCode(art *manifest.Artifact, ad *manifest.Adapter, scope string, spec *manifest.HookSpec, readExisting ReadExisting) ([]diff.FileDiff, error) {
	target := ad.Layout.Hook.ForScope(scope)
	if !target.OK() {
		return nil, nil
	}
	if spec.Matcher == "" {
		return nil, fmt.Errorf("adapter: opencode gate hook %q needs a matcher (the permission key to deny)", art.Name)
	}

	keys, dropped := openCodePermissionKeys(spec.Matcher)
	if len(keys) == 0 {
		return nil, fmt.Errorf("adapter: opencode gate hook %q has no matcher token that maps to an opencode permission key (matcher %q); it cannot be gated on opencode", art.Name, spec.Matcher)
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
		warning = fmt.Sprintf("gate %q: matcher token(s) %s have no opencode permission key — denied only %s", art.Name, strings.Join(dropped, ", "), strings.Join(keys, ", "))
	}

	// Each key is an independent scalar deny, threaded through MergeSettings so the
	// composed file carries every deny, and each gets its own SettingEdit so remove
	// and drift key on the exact permission entry.
	diffs := make([]diff.FileDiff, 0, len(keys))
	cur := existing
	for _, key := range keys {
		dotted := target.Path + "." + key // permission.<key>
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

// openCodePermissionKeys splits a Claude matcher on "|", maps each token to its
// OpenCode permission key, and returns the DISTINCT keys (input order preserved)
// plus the tokens that had no mapping. A token is looked up case-insensitively.
func openCodePermissionKeys(matcher string) (keys, dropped []string) {
	seen := map[string]bool{}
	for tok := range strings.SplitSeq(matcher, "|") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		key, ok := _claudeToOpenCode[strings.ToLower(tok)]
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
