package scan

import "github.com/darkquasar/patronus/internal/toolpath"

// EnvLookup aliases the shared toolpath type so existing scan callers and tests
// keep working unchanged.
type EnvLookup = toolpath.EnvLookup

// resolver wraps the shared toolpath.Resolver, adapting scan's Scope type to the
// string scope the resolver expects.
type resolver struct {
	r toolpath.Resolver
}

func newResolver(env EnvLookup, home, projectDir string) *resolver {
	return &resolver{r: toolpath.New(env, home, projectDir)}
}

// homeDir resolves the user's home directory from the environment.
func homeDir(env EnvLookup) string { return toolpath.HomeDir(env) }

// resolveMarker resolves a marker for the given tool/scope to an absolute path.
func (r *resolver) resolveMarker(marker, tool string, scope Scope) string {
	return r.r.ResolveMarker(marker, tool, string(scope))
}
