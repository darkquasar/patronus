package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/lock"
)

func TestLaneClosureExcludesForeignSuffixes(t *testing.T) {
	project := fixtureCatalog(t)
	t.Setenv("HOME", t.TempDir())
	t.Chdir(project)
	reg, err := harness.NewRegistry(harness.Descriptor{Name: "toy", ItemSuffix: "tt", Profiles: []string{"toy-profile"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"x-tt", "y-aa"} {
		codexWrite(t, filepath.Join(project, "artifacts/skills", name, "patronus.yaml"), []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: "+name+"\ndescription: invented\nversion: 1.0.0\nentry: SKILL.md\ntargets: [toy, claude]\n"))
		codexWrite(t, filepath.Join(project, "artifacts/skills", name, "SKILL.md"), []byte("invented "+name+" bytes\n"))
	}
	codexWrite(t, filepath.Join(project, "profiles/toy-profile.yaml"), []byte("apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: toy-profile\nversion: 1.0.0\ntargets: [toy]\nlayers:\n  capabilities: [x-tt, y-aa@claude]\n"))
	out, err := runWith(t, newLockCmd(reg), "--profile", "toy-profile", "--target", "toy", "--local-registry")
	if err != nil {
		t.Fatalf("lock: %v\n%s", err, out)
	}
	l, err := lock.Load(filepath.Join(project, "patronus.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if l.Version != lock.Version || l.Target != "" || len(l.Entries) != 1 || l.Entries[0].Name != "x-tt" {
		t.Fatalf("toy closure: %+v", l)
	}
	for _, e := range l.Entries {
		if strings.HasSuffix(e.Name, "-aa") || strings.Contains(e.Name, "@claude") {
			t.Fatalf("foreign member leaked: %+v", e)
		}
	}
	if out, err := runWith(t, newLockCmd(reg), "--profile", "toy-profile", "--target", "all", "--local-registry"); err == nil || !strings.Contains(err.Error(), "toy-profile requires --target toy") {
		t.Fatalf("foreign target admitted: %v\n%s", err, out)
	}
}
