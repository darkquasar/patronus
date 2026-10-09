package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidPackagingAndExplicitCases(t *testing.T) {
	f := newFixture(t)
	code, out, diagnostic := f.check(t)
	if code != 0 || !strings.Contains(out, "NOT CHECKED") {
		t.Fatalf("valid packaging: code=%d out=%s diagnostic=%s", code, out, diagnostic)
	}
	code, out, diagnostic = f.check(t, "--profile-cases")
	if code != 0 || strings.TrimSpace(out) != `[{"profile":"invented","target":"toy"}]` {
		t.Fatalf("cases: code=%d out=%s diagnostic=%s", code, out, diagnostic)
	}
	// Declared directory sidecars may use the public trailing-slash spelling.
	f.manifest["files"] = []string{"references/"}
	delete(f.members, "notes.txt")
	f.members["references/note.md"] = "Inert nested sidecar.\n"
	writeManifest(t, f)
	writeFixtureFile(t, filepath.Join(f.root, "artifacts/skills/sample/references/note.md"), []byte(f.members["references/note.md"]))
	f.publish(t, nil)
	code, _, diagnostic = f.check(t)
	if code != 0 {
		t.Fatal(diagnostic)
	}
	// Inline profiles support the public scalar-or-sequence representation.
	writeFixtureFile(t, filepath.Join(f.root, "profiles/invented.yaml"), []byte("apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: invented\nversion: 1.0.0\nlayers:\n  capabilities: sample\n"))
	code, _, diagnostic = f.check(t)
	if code != 0 {
		t.Fatal(diagnostic)
	}
}
func TestRejectsInvalidPackagingBatch(t *testing.T) {
	cases := []struct {
		name, want string
		mutate     func(*testing.T, *fixture)
	}{
		{"empty-source", "empty source", func(t *testing.T, f *fixture) { f.root = t.TempDir() }},
		{"unsafe-directory", "unsafe path", func(t *testing.T, f *fixture) {
			f.manifest["files"] = []string{"../references/"}
			writeManifest(t, f)
			f.publish(t, nil)
		}},
		{"missing-index-sidecar", "index sidecar", func(t *testing.T, f *fixture) {
			if err := os.Remove(f.index + ".sha256"); err != nil {
				t.Fatal(err)
			}
		}},
		{"tampered-index", "index digest", func(t *testing.T, f *fixture) { writeFixtureFile(t, f.index, []byte("{}")) }},
		{"tampered-tarball", "SHA-256 mismatch", func(t *testing.T, f *fixture) {
			writeFixtureFile(t, filepath.Join(filepath.Dir(f.index), "sample/1.0.0/sample-1.0.0.tar.gz"), []byte("tampered"))
		}},
		{"missing-sidecar", "notes.txt", func(t *testing.T, f *fixture) {
			if err := os.Remove(filepath.Join(f.root, "artifacts/skills/sample/notes.txt")); err != nil {
				t.Fatal(err)
			}
		}},
		{"symlink-sidecar", "symlink", func(t *testing.T, f *fixture) {
			p := filepath.Join(f.root, "artifacts/skills/sample/notes.txt")
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(f.index, p); err != nil {
				t.Fatal(err)
			}
		}},
		{"wrong-frontmatter", "frontmatter name", func(t *testing.T, f *fixture) {
			writeFixtureFile(t, filepath.Join(f.root, "artifacts/skills/sample/SKILL.md"), []byte("---\nname: wrong\n---\n"))
		}},
		{"missing-reference", "missing declared reference", func(t *testing.T, f *fixture) {
			f.manifest["requires"] = []string{"absent"}
			writeManifest(t, f)
			f.publish(t, nil)
		}},
		{"unknown-reference-target", "unknown reference target", func(t *testing.T, f *fixture) {
			writeFixtureFile(t, filepath.Join(f.root, "profiles/invented.yaml"), []byte("apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: invented\nversion: 1.0.0\nlayers:\n  capabilities: sample@absent\n"))
		}},
		{"manifest-mismatch", "source/index manifest mismatch", func(t *testing.T, f *fixture) { f.manifest["description"] = "changed"; f.publish(t, nil) }},
		{"archive-manifest-mismatch", "archive/index manifest mismatch", func(t *testing.T, f *fixture) {
			f.members["patronus.yaml"] = `{"family":"artifact","name":"wrong"}`
			f.publish(t, nil)
		}},
		{"unsafe-entry", "unsafe path", func(t *testing.T, f *fixture) {
			f.manifest["entry"] = "../escape"
			writeManifest(t, f)
			f.publish(t, nil)
		}},
		{"missing-source-item", "index/source mismatch", func(t *testing.T, f *fixture) {
			if err := os.RemoveAll(filepath.Join(f.root, "artifacts")); err != nil {
				t.Fatal(err)
			}
			writeFixtureFile(t, filepath.Join(f.root, "profiles/invented.yaml"), []byte("apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: invented\nversion: 1.0.0\nlayers: {}\n"))
			rewriteIndex(t, f, func(ix map[string]any) {
				entries := ix["profiles"].([]any)
				entries[0].(map[string]any)["manifest"].(map[string]any)["layers"] = map[string]any{}
			})
		}},
		{"empty-index", "empty index", func(t *testing.T, f *fixture) {
			rewriteIndex(t, f, func(ix map[string]any) { ix["artifacts"] = []any{}; ix["profiles"] = []any{} })
		}},
		{"missing-index-item", "set mismatch", func(t *testing.T, f *fixture) {
			rewriteIndex(t, f, func(ix map[string]any) { ix["profiles"] = []any{} })
		}},
		{"duplicate-index-item", "duplicate", func(t *testing.T, f *fixture) {
			rewriteIndex(t, f, func(ix map[string]any) { a := ix["artifacts"].([]any); ix["artifacts"] = append(a, a[0]) })
		}},
		{"unsupported-schema", "schemaVersion", func(t *testing.T, f *fixture) {
			rewriteIndex(t, f, func(ix map[string]any) { ix["schemaVersion"] = 9 })
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.mutate(t, f)
			code, _, diagnostic := f.check(t)
			if code == 0 || !strings.Contains(diagnostic, tc.want) {
				t.Fatalf("code=%d diagnostic=%q want=%q", code, diagnostic, tc.want)
			}
		})
	}
}
func writeManifest(t *testing.T, f *fixture) {
	t.Helper()
	b, err := json.Marshal(f.manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(f.root, "artifacts/skills/sample/patronus.yaml"), b)
}
func rewriteIndex(t *testing.T, f *fixture, change func(map[string]any)) {
	t.Helper()
	b, err := os.ReadFile(f.index)
	if err != nil {
		t.Fatal(err)
	}
	var ix map[string]any
	if err := json.Unmarshal(b, &ix); err != nil {
		t.Fatal(err)
	}
	change(ix)
	b, err = json.Marshal(ix)
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, f.index, b)
	writeFixtureFile(t, f.index+".sha256", []byte(fmt.Sprintf("sha256:%x\n", sha256.Sum256(b))))
}
func TestProfileCasesRejectUnresolvedMetadata(t *testing.T) {
	for _, body := range []string{"schema_version: 1\nprofiles: {}", "schema_version: 1\nprofiles:\n  invented: {targets: []}", "schema_version: 1\nprofiles:\n  invented: {targets: [absent]}", "schema_version: 1\nprofiles:\n  invented: {targets: [toy]}\n  absent: {targets: [toy]}"} {
		t.Run(body, func(t *testing.T) {
			f := newFixture(t)
			writeFixtureFile(t, filepath.Join(f.root, "docs/compatibility/profile-targets.yaml"), []byte(body))
			code, _, diagnostic := f.check(t, "--profile-cases")
			if code == 0 || !strings.Contains(diagnostic, "profile cases") {
				t.Fatalf("code=%d diagnostic=%s", code, diagnostic)
			}
		})
	}
}

