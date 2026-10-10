package adapter

import (
	"path/filepath"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// RedactedPlaceholder replaces every value a redact glob matches in a rendered
// view. It is presentation only; applied bytes never carry it. Square brackets
// because the JSON serializer HTML-escapes angle brackets.
const RedactedPlaceholder = "[redacted]"

// StampRedact copies the adapter's setting-layout redact globs onto every diff
// whose path is that layout's file at either scope. It is the one place the
// adapter engine, the recipe engine and the removal inverse path share, so a
// rendered view of a settings file is redacted however the diff was produced.
// Diffs on other paths and adapters without redact data are untouched.
func StampRedact(diffs []diff.FileDiff, ad *manifest.Adapter, res toolpath.Resolver) {
	if ad == nil || ad.Layout.Setting == nil || len(ad.Layout.Setting.Redact) == 0 {
		return
	}
	files := map[string]bool{}
	for _, scope := range []string{"global", "local"} {
		if target := ad.Layout.Setting.ForScope(scope); target.OK() {
			files[res.ResolveMarker(target.File, ad.Tool, scope)] = true
		}
	}
	for i := range diffs {
		if files[diffs[i].Path] {
			diffs[i].Redact = append([]string(nil), ad.Layout.Setting.Redact...)
		}
	}
}

// RedactForDisplay returns a copy of d whose Before and After have every value
// matching d.Redact replaced by RedactedPlaceholder, for rendering only. Bytes
// a glob does not touch come back unchanged; bytes that do not parse are
// replaced wholesale, since an unparseable file cannot be proven free of the
// values it hides. A diff without Redact is returned as is.
func RedactForDisplay(d diff.FileDiff) diff.FileDiff {
	if len(d.Redact) == 0 {
		return d
	}
	format := formatOfPath(d.Path)
	out := d
	out.Before = redactBytes(d.Before, format, d.Redact)
	out.After = redactBytes(d.After, format, d.Redact)
	return out
}

func redactBytes(raw []byte, format string, globs []string) []byte {
	if len(raw) == 0 {
		return raw
	}
	root, err := parseConfig(raw, format)
	if err != nil {
		return []byte("[redacted: unparseable " + format + " not shown]\n")
	}
	if !RedactDotted(root, globs) {
		return raw
	}
	shown, err := serializeConfig(root, format)
	if err != nil {
		return []byte("[redacted: could not re-render " + format + "]\n")
	}
	return shown
}

// RedactDotted replaces, in place, every value under root whose dotted path
// matches one of globs with RedactedPlaceholder. A glob is a dotted path whose
// segments are literal keys or "*" (one segment). It reports whether anything
// was replaced.
func RedactDotted(root map[string]any, globs []string) bool {
	changed := false
	for _, glob := range globs {
		segments := strings.Split(glob, ".")
		if len(segments) == 0 || glob == "" {
			continue
		}
		changed = redactWalk(root, segments) || changed
	}
	return changed
}

func redactWalk(node map[string]any, segments []string) bool {
	changed := false
	for key, value := range node {
		if segments[0] != "*" && segments[0] != key {
			continue
		}
		if len(segments) == 1 {
			node[key] = RedactedPlaceholder
			changed = true
			continue
		}
		child, ok := asStringMap(value)
		if !ok {
			continue
		}
		if redactWalk(child, segments[1:]) {
			node[key] = child
			changed = true
		}
	}
	return changed
}

// formatOfPath derives the structured format of a settings file from its
// extension; the layouts that declare redact data are JSON, JSONC or TOML.
func formatOfPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".toml":
		return "toml"
	case ".jsonc":
		return "jsonc"
	default:
		return "json"
	}
}
