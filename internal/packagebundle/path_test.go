package packagebundle

import (
	"strings"
	"testing"
)

func TestValidatePathLiteralAt(t *testing.T) {
	for _, name := range []string{"@example", "name@revision", "pi/node_modules/@example/tool/index.js", "pi/node_modules/@example/tool/node_modules/@nested/helper/index.js"} {
		t.Run(name, func(t *testing.T) {
			got, err := ValidatePath(name, false)
			if err != nil || got != name {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
	for _, name := range []string{"@example/../escape", "@example/./file", "/@example/file", "@example//file", "@example\\file", "@example/C:/file", "@example/\x00file", "@example/é", "@example/"} {
		t.Run("unsafe "+name, func(t *testing.T) {
			if _, err := ValidatePath(name, false); err == nil {
				t.Fatalf("accepted unsafe path %q", name)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	for _, name := range []string{"", "../x", "/x", "a//b", "a/./b", "a\\b", "C:/x", "a/../b", "a\x00b", "é", "a/", strings.Repeat("x", 513), strings.Repeat("a/", 16) + "b"} {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidatePath(name, false); err == nil {
				t.Fatalf("accepted %q", name)
			}
		})
	}
	got, err := ValidatePath("config/", true)
	if err != nil || got != "config" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := ValidatePath("config//", true); err == nil {
		t.Fatal("accepted double trailing slash")
	}
}