const _validLedger = "schema_version: 1\nprofile: invented\nbaseline_profile: invented\nbaseline_target: toy\ntarget: toy\nitem_suffix: tz\nentries:\n  - core_item: sample\n    disposition: port-tz\n    toy_items: [sample]\n    outcome: Partial\n    reason: Invented structural evidence only\n    evidence: runtime-pending\n"
const _validLock = "version: 3\nprofile: invented\ntarget: toy\nentries:\n  - name: sample\n    kind: artifact\n    version: 1.0.0\n"

func TestLedgerConsumesPublicClosures(t *testing.T) {
	f := newFixture(t)
	writeFixtureFile(t, filepath.Join(f.root, "docs/compatibility/invented.yaml"), []byte(_validLedger))
	dir := filepath.Join(f.root, "closures")
	writeFixtureFile(t, filepath.Join(dir, "invented--toy.lock"), []byte(_validLock))
	code, out, diagnostic := f.check(t, "--closures", dir)
	if code != 0 || !strings.Contains(out, "checked from supplied") {
		t.Fatalf("code=%d out=%s diagnostic=%s", code, out, diagnostic)
	}
	for _, bad := range []string{strings.ReplaceAll(_validLock, "profile: invented", "profile: wrong"), strings.ReplaceAll(_validLock, "version: 1.0.0", "version: 2.0.0"), "version: 3\nprofile: invented\ntarget: toy\nentries: []\n"} {
		writeFixtureFile(t, filepath.Join(dir, "invented--toy.lock"), []byte(bad))
		code, _, diagnostic = f.check(t, "--closures", dir)
		if code == 0 || !strings.Contains(diagnostic, "closure") {
			t.Fatalf("bad public closure accepted: %s", diagnostic)
		}
	}
}
func TestLedgerRejectsMissingMappingAndUnsupportedLabels(t *testing.T) {
	for _, bad := range []string{strings.ReplaceAll(_validLedger, "core_item: sample", "core_item: absent"), strings.ReplaceAll(_validLedger, "outcome: Partial", "outcome: Guaranteed"), strings.ReplaceAll(_validLedger, "evidence: runtime-pending", "evidence: assumed"), strings.ReplaceAll(_validLedger, "toy_items: [sample]", "toy_items: [absent]")} {
		f := newFixture(t)
		writeFixtureFile(t, filepath.Join(f.root, "docs/compatibility/invented.yaml"), []byte(bad))
		code, _, diagnostic := f.check(t)
		if code == 0 || !strings.Contains(diagnostic, "invented.yaml") {
			t.Fatalf("bad ledger accepted: %s", diagnostic)
		}
	}
}
func TestLedgerTargetVocabularyIsDataDriven(t *testing.T) {
	cases := []struct {
		name, ledger, want string
	}{
		{"missing-target", strings.ReplaceAll(_validLedger, "\ntarget: toy\n", "\n"), "ledger requires target: and item_suffix:"},
		{"missing-item-suffix", strings.ReplaceAll(_validLedger, "item_suffix: tz\n", ""), "ledger requires target: and item_suffix:"},
		{"unknown-target", strings.ReplaceAll(_validLedger, "\ntarget: toy\n", "\ntarget: absent\n"), "unknown ledger target absent"},
		{"all-is-not-a-target", strings.ReplaceAll(_validLedger, "\ntarget: toy\n", "\ntarget: all\n"), "unknown ledger target all"},
		{"unsafe-item-suffix", strings.ReplaceAll(_validLedger, "item_suffix: tz\n", "item_suffix: T-Z\n"), "unsafe item_suffix"},
		{"foreign-port-disposition", strings.ReplaceAll(_validLedger, "disposition: port-tz", "disposition: port-cx"), "unsupported disposition"},
		{"foreign-items-key", strings.ReplaceAll(_validLedger, "toy_items: [sample]", "codex_items: [sample]"), "missing endpoints"},
		{"additional-unknown-item", _validLedger + "additional_items:\n  - items: [absent]\n    outcome: Partial\n    reason: invented\n    evidence: not-run\n", "additional_items[0]: missing endpoint absent"},
		{"additional-missing-reason", _validLedger + "additional_items:\n  - items: [sample]\n    outcome: Partial\n    evidence: not-run\n", "additional_items[0]: missing reason"},
		{"additional-equivalent-without-items", _validLedger + "additional_items:\n  - outcome: Equivalent\n    reason: invented\n    evidence: not-run\n", "additional_items[0]: Equivalent requires items"},
		{"additional-bad-outcome", _validLedger + "additional_items:\n  - items: [sample]\n    outcome: Guaranteed\n    reason: invented\n    evidence: not-run\n", "additional_items[0]: unsupported outcome"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			writeFixtureFile(t, filepath.Join(f.root, "docs/compatibility/invented.yaml"), []byte(tc.ledger))
			code, _, diagnostic := f.check(t)
			if code == 0 || !strings.Contains(diagnostic, tc.want) {
				t.Fatalf("code=%d diagnostic=%q want=%q", code, diagnostic, tc.want)
			}
		})
	}
}
func TestLedgerAdditionalItemsCountAsClosureEndpoints(t *testing.T) {
	f := newFixture(t)
	// The baseline member is omitted by its entry, so sample reaches the mapped
	// endpoints only through additional_items: the additional-only mechanism is
	// what the closure coverage below depends on.
	omitted := strings.Replace(strings.Replace(_validLedger, "disposition: port-tz", "disposition: omit", 1), "    toy_items: [sample]\n", "", 1)
	writeFixtureFile(t, filepath.Join(f.root, "docs/compatibility/invented.yaml"), []byte(omitted+"additional_items:\n  - items: [sample]\n    outcome: Partial\n    reason: invented lane item\n    evidence: not-run\n"))
	dir := filepath.Join(f.root, "closures")
	writeFixtureFile(t, filepath.Join(dir, "invented--toy.lock"), []byte(_validLock))
	code, out, diagnostic := f.check(t, "--closures", dir)
	if code != 0 || !strings.Contains(out, "checked from supplied") {
		t.Fatalf("code=%d out=%s diagnostic=%s", code, out, diagnostic)
	}
	// Without the additional row the omitted member has no ledger evidence at all.
	writeFixtureFile(t, filepath.Join(f.root, "docs/compatibility/invented.yaml"), []byte(omitted))
	code, _, diagnostic = f.check(t, "--closures", dir)
	if code == 0 || !strings.Contains(diagnostic, "lacks ledger evidence") {
		t.Fatalf("omitted member without additional evidence accepted: %s", diagnostic)
	}
}

