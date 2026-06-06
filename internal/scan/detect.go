package scan

import "os"

// detectScope resolves each marker for a tool/scope and reports detection plus
// the resolved paths that actually exist on disk.
func detectScope(res *resolver, tool string, scope Scope, markers []string) *Detection {
	det := &Detection{Scope: scope, MatchedPaths: []string{}}
	for _, marker := range markers {
		path := res.resolveMarker(marker, tool, scope)
		if path == "" {
			continue
		}
		if pathExists(path) {
			det.Detected = true
			det.MatchedPaths = append(det.MatchedPaths, path)
		}
	}
	return det
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
