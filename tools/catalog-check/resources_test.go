package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDistributedReferenceContracts(t *testing.T) {
	for _, tc := range []struct {
		name, body, diagnostic string
	}{
		{"valid", "[local]({skillDir}/notes.txt) [sibling]({skillsDir}/sample/notes.txt)", ""},
		{"missing", "[missing]({skillDir}/absent.md)", "missing distributed reference"},
		{"literal-example", "````markdown\n[example](absent.md)\n```\n[still example](absent.md)\n````", ""},
		{"absolute-author", "Read /home/invented/private.md", "author-machine absolute path"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.members["SKILL.md"] = "---\nname: sample\n---\n" + tc.body + "\n"
			writeFixtureFile(t, filepath.Join(f.root, "artifacts/skills/sample/SKILL.md"), []byte(f.members["SKILL.md"]))
			f.publish(t, nil)
			code, _, diagnostic := f.check(t)
			if tc.diagnostic == "" {
				if code != 0 {
					t.Fatal(diagnostic)
				}
			} else if code == 0 || !strings.Contains(diagnostic, tc.diagnostic) {
				t.Fatalf("code=%d diagnostic=%s", code, diagnostic)
			}
		})
	}
}

func TestReferenceExceptionBoundToManifestAndBody(t *testing.T) {
	f := newFixture(t)
	body := "---\nname: sample\n---\n[upstream](absent.md)\n"
	f.members["SKILL.md"] = body
	writeFixtureFile(t, filepath.Join(f.root, "artifacts/skills/sample/SKILL.md"), []byte(body))
	manifest, err := os.ReadFile(filepath.Join(f.root, "artifacts/skills/sample/patronus.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(append(manifest, []byte(body)...))
	policy := fmt.Sprintf("schema_version: 1\nexceptions:\n  - source: artifacts/skills/sample/SKILL.md\n    manifest_content_sha256: %x\n    references: [absent.md]\n    upstream_only: true\n    reason: Invented reviewed upstream reference\n", digest)
	writeFixtureFile(t, filepath.Join(f.root, "docs/compatibility/distributed-reference-exceptions.yaml"), []byte(policy))
	f.publish(t, nil)
	if code, _, diagnostic := f.check(t); code != 0 {
		t.Fatal(diagnostic)
	}
	f.members["SKILL.md"] += "Changed bytes.\n"
	writeFixtureFile(t, filepath.Join(f.root, "artifacts/skills/sample/SKILL.md"), []byte(f.members["SKILL.md"]))
	f.publish(t, nil)
	if code, _, diagnostic := f.check(t); code == 0 || !strings.Contains(diagnostic, "stale reference exception") {
		t.Fatalf("changed content inherited exception: code=%d diagnostic=%s", code, diagnostic)
	}
}

func TestNativeAgentSelectionContracts(t *testing.T) {
	for _, tc := range []struct{ name, requires, skillType, skillTarget, diagnostic string }{
		{"valid", "guide", "skill", "pi", ""},
		{"undeclared", "other", "skill", "pi", "not declared in requires"},
		{"missing", "guide", "", "pi", "not a Pi-compatible skill"},
		{"wrong-type", "guide", "command", "pi", "not a Pi-compatible skill"},
		{"wrong-target", "guide", "skill", "claude", "not a Pi-compatible skill"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &catalog{items: map[string]*item{
				"reader": {manifest: document{"family": "artifact", "type": "agent", "name": "reader", "targets": []any{"pi"}, "entry": "agent.md", "requires": []any{tc.requires}}, payload: map[string][]byte{"agent.md": []byte("---\nname: reader\nskills: guide\n---\nInert agent.\n")}},
			}}
			if tc.skillType != "" {
				c.items["guide"] = &item{manifest: document{"family": "artifact", "type": tc.skillType, "targets": []any{tc.skillTarget}}}
			}
			err := c.checkAgentSelections()
			if tc.diagnostic == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
				t.Fatalf("err=%v want=%s", err, tc.diagnostic)
			}
		})
	}
}