// HS-IR-05 / HS-FR-05: closure coverage binds the ledger's declared
// profile/target case. With a profile admitted for two targets, the toy ledger
// requires its endpoint in the toy closure only: an other-target closure that
// lacks it is not judged, and a toy closure that lacks it fails even though
// the other closure holds it.
func TestLedgerClosureBindsDeclaredTarget(t *testing.T) {
	f := newFixture(t)
	f.publishSecond(t, "extra")
	writeFixtureFile(t, filepath.Join(f.root, "adapters/other.yaml"), []byte("tool: other\nfamily: adapter\n"))
	writeFixtureFile(t, filepath.Join(f.root, "docs/compatibility/profile-targets.yaml"), []byte("schema_version: 1\nprofiles:\n  invented:\n    targets: [toy, other]\n"))
	writeFixtureFile(t, filepath.Join(f.root, "docs/compatibility/invented.yaml"), []byte(_validLedger))
	dir := filepath.Join(f.root, "closures")
	withSample := _validLock
	onlyExtra := "version: 3\nprofile: invented\ntarget: %s\nentries:\n  - name: extra\n    kind: artifact\n    version: 1.0.0\n"
	writeFixtureFile(t, filepath.Join(dir, "invented--toy.lock"), []byte(withSample))
	writeFixtureFile(t, filepath.Join(dir, "invented--other.lock"), []byte(fmt.Sprintf(onlyExtra, "other")))
	if code, out, diagnostic := f.check(t, "--closures", dir); code != 0 || !strings.Contains(out, "checked from supplied") {
		t.Fatalf("other-target closure without the endpoint was judged: code=%d %s %s", code, out, diagnostic)
	}
	writeFixtureFile(t, filepath.Join(dir, "invented--toy.lock"), []byte(fmt.Sprintf(onlyExtra, "toy")))
	writeFixtureFile(t, filepath.Join(dir, "invented--other.lock"), []byte(strings.ReplaceAll(withSample, "target: toy", "target: other")))
	code, _, diagnostic := f.check(t, "--closures", dir)
	if code == 0 || !strings.Contains(diagnostic, "invented--toy closure missing ledger endpoint sample") {
		t.Fatalf("declared-target closure missing the endpoint accepted: code=%d %s", code, diagnostic)
	}
	// The ledger's own target field (not baseline_target) selects the closure.
	otherLedger := strings.Replace(_validLedger, "\ntarget: toy\n", "\ntarget: other\n", 1)
	otherLedger = strings.ReplaceAll(otherLedger, "toy_items:", "other_items:")
	writeFixtureFile(t, filepath.Join(f.root, "docs/compatibility/invented.yaml"), []byte(otherLedger))
	if code, out, diagnostic := f.check(t, "--closures", dir); code != 0 || !strings.Contains(out, "checked from supplied") {
		t.Fatalf("other-target ledger not judged against the other closure: code=%d %s %s", code, out, diagnostic)
	}
}

