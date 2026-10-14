package scan

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeResource(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// toyName is an invented naming rule: the frontmatter-free body's first line
// is the name; an empty body is ignored; "bad" is malformed.
func toyName(raw []byte, path string) (string, error) {
	line, _, _ := strings.Cut(strings.TrimSpace(string(raw)), "\n")
	if line == "bad" {
		return "", os.ErrInvalid
	}
	return line, nil
}

func toyRoot(path string, policy DuplicatePolicy) SkillRoot {
	return SkillRoot{Path: path, Origin: "toy " + filepath.Base(path), Recursive: true, DuplicatePolicy: policy, Name: toyName}
}

func resourceNames(rs []Resource) string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Name+"@"+filepath.Base(filepath.Dir(r.Path)))
	}
	return strings.Join(out, ",")
}

func TestDiscoverSkillsCodexRules(t *testing.T) {
	root := t.TempDir()
	writeResource(t, filepath.Join(root, "alpha/SKILL.md"), "alpha\n")
	writeResource(t, filepath.Join(root, "alpha/nested/SKILL.md"), "nested\n")
	writeResource(t, filepath.Join(root, ".hidden/SKILL.md"), "hidden\n")
	writeResource(t, filepath.Join(root, "node_modules/dep/SKILL.md"), "dep\n")
	writeResource(t, filepath.Join(root, "alpha/notes.md"), "loose\n")
	writeResource(t, filepath.Join(root, ".gitignore"), "ignored\n")
	got, err := DiscoverSkills([]SkillRoot{toyRoot(root, RefuseDifferentBytesVerifiedException)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Recursive: nested SKILL.md, dot directories and node_modules are all
	// walked; loose markdown and ignore files are not consulted.
	if want := "hidden@.hidden,alpha@alpha,nested@nested,dep@dep"; resourceNames(got) != want {
		t.Fatalf("recursive discovery = %s want %s", resourceNames(got), want)
	}
	for _, r := range got {
		if r.Kind != "skill" || r.Origin != "toy "+filepath.Base(root) || len(r.Content) == 0 {
			t.Fatalf("resource fields: %+v", r)
		}
	}
}

func TestDiscoverSkillsClaudeNeutralReferenceProof(t *testing.T) {
	for _, name := range []string{"rust-style-reference", "decision-clarity-reference"} {
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			var roots []SkillRoot
			for _, lane := range []string{"claude", "opencode", "pi"} {
				root := filepath.Join(base, lane)
				writeResource(t, filepath.Join(root, name, "SKILL.md"), name+"\nidentical\n")
				roots = append(roots, toyRoot(root, RefuseDifferentBytesVerifiedException))
			}
			if _, err := DiscoverSkills(roots[:2], func(Resource) bool { return true }); err == nil || !strings.Contains(err.Error(), "complete payload required") {
				t.Fatalf("single-file verification bypassed neutral proof: %v", err)
			}
			proof := func(a, b Resource) bool {
				return !strings.Contains(a.Path, "/pi/") && !strings.Contains(b.Path, "/pi/")
			}
			found, err := DiscoverSkills(roots[:2], nil, proof)
			if err != nil || len(found) != 1 {
				t.Fatalf("proven logical reference: %+v %v", found, err)
			}
			if _, err := DiscoverSkills(roots, nil, proof); err == nil || !strings.Contains(err.Error(), "/pi/") {
				t.Fatalf("third unverified copy disappeared: %v", err)
			}
			if _, err := DiscoverSkills(roots, nil, proof); err == nil || !strings.Contains(err.Error(), "/claude/") || !strings.Contains(err.Error(), "/opencode/") || !strings.Contains(err.Error(), "/pi/") {
				t.Fatalf("duplicate diagnostics lost a known copy: %v", err)
			}
			roots[1].Legacy = true
			if _, err := DiscoverSkills(roots[:2], nil, func(Resource, Resource) bool { return true }); err == nil {
				t.Fatal("legacy copy benefited from neutral exception")
			}
		})
	}
	base := t.TempDir()
	var roots []SkillRoot
	for _, lane := range []string{"codex", "pi"} {
		root := filepath.Join(base, lane)
		writeResource(t, filepath.Join(root, "workflow/SKILL.md"), "workflow\n")
		roots = append(roots, toyRoot(root, RefuseAlways))
	}
	if _, err := DiscoverSkills(roots, nil, func(Resource, Resource) bool { return true }); err == nil {
		t.Fatal("workflow inherited neutral exception")
	}
}

