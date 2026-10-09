package scan

import (
	"fmt"
	"os"
	"path/filepath"
)

// SafePath rejects aliases at every existing component, including the root.
// Missing final components are allowed for a prospective destination. The
// messages keep the historical "pi path" prefix; target-specific callers wrap
// them (CodexSafePath adds "codex admission").
func SafePath(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("pi path %q must be canonical and absolute", path)
	}
	for p := path; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("pi path %s: %w", p, err)
		}
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("pi path %s: unapproved symlink/redirection", p)
			}
			if p != path && !info.IsDir() {
				return fmt.Errorf("pi path %s: non-directory ancestor", p)
			}
			if p == path && !info.IsDir() && !info.Mode().IsRegular() {
				return fmt.Errorf("pi path %s: nonregular input", p)
			}
			if info.Mode().Perm()&0444 == 0 {
				return fmt.Errorf("pi path %s: unreadable input", p)
			}
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return nil
}
