package main

import (
	"testing"

	"github.com/darkquasar/patronus/internal/registry"
)

// TestCheckTokensFlags: the malformed shapes the guard exists to catch.
func TestCheckTokensFlags(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		token string
	}{
		{name: "dropped-letter", body: "run {skilDir}/x.sh", token: "{skilDir}"},
		{name: "dropped-letter-plural", body: "run {skilsDir}/x.sh", token: "{skilsDir}"},
		{name: "wrong-case", body: "run {SkillDir}/x.sh", token: "{SkillDir}"},
		{name: "singular-plural-swap", body: "run {skillDirs}/x.sh", token: "{skillDirs}"},
		{name: "doubled-letter", body: "run {skilllDir}/x.sh", token: "{skilllDir}"},
		{name: "underscored", body: "run {skill_dir}/x.sh", token: "{skill_dir}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkTokens("SKILL.md", []byte(tt.body))
			if len(got) != 1 {
				t.Fatalf("want 1 finding, got %d: %v", len(got), got)
			}
			if got[0].Token != tt.token {
				t.Errorf("token = %q, want %q", got[0].Token, tt.token)
			}
			if got[0].Line != 1 {
				t.Errorf("line = %d, want 1", got[0].Line)
			}
		})
	}
}

// TestCheckTokensAllows: shapes a real skill body legitimately carries. A false
// positive here would make the guard unusable, which is why unknown braces are
// left alone by design.
func TestCheckTokensAllows(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "known-token", body: "run {skillDir}/scripts/verify.sh"},
		{name: "known-token-plural", body: "run {skillsDir}/sibling/serve.sh"},
		{name: "both-known", body: "{skillDir} and {skillsDir}"},
		{name: "js-template-literal", body: "return `digraph ${skillName}_combined {`"},
		{name: "shell-expansion", body: `echo "${skillDir_override}"`},
		{name: "other-catalog-token", body: "command: {installPath} --context {toolContext}"},
		{name: "json-braces", body: `{"skills": ["a"]}`},
		{name: "python-fstring", body: `f"{skip_count} skipped"`},
		{name: "ski-prefixed-variable", body: `f"{skimmed} and {skipped}"`},
		{name: "no-braces", body: "a plain skill body mentioning skills and skillDir"},
		// The pattern needs BOTH a ski… prefix and a dir suffix, so a typo that
		// mangles both halves is out of scope — accepted so the guard never fires
		// on ordinary code. See skillTokenPattern.
		{name: "out-of-scope-both-halves-mangled", body: "run {skillPath}/x.sh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkTokens("SKILL.md", []byte(tt.body)); len(got) != 0 {
				t.Errorf("want no findings, got %v", got)
			}
		})
	}
}

func TestCheckTokensReportsLineAndFile(t *testing.T) {
	body := "line one\nline two has {skilDir}\nline three {skillDir} is fine\nline four {skillsdir}\n"
	got := checkTokens("artifacts/skills/x/SKILL.md", []byte(body))
	if len(got) != 2 {
		t.Fatalf("want 2 findings, got %d: %v", len(got), got)
	}
	if got[0].Line != 2 || got[0].Token != "{skilDir}" {
		t.Errorf("first = %+v, want line 2 {skilDir}", got[0])
	}
	if got[1].Line != 4 || got[1].Token != "{skillsdir}" {
		t.Errorf("second = %+v, want line 4 {skillsdir}", got[1])
	}
	if got[0].File != "artifacts/skills/x/SKILL.md" {
		t.Errorf("file = %q, want the path passed in", got[0].File)
	}
}

func TestCheckTokensSkipsNonUTF8(t *testing.T) {
	binary := []byte{0xff, 0xfe, '{', 's', 'k', 'i', 'l', 'D', 'i', 'r', '}', 0x00}
	if got := checkTokens("assets/logo.bin", binary); len(got) != 0 {
		t.Errorf("want no findings for a binary asset, got %v", got)
	}
}

// TestCheckTokensRealCatalogIsClean is the guard actually guarding: every shipped
// artifact body must be free of malformed tokens.
func TestCheckTokensRealCatalogIsClean(t *testing.T) {
	root, err := registry.DiscoverRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	bad, err := scanArtifactTokens(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range bad {
		t.Errorf("shipped catalog carries a malformed token: %s", b)
	}
}