func TestProfileCasesAgreeWithDeclaredManifestTargets(t *testing.T) {
	const declaring = "apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: invented\nversion: 1.0.0\ntargets: [%s]\nlayers:\n  capabilities: [sample]\n"
	const extending = "apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: child\nversion: 1.0.0\nextends: invented\nlayers:\n  capabilities: [sample]\n"
	cases := []struct {
		name, profile, metadata string
		wantCode                int
		want                    string
	}{
		{"declared-agrees", fmt.Sprintf(declaring, "toy"), "schema_version: 1\nprofiles:\n  invented: {targets: [toy]}\n  child: {targets: [toy]}\n", 0, `[{"profile":"child","target":"toy"},{"profile":"invented","target":"toy"}]`},
		{"declared-mismatch", fmt.Sprintf(declaring, "other"), "schema_version: 1\nprofiles:\n  invented: {targets: [toy]}\n  child: {targets: [toy]}\n", 1, "profile cases: invented: manifest declares targets [other] but profile-targets.yaml admits [toy]"},
		{"extends-without-own-targets-is-not-declaring", fmt.Sprintf(declaring, "toy"), "schema_version: 1\nprofiles:\n  invented: {targets: [toy]}\n  child: {targets: [other]}\n", 0, `[{"profile":"child","target":"other"},{"profile":"invented","target":"toy"}]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			writeFixtureFile(t, filepath.Join(f.root, "adapters/other.yaml"), []byte("tool: other\nfamily: adapter\n"))
			writeFixtureFile(t, filepath.Join(f.root, "profiles/invented.yaml"), []byte(tc.profile))
			writeFixtureFile(t, filepath.Join(f.root, "profiles/child.yaml"), []byte(extending))
			writeFixtureFile(t, filepath.Join(f.root, "docs/compatibility/profile-targets.yaml"), []byte(tc.metadata))
			rewriteIndex(t, f, func(ix map[string]any) {
				ix["profiles"] = []any{
					map[string]any{"manifest": profileDocument(t, tc.profile)},
					map[string]any{"manifest": profileDocument(t, extending)},
				}
			})
			code, out, diagnostic := f.check(t, "--profile-cases")
			if code != tc.wantCode || !strings.Contains(strings.TrimSpace(out)+diagnostic, tc.want) {
				t.Fatalf("code=%d out=%s diagnostic=%q want=%q", code, out, diagnostic, tc.want)
			}
		})
	}
}
func profileDocument(t *testing.T, body string) map[string]any {
	t.Helper()
	m, err := decodeYAML([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any(m)
}
func TestSchema2AndSourceOnlyPlugins(t *testing.T) {
	legacy := document{"family": "recipe", "kind": "Recipe", "version": "1.0.0"}
	public := document{"family": "recipe", "version": "1.0.0", "wire": document{}}
	if !manifestEqual(legacy, public) {
		t.Fatal("public recipe serialization rejected the legacy source Kind header")
	}
	public["version"] = "2.0.0"
	if manifestEqual(legacy, public) {
		t.Fatal("normalization ignored a real version difference")
	}
	f := newFixture(t)
	rewriteIndex(t, f, func(ix map[string]any) { ix["schemaVersion"] = 2 })
	writeFixtureFile(t, filepath.Join(f.root, "plugins/extra.yaml"), []byte("apiVersion: patronus/v2\nfamily: plugin\nrole: capability\nname: extra\nversion: 1.0.0\ntargets: [toy]\nsources: {toy: {kind: local, plugin: inert}}\n"))
	code, _, diagnostic := f.check(t)
	if code != 0 {
		t.Fatal(diagnostic)
	}
}
func TestUsageRequiresExplicitLocalInputs(t *testing.T) {
	if code := run(nil, os.Stdout, os.Stderr); code != 2 {
		t.Fatalf("usage exit = %d", code)
	}
}