func TestDiscoverSkillsPiRules(t *testing.T) {
	root := t.TempDir()
	writeResource(t, filepath.Join(root, "alpha/SKILL.md"), "alpha\n")
	writeResource(t, filepath.Join(root, "alpha/nested/SKILL.md"), "nested\n")
	writeResource(t, filepath.Join(root, ".hidden/SKILL.md"), "hidden\n")
	writeResource(t, filepath.Join(root, "node_modules/dep/SKILL.md"), "dep\n")
	writeResource(t, filepath.Join(root, "group/beta/SKILL.md"), "beta\n")
	writeResource(t, filepath.Join(root, "loose.md"), "loose\n")
	writeResource(t, filepath.Join(root, "group/deep.md"), "deep\n")
	writeResource(t, filepath.Join(root, "empty/SKILL.md"), "\n")
	pi := SkillRoot{Path: root, Origin: "toy", SkipDotDirs: true, IgnoreRules: true, LooseMarkdown: true, DuplicatePolicy: RefuseAlways, Name: toyName}
	got, err := DiscoverSkills([]SkillRoot{pi}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Leaf directories: a directory holding SKILL.md is one resource and is not
	// entered; dot directories and node_modules are skipped; directories without
	// SKILL.md are entered; a loose .md at the root counts, deeper ones do not
	// (AgentsSkills false); an ignored (empty-name) file is silent.
	if want := "alpha@alpha,beta@beta,loose@" + filepath.Base(root); resourceNames(got) != want {
		t.Fatalf("leaf discovery = %s want %s", resourceNames(got), want)
	}
	agents := pi
	agents.AgentsSkills = true
	got, err = DiscoverSkills([]SkillRoot{agents}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := "alpha@alpha,beta@beta,deep@group"; resourceNames(got) != want {
		t.Fatalf("agents-skills discovery = %s want %s", resourceNames(got), want)
	}
	writeResource(t, filepath.Join(root, "group/.ignore"), "x\n")
	if _, err := DiscoverSkills([]SkillRoot{pi}, nil); err == nil || !strings.Contains(err.Error(), "ignore rules require qualified discovery") {
		t.Fatalf("ignore rules not enforced: %v", err)
	}
}

func TestDiscoverSkillsNameErrorsAndSafePath(t *testing.T) {
	root := t.TempDir()
	writeResource(t, filepath.Join(root, "alpha/SKILL.md"), "bad\n")
	if _, err := DiscoverSkills([]SkillRoot{toyRoot(root, RefuseAlways)}, nil); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("name error not returned as-is: %v", err)
	}
	called := ""
	r := toyRoot(root, RefuseAlways)
	r.SafePath = func(path string) error { called = path; return os.ErrPermission }
	if _, err := DiscoverSkills([]SkillRoot{r}, nil); !errors.Is(err, os.ErrPermission) || called != root {
		t.Fatalf("root safe-path hook not honoured: %v %q", err, called)
	}
	if got, err := DiscoverSkills([]SkillRoot{toyRoot(filepath.Join(root, "absent"), RefuseAlways)}, nil); err != nil || len(got) != 0 {
		t.Fatalf("missing root must yield nothing: %v %+v", err, got)
	}
}

func TestDiscoverSkillsDuplicatePolicy(t *testing.T) {
	message := func(name string, a, b Resource) error {
		return &conflictError{name: name, a: a.Path, b: b.Path}
	}
	for _, tc := range []struct {
		name     string
		policy   DuplicatePolicy
		second   string
		verified func(Resource) bool
		wantErr  bool
	}{
		{"refuse-always refuses identical bytes", RefuseAlways, "dup\n", nil, true},
		{"refuse-always ignores verification", RefuseAlways, "dup\n", func(Resource) bool { return true }, true},
		{"verified-exception admits identical bytes", RefuseDifferentBytesVerifiedException, "dup\n", nil, false},
		{"verified-exception refuses different bytes", RefuseDifferentBytesVerifiedException, "dup\nmore\n", nil, true},
		{"verified-exception refuses when one side verified", RefuseDifferentBytesVerifiedException, "dup\nmore\n", func(r Resource) bool { return strings.Contains(r.Path, "/a/") }, true},
		{"verified-exception admits when both verified", RefuseDifferentBytesVerifiedException, "dup\nmore\n", func(Resource) bool { return true }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			a, b := filepath.Join(base, "a"), filepath.Join(base, "b")
			writeResource(t, filepath.Join(a, "dup/SKILL.md"), "dup\n")
			writeResource(t, filepath.Join(b, "dup/SKILL.md"), tc.second)
			ra, rb := toyRoot(a, tc.policy), toyRoot(b, tc.policy)
			ra.ConflictMessage, rb.ConflictMessage = message, message
			got, err := DiscoverSkills([]SkillRoot{ra, rb}, tc.verified)
			if tc.wantErr {
				var ce *conflictError
				if !errors.As(err, &ce) || ce.name != "dup" || ce.a != filepath.Join(a, "dup/SKILL.md") || ce.b != filepath.Join(b, "dup/SKILL.md") {
					t.Fatalf("conflict = %v", err)
				}
				return
			}
			if err != nil || len(got) != 2 {
				t.Fatalf("admitted pair: %v %+v", err, got)
			}
		})
	}
}

