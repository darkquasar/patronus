package main

import (
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
)

func TestDirectoryHasNoBinaryPathAdvice(t *testing.T) {
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Path: "/home/user/.patronus/packages/pi-sandbox", Action: diff.Fetch, Directory: &diff.DirectorySpec{Recipe: "pi-sandbox"}}}}
	if rows := pathReadiness(cs, nil); len(rows) != 0 {
		t.Fatalf("directory treated as binary: %#v", rows)
	}
}
