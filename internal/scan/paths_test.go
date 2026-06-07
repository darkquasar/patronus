package scan

import (
	"path/filepath"
	"testing"
)

func envFrom(m map[string]string) EnvLookup {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

// The detailed marker-resolution behavior is covered in internal/toolpath. This
// verifies scan's thin wrapper delegates correctly across both scopes (mapping
// scan.Scope -> the string scope toolpath expects).
func TestResolverWrapperDelegates(t *testing.T) {
	r := newResolver(envFrom(map[string]string{"HOME": "/home/u"}), "/home/u", "/proj")

	if got, want := r.resolveMarker("~/.claude/", "claude", ScopeGlobal), filepath.Join("/home/u", ".claude"); got != want {
		t.Errorf("global: got %q, want %q", got, want)
	}
	if got, want := r.resolveMarker(".mcp.json", "claude", ScopeLocal), filepath.Join("/proj", ".mcp.json"); got != want {
		t.Errorf("local: got %q, want %q", got, want)
	}
}