// A legacy root never benefits from the verified-pair exception; the generic
// message is used when a root declares none.
func TestDiscoverSkillsLegacyAndGenericMessage(t *testing.T) {
	base := t.TempDir()
	a, b := filepath.Join(base, "a"), filepath.Join(base, "b")
	writeResource(t, filepath.Join(a, "dup/SKILL.md"), "dup\n")
	writeResource(t, filepath.Join(b, "dup/SKILL.md"), "dup\nmore\n")
	ra, rb := toyRoot(a, RefuseDifferentBytesVerifiedException), toyRoot(b, RefuseDifferentBytesVerifiedException)
	ra.Legacy = true
	_, err := DiscoverSkills([]SkillRoot{ra, rb}, func(Resource) bool { return true })
	want := "skill \"dup\" conflicts at " + filepath.Join(a, "dup/SKILL.md") + " and " + filepath.Join(b, "dup/SKILL.md") + "; incompatible same-name resources cannot be shadowed"
	if err == nil || err.Error() != want {
		t.Fatalf("legacy conflict\n got: %v\nwant: %s", err, want)
	}
}

// Roots from two targets: the selected (later) root's policy decides.
func TestDiscoverSkillsCrossRootConflict(t *testing.T) {
	base := t.TempDir()
	a, b := filepath.Join(base, "a"), filepath.Join(base, "b")
	writeResource(t, filepath.Join(a, "dup/SKILL.md"), "dup\n")
	writeResource(t, filepath.Join(b, "dup/SKILL.md"), "dup\n")
	strict := toyRoot(a, RefuseAlways)
	lenient := toyRoot(b, RefuseDifferentBytesVerifiedException)
	if _, err := DiscoverSkills([]SkillRoot{strict, lenient}, nil); err != nil {
		t.Fatalf("lenient selected root refused identical bytes: %v", err)
	}
	if _, err := DiscoverSkills([]SkillRoot{lenient, strict}, nil); err == nil {
		t.Fatal("strict selected root admitted a same-name resource")
	}
	// ResourceConflict over an already discovered inventory honours the
	// selected resource's policy and aliases.
	inventory := []Resource{{Kind: "skill", Name: "dup", Path: filepath.Join(a, "dup/SKILL.md"), Content: []byte("dup\n")}}
	selected := Resource{Kind: "skill", Name: "other", Aliases: []string{"dup"}, Path: "/selected/SKILL.md", Content: []byte("dup\n")}
	if err := ResourceConflict(inventory, selected, RefuseAlways, nil, nil); err == nil {
		t.Fatal("alias conflict not refused under RefuseAlways")
	}
	if err := ResourceConflict(inventory, selected, RefuseDifferentBytesVerifiedException, nil, nil); err != nil {
		t.Fatalf("alias with identical bytes refused under the verified-exception policy: %v", err)
	}
	foreign := Resource{Kind: "agent", Name: "dup", Path: "/agent.md"}
	if err := ResourceConflict(inventory, foreign, RefuseAlways, nil, nil); err != nil {
		t.Fatalf("conflicts are kind-scoped: %v", err)
	}
	if !bytes.Equal(inventory[0].Content, []byte("dup\n")) {
		t.Fatal("inventory mutated")
	}
}

type conflictError struct{ name, a, b string }

