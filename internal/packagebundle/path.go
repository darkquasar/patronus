package packagebundle

import (
	"fmt"
	"strings"
)

// ValidatePath checks the original portable member spelling before normalization.
func ValidatePath(name string, directory bool) (string, error) {
	return validatePath(name, directory, 16, 512)
}
func validatePath(name string, directory bool, components, pathBytes int) (string, error) {
	if len(name) > pathBytes {
		return "", fmt.Errorf("member path exceeds %d bytes", pathBytes)
	}
	if directory {
		name = strings.TrimSuffix(name, "/")
	}
	parts := strings.Split(name, "/")
	if len(parts) > components {
		return "", fmt.Errorf("member path exceeds %d components", components)
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid member path %q", name)
		}
		for _, c := range []byte(part) {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.' || c == '@') {
				return "", fmt.Errorf("invalid member path %q", name)
			}
		}
	}
	return name, nil
}

type pathNode struct {
	name                string
	directory, explicit bool
}

func addPath(nodes map[string]pathNode, name string, directory bool) error {
	parts := strings.Split(name, "/")
	for i := range parts {
		p := strings.Join(parts[:i+1], "/")
		key := strings.ToLower(p)
		last := i == len(parts)-1
		dir := !last || directory
		old, ok := nodes[key]
		if ok && (old.name != p || !old.directory || !dir || last && old.explicit) {
			return fmt.Errorf("duplicate or colliding member %q", name)
		}
		nodes[key] = pathNode{name: p, directory: dir, explicit: last || old.explicit}
	}
	return nil
}
