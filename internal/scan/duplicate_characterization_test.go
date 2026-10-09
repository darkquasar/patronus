package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Characterization of the two duplicate-identity policies at the base revision
// (HP-03a): Pi refuses a same-name skill in two roots regardless of bytes
// (pi_discovery.go:324-327, 533-549); Codex admits identical bytes and refuses
// differing bytes unless the caller proves both copies are verified, unchanged
// Codex installations (codex_discovery.go:110-118).

func harnessSkillDir(t *testing.T, root, name, body string) string {
	t.Helper()
	path := filepath.Join(root, name, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHarnessDuplicatePolicyPiRefusesIdenticalBytes(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	body := "---\nname: fixture-dup\ndescription: invented\n---\nBody\n"
	pa := harnessSkillDir(t, a, "fixture-dup", body)
	pb := harnessSkillDir(t, b, "fixture-dup", body)
	roots := []PiResourceRoot{{Kind: "skill", Path: a, Origin: "first"}, {Kind: "skill", Path: b, Origin: "second"}}
	_, err := DiscoverPiResources(roots)
	want := "pi skill identity \"fixture-dup\" conflict: " + pa + " (first) and " + pb + " (second); silent shadowing refused"
	if err == nil || err.Error() != want {
		t.Fatalf("pi identical bytes\n got: %v\nwant: %s", err, want)
	}
}

func TestHarnessDuplicatePolicyCodex(t *testing.T) {
	body := "---\nname: fixture-dup\ndescription: invented\n---\nBody\n"
	for _, tc := range []struct {
		name     string
		second   string
		verified func(CodexSkill) bool
		wantErr  bool
	}{
		{"identical bytes admitted", body, nil, false},
		{"different bytes refused", body + "more\n", nil, true},
		{"different bytes refused when only one is verified", body + "more\n", func(r CodexSkill) bool { return strings.HasSuffix(filepath.Dir(filepath.Dir(r.Path)), "a") }, true},
		{"different bytes admitted when both verified", body + "more\n", func(CodexSkill) bool { return true }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			a, b := filepath.Join(base, "a"), filepath.Join(base, "b")
			pa := harnessSkillDir(t, a, "fixture-dup", body)
			pb := harnessSkillDir(t, b, "fixture-dup", tc.second)
			got, err := DiscoverCodexSkillsWithOwnership([]CodexSkillRoot{{Path: a}, {Path: b}}, tc.verified)
			if tc.wantErr {
				want := "codex skill name \"fixture-dup\" conflicts at " + pa + " and " + pb + "; incompatible same-name resources cannot be shadowed"
				if err == nil || err.Error() != want {
					t.Fatalf("codex conflict\n got: %v\nwant: %s", err, want)
				}
				return
			}
			if err != nil || len(got) != 2 {
				t.Fatalf("codex admitted pair: %v %+v", err, got)
			}
		})
	}
}

// A legacy Codex root never benefits from the verified-pair exception.
func TestHarnessDuplicatePolicyCodexLegacyNoException(t *testing.T) {
	base := t.TempDir()
	a, b := filepath.Join(base, "a"), filepath.Join(base, "b")
	body := "---\nname: fixture-dup\ndescription: invented\n---\nBody\n"
	harnessSkillDir(t, a, "fixture-dup", body)
	harnessSkillDir(t, b, "fixture-dup", body+"more\n")
	_, err := DiscoverCodexSkillsWithOwnership([]CodexSkillRoot{{Path: a, Legacy: true}, {Path: b}}, func(CodexSkill) bool { return true })
	if err == nil || !strings.Contains(err.Error(), "incompatible same-name resources cannot be shadowed") {
		t.Fatalf("legacy pair admitted: %v", err)
	}
}