func (e *conflictError) Error() string { return e.name + ": " + e.a + " vs " + e.b }

// Pi agent and command discovery is not routed through the shared skill walk:
// an agent root still yields its native aliases, and a skill root yields the
// same resources through both spellings.
func TestDiscoverPiResourcesAgentsKeepAliasesAndSkillsMatchShared(t *testing.T) {
	base := t.TempDir()
	agents, skills := filepath.Join(base, "agents"), filepath.Join(base, "skills")
	writeResource(t, filepath.Join(agents, "fixture-agent.md"), "---\nname: fixture-agent\ndescription: invented\naliases: fa, fixture\n---\nBody\n")
	writeResource(t, filepath.Join(skills, "fixture-skill/SKILL.md"), "---\nname: fixture-skill\ndescription: invented\n---\nBody\n")
	writeResource(t, filepath.Join(skills, "silent/SKILL.md"), "---\nname: silent\n---\nno description\n")
	got, err := DiscoverPiResources([]PiResourceRoot{{Kind: "agent", Path: agents, Origin: "user"}, {Kind: "skill", Path: skills, Origin: "native user"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Kind != "agent" || strings.Join(got[0].Aliases, ",") != "fa,fixture" || got[1].Kind != "skill" || got[1].Name != "fixture-skill" || got[1].Origin != "native user" {
		t.Fatalf("resources = %+v", got)
	}
	shared, err := DiscoverSkills([]SkillRoot{piSkillRoot(PiResourceRoot{Kind: "skill", Path: skills, Origin: "native user"})}, nil)
	if err != nil || len(shared) != 1 || !reflect.DeepEqual(piResourceFromShared(shared[0]), got[1]) {
		t.Fatalf("shared walk differs: %+v %v", shared, err)
	}
}

// HS-IR-01: under the verified-exception policy a duplicate found early in a
// root is reported before a later malformed skill in the same root; under
// RefuseAlways the walk error keeps precedence (historical Pi order).
func TestDiscoverSkillsInlineConflictPrecedesLaterWalkError(t *testing.T) {
	base := t.TempDir()
	a, b := filepath.Join(base, "a"), filepath.Join(base, "b")
	writeResource(t, filepath.Join(a, "dup/SKILL.md"), "dup\n")
	writeResource(t, filepath.Join(b, "dup/SKILL.md"), "dup\nmore\n")
	writeResource(t, filepath.Join(b, "zzz/SKILL.md"), "bad\n")
	_, err := DiscoverSkills([]SkillRoot{toyRoot(a, RefuseDifferentBytesVerifiedException), toyRoot(b, RefuseDifferentBytesVerifiedException)}, nil)
	if err == nil || !strings.Contains(err.Error(), "conflicts at") {
		t.Fatalf("inline conflict lost to a later walk error: %v", err)
	}
	_, err = DiscoverSkills([]SkillRoot{toyRoot(a, RefuseAlways), toyRoot(b, RefuseAlways)}, nil)
	if !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("deferred policy let a conflict precede the walk error: %v", err)
	}
}

// HS-FR-02: duplicates within one root are judged inline under the
// verified-exception policy: differing bytes refused (also before a later
// malformed skill), identical bytes admitted, verified pair admitted.
func TestDiscoverSkillsSameRootDuplicates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		second   string
		later    string
		verified func(Resource) bool
		wantErr  string
	}{
		{"different bytes refused", "dup\nmore\n", "", nil, "conflicts at"},
		{"different bytes refused before later fault", "dup\nmore\n", "bad\n", nil, "conflicts at"},
		{"identical bytes admitted", "dup\n", "", nil, ""},
		{"verified pair admitted", "dup\nmore\n", "", func(Resource) bool { return true }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeResource(t, filepath.Join(root, "a/dup/SKILL.md"), "dup\n")
			writeResource(t, filepath.Join(root, "b/dup/SKILL.md"), tc.second)
			if tc.later != "" {
				writeResource(t, filepath.Join(root, "z/late/SKILL.md"), tc.later)
			}
			got, err := DiscoverSkills([]SkillRoot{toyRoot(root, RefuseDifferentBytesVerifiedException)}, tc.verified)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("same-root duplicate: %v", err)
				}
				return
			}
			if err != nil || len(got) != 2 {
				t.Fatalf("admitted pair: %v %+v", err, got)
			}
		})
	}
}
