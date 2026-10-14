package harness

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// recorder logs the phase hooks Admit calls, in order, for one admission.
type recorder struct {
	log  *[]string
	fail map[string]error
}

func (r recorder) note(phase string, fd diff.FileDiff) error {
	*r.log = append(*r.log, phase+":"+filepath.Base(fd.Path))
	return r.fail[phase]
}

func (r recorder) Prepare(_ Descriptor, _ *Admission) error {
	*r.log = append(*r.log, "prepare")
	return r.fail["prepare"]
}
func (r recorder) BeforeOwnership(_ Descriptor, _ *Admission, fd diff.FileDiff) error {
	return r.note("before-ownership", fd)
}
func (r recorder) AfterOwnership(_ Descriptor, _ *Admission, fd diff.FileDiff, _ *ContextFacts) error {
	return r.note("after-ownership", fd)
}
func (r recorder) AfterContext(_ Descriptor, _ *Admission, fd diff.FileDiff, _ *ContextFacts) error {
	return r.note("after-context", fd)
}

var _ Validator = recorder{}

type admitFixture struct {
	home, project string
	res           toolpath.Resolver
}

func newAdmitFixture(t *testing.T) admitFixture {
	t.Helper()
	home, project := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	return admitFixture{home: home, project: project, res: toolpath.New(os.LookupEnv, home, project)}
}

func (f admitFixture) write(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0644); err != nil {
		t.Fatal(err)
	}
}

func (f admitFixture) admission(cs *diff.ChangeSet, owners []state.Item) *Admission {
	return &Admission{Changes: cs, Owners: owners, Resolver: f.res, Home: f.home, Project: f.project, Env: os.LookupEnv}
}

// toy is strict (Codex-shaped) and runs the resource rules after context; lax
// is lenient (Pi-shaped) and runs them before ownership.
func toyDescriptor(roots RootsFunc) Descriptor {
	return Descriptor{Name: "toy", ContextFile: "TOY.md", ShadowFiles: []string{"TOY.override.md"}, GlobalMarker: "~/.toy", SkillRoots: roots, ResourcePhase: ResourceAfterContext, StrictUnowned: true, StrictSections: true, SafePathCheck: true, AdmissionPrefix: "toy admission", ErrorPrefix: "toy"}
}

func laxDescriptor(roots RootsFunc) Descriptor {
	return Descriptor{Name: "lax", ContextFile: "TOY.md", SectionPrefix: "lax:", GlobalMarker: "~/.lax", SkillRoots: roots, ResourcePhase: ResourceBeforeOwnership, SafePathCheck: true, MixedContextConsent: true, ErrorPrefix: "lax"}
}

func toyRootsFunc(home string) RootsFunc {
	return func(in RootInputs) ([]scan.SkillRoot, error) {
		return []scan.SkillRoot{{Path: filepath.Join(home, ".toy/skills"), Recursive: true, DuplicatePolicy: scan.RefuseDifferentBytesVerifiedException, Name: toyName}}, nil
	}
}

func toyName(raw []byte, path string) (string, error) {
	line, _, _ := strings.Cut(strings.TrimSpace(string(raw)), "\n")
	return line, nil
}

func TestAdmitClaudeNeutralReferenceCoexistence(t *testing.T) {
	for _, pair := range [][2]string{{"claude", "opencode"}, {"codex", "opencode"}, {"codex", "pi"}} {
		t.Run(pair[0]+"+"+pair[1], func(t *testing.T) {
			f := newAdmitFixture(t)
			name := "decision-clarity-reference"
			body := []byte("---\nname: " + name + "\ndescription: invented neutral reference\n---\nRead body.md at this exact installed path.\n")
			var roots []scan.SkillRoot
			var owners []state.Item
			var selected string
			for _, tool := range pair {
				root := filepath.Join(f.project, tool, "skills")
				path := filepath.Join(root, name, "SKILL.md")
				selected = path
				policy := scan.RefuseDifferentBytesVerifiedException
				if tool == "pi" || tool == "claude" {
					policy = scan.RefuseAlways
				}
				roots = append(roots, scan.SkillRoot{Path: root, Origin: tool, Name: scan.SkillDirName, DuplicatePolicy: policy})
				owner := state.Item{Artifact: name, ItemVersion: "1.0.0", Type: "skill", Tool: tool, Scope: "local"}
				for rel, bytes := range map[string][]byte{"SKILL.md": body, "body.md": []byte("invented shared decision guidance\n")} {
					p := filepath.Join(filepath.Dir(path), rel)
					f.write(t, p, bytes)
					owner.Files = append(owner.Files, state.FileState{Path: p, Action: "CREATE", Checksum: state.Checksum(bytes)})
				}
				owners = append(owners, owner)
			}
			d := Descriptor{Name: pair[1], GlobalMarker: "~/." + pair[1], ResourcePhase: ResourceAfterContext, SkillName: scan.SkillDirName, SkillRoots: func(RootInputs) ([]scan.SkillRoot, error) { return roots, nil }}
			fd := diff.FileDiff{Artifact: name, Version: "1.0.0", Tool: d.Name, Scope: "local", Type: "skill", Path: selected, Action: diff.Skip, Before: body, After: body}
			in := f.admission(&diff.ChangeSet{Diffs: []diff.FileDiff{fd}}, owners)
			if err := Admit(d, in); err != nil {
				t.Fatalf("identical managed references refused: %v", err)
			}
			if len(in.Inventory) != 1 {
				t.Fatalf("expected one logical reference, got %+v", in.Inventory)
			}
		})
	}
}

func TestAdmitNeutralReferenceFreshSecondHost(t *testing.T) {
	for _, pair := range [][2]string{{"claude", "opencode"}, {"codex", "opencode"}, {"codex", "pi"}} {
		for _, scope := range []string{"local", "global"} {
			t.Run(pair[0]+"+"+pair[1]+"/"+scope, func(t *testing.T) {
				f := newAdmitFixture(t)
				name := "rust-style-reference"
				entry := []byte("---\nname: " + name + "\ndescription: invented neutral reference\n---\nRead body.md.\n")
				sidecar := []byte("invented shared guidance\n")
				foreignRoot := filepath.Join(f.project, pair[0], scope, "skills")
				selectedRoot := filepath.Join(f.project, pair[1], scope, "skills")
				foreignEntry := filepath.Join(foreignRoot, name, "SKILL.md")
				foreignSidecar := filepath.Join(foreignRoot, name, "body.md")
				f.write(t, foreignEntry, entry)
				f.write(t, foreignSidecar, sidecar)
				owner := state.Item{
					Artifact: name, ItemVersion: "1.0.0", Type: "skill", Tool: pair[0], Scope: scope,
					Files: []state.FileState{
						{Path: foreignEntry, Action: "CREATE", Checksum: state.Checksum(entry)},
						{Path: foreignSidecar, Action: "CREATE", Checksum: state.Checksum(sidecar)},
					},
				}
				selectedEntry := filepath.Join(selectedRoot, name, "SKILL.md")
				selectedSidecar := filepath.Join(selectedRoot, name, "body.md")
				changes := &diff.ChangeSet{Diffs: []diff.FileDiff{
					{Artifact: name, Version: "1.0.0", Tool: pair[1], Scope: scope, Type: "skill", Path: selectedEntry, Action: diff.Create, After: entry},
					{Artifact: name, Version: "1.0.0", Tool: pair[1], Scope: scope, Type: "skill", Path: selectedSidecar, Action: diff.Create, After: sidecar},
				}}
				policy := scan.RefuseDifferentBytesVerifiedException
				if pair[1] == "pi" {
					policy = scan.RefuseAlways
				}
				d := Descriptor{
					Name: pair[1], GlobalMarker: "~/." + pair[1], ResourcePhase: ResourceAfterContext,
					SkillName: scan.SkillDirName,
					SkillRoots: func(RootInputs) ([]scan.SkillRoot, error) {
						return []scan.SkillRoot{
							{Path: selectedRoot, Origin: "selected", Name: scan.SkillDirName, DuplicatePolicy: policy},
							{Path: foreignRoot, Origin: "foreign", Name: scan.SkillDirName, DuplicatePolicy: scan.RefuseDifferentBytesVerifiedException},
						}, nil
					},
				}
				if err := Admit(d, f.admission(changes, []state.Item{owner})); err != nil {
					t.Fatalf("fresh second-host reference create refused: %v", err)
				}
			})
		}
	}
}

func TestAdmitClaudeNeutralReferenceSamePathOwners(t *testing.T) {
	f := newAdmitFixture(t)
	name := "rust-style-reference"
	root := filepath.Join(f.project, ".agents/skills")
	path := filepath.Join(root, name, "SKILL.md")
	body := []byte("---\nname: " + name + "\ndescription: invented\n---\nshared reference\n")
	f.write(t, path, body)
	owners := []state.Item{}
	for _, tool := range []string{"codex", "opencode"} {
		owners = append(owners, state.Item{Artifact: name, ItemVersion: "1.0.0", Type: "skill", Scope: "local", Tool: tool, Files: []state.FileState{{Path: path, Action: "CREATE", Checksum: state.Checksum(body)}}})
	}
	d := toyDescriptor(func(RootInputs) ([]scan.SkillRoot, error) {
		return []scan.SkillRoot{{Path: root, Name: scan.SkillDirName, DuplicatePolicy: scan.RefuseDifferentBytesVerifiedException}}, nil
	})
	d.Name = "opencode"
	fd := diff.FileDiff{Artifact: name, Version: "1.0.0", Type: "skill", Tool: d.Name, Scope: "local", Path: path, Action: diff.Skip, Before: body, After: body}
	if err := Admit(d, f.admission(&diff.ChangeSet{Diffs: []diff.FileDiff{fd}}, owners)); err != nil {
		t.Fatalf("same-path neutral owners refused: %v", err)
	}
	for _, fault := range []string{"version", "scope", "edit", "delete"} {
		t.Run(fault, func(t *testing.T) {
			other := append([]state.Item(nil), owners...)
			selected := fd
			switch fault {
			case "version":
				other[0].ItemVersion = "2.0.0"
			case "scope":
				other[0].Scope = "global"
			case "edit":
				selected.Action, selected.After = diff.Create, []byte("changed\n")
			case "delete":
				selected.Action, selected.After = diff.Delete, nil
			}
			if err := Admit(d, f.admission(&diff.ChangeSet{Diffs: []diff.FileDiff{selected}}, other)); err == nil {
				t.Fatalf("same-path %s silently admitted", fault)
			}
		})
	}
}

func neutralAdmission(t *testing.T, pair [2]string, name string) (Descriptor, *Admission, admitFixture) {
	t.Helper()
	f := newAdmitFixture(t)
	body := []byte("---\nname: " + name + "\ndescription: invented\n---\nneutral guidance\n")
	var roots []scan.SkillRoot
	var owners []state.Item
	var selected string
	for _, tool := range pair {
		root := filepath.Join(f.project, tool, "skills")
		selected = filepath.Join(root, name, "SKILL.md")
		policy := scan.RefuseDifferentBytesVerifiedException
		if tool == "pi" || tool == "claude" {
			policy = scan.RefuseAlways
		}
		roots = append(roots, scan.SkillRoot{Path: root, Origin: tool, Name: scan.SkillDirName, DuplicatePolicy: policy})
		owner := state.Item{Artifact: name, ItemVersion: "1.0.0", Type: "skill", Tool: tool, Scope: "local"}
		for rel, raw := range map[string][]byte{"SKILL.md": body, "body.md": []byte("own-lane shared guidance\n")} {
			path := filepath.Join(filepath.Dir(selected), rel)
			f.write(t, path, raw)
			owner.Files = append(owner.Files, state.FileState{Path: path, Checksum: state.Checksum(raw), Action: "CREATE"})
		}
		owners = append(owners, owner)
	}
	d := Descriptor{Name: pair[1], GlobalMarker: "~/." + pair[1], ResourcePhase: ResourceAfterContext, SkillName: scan.SkillDirName, SkillRoots: func(RootInputs) ([]scan.SkillRoot, error) { return roots, nil }}
	fd := diff.FileDiff{Artifact: name, Version: "1.0.0", Tool: d.Name, Scope: "local", Type: "skill", Path: selected, Action: diff.Skip, Before: body, After: body}
	return d, f.admission(&diff.ChangeSet{Diffs: []diff.FileDiff{fd}}, owners), f
}

func TestAdmitClaudeNeutralReferenceSelectedVersionMismatch(t *testing.T) {
	d, in, _ := neutralAdmission(t, [2]string{"claude", "opencode"}, "decision-clarity-reference")
	in.Changes.Diffs[0].Version = "2.0.0"
	if err := Admit(d, in); err == nil {
		t.Fatal("selected version mismatch was admitted")
	}
}

func TestAdmitManagedEditorialCoexistence(t *testing.T) {
	for _, pair := range [][2]string{{"claude", "opencode"}, {"codex", "opencode"}, {"codex", "pi"}} {
		t.Run(strings.Join(pair[:], "+"), func(t *testing.T) {
			d, in, _ := neutralAdmission(t, pair, "writing-editorial")
			if err := Admit(d, in); err != nil {
				t.Fatalf("identical managed editorial copies refused: %v", err)
			}
			if len(in.Inventory) != 1 {
				t.Fatalf("editorial copies did not reconcile once: %+v", in.Inventory)
			}
		})
	}
}

func TestAdmitClaudeNeutralReferencePartialCoOwner(t *testing.T) {
	d, in, f := neutralAdmission(t, [2]string{"claude", "opencode"}, "decision-clarity-reference")
	sidecar := filepath.Join(f.project, "claude/skills/decision-clarity-reference/body.md")
	in.Owners = append(in.Owners, state.Item{Artifact: "decision-clarity-reference", ItemVersion: "1.0.0", Tool: "codex", Type: "skill", Scope: "local", Files: []state.FileState{{Path: sidecar, Action: "CREATE", Checksum: state.Checksum([]byte("own-lane shared guidance\n"))}}})
	if err := Admit(d, in); err == nil {
		t.Fatal("partially owned co-owner was admitted")
	}
}

func TestAdmitClaudeNeutralReferenceUnsafeCopies(t *testing.T) {
	for _, pair := range [][2]string{{"claude", "opencode"}, {"codex", "opencode"}, {"codex", "pi"}} {
		for _, name := range []string{"decision-clarity-reference", "rust-style-reference", "writing-editorial"} {
			for _, fault := range []string{"unmanaged", "one unmanaged", "edited entry", "edited sidecar", "different managed payload", "partial ownership", "missing sidecar", "extra file", "version mismatch", "empty version", "artifact mismatch", "foreign tool", "missing skill type"} {
				t.Run(strings.Join(pair[:], "+")+"/"+name+"/"+fault, func(t *testing.T) {
					d, in, f := neutralAdmission(t, pair, name)
					foreignEntry := filepath.Join(f.project, pair[0], "skills", name, "SKILL.md")
					sidecar := filepath.Join(filepath.Dir(foreignEntry), "body.md")
					switch fault {
					case "unmanaged":
						in.Owners = nil
					case "one unmanaged":
						in.Owners = in.Owners[1:]
					case "edited entry":
						f.write(t, foreignEntry, []byte("---\nname: "+name+"\ndescription: changed\n---\nedited\n"))
					case "edited sidecar":
						f.write(t, sidecar, []byte("edited\n"))
					case "different managed payload":
						f.write(t, sidecar, []byte("different owned guidance\n"))
						for i := range in.Owners[0].Files {
							if in.Owners[0].Files[i].Path == sidecar {
								in.Owners[0].Files[i].Checksum = state.Checksum([]byte("different owned guidance\n"))
							}
						}
					case "partial ownership":
						var files []state.FileState
						for _, file := range in.Owners[0].Files {
							if file.Path != sidecar {
								files = append(files, file)
							}
						}
						in.Owners[0].Files = files
					case "missing sidecar":
						if err := os.Remove(sidecar); err != nil {
							t.Fatal(err)
						}
					case "extra file":
						f.write(t, filepath.Join(filepath.Dir(foreignEntry), "unowned.md"), []byte("unmanaged\n"))
					case "version mismatch":
						in.Owners[0].ItemVersion = "2.0.0"
					case "empty version":
						in.Owners[0].ItemVersion = ""
					case "artifact mismatch":
						in.Owners[0].Artifact = "different-reference"
					case "foreign tool":
						in.Owners[0].Tool = "external"
					case "missing skill type":
						in.Owners[0].Type = ""
					}
					err := Admit(d, in)
					if err == nil || !strings.Contains(err.Error(), foreignEntry) || !strings.Contains(err.Error(), in.Changes.Diffs[0].Path) || !strings.Contains(err.Error(), name) {
						t.Fatalf("unsafe copy %s not refused with both identities/paths: %v", fault, err)
					}
				})
			}
		}
	}
}

func TestAdmitClaudeNeutralReferenceOwnLaneRead(t *testing.T) {
	d, in, f := neutralAdmission(t, [2]string{"claude", "opencode"}, "rust-style-reference")
	in.Owners[0].ItemVersion = "2.0.0"
	foreign := filepath.Join(f.project, "claude/skills/rust-style-reference/body.md")
	local := filepath.Join(f.project, "opencode/skills/rust-style-reference/body.md")
	f.write(t, foreign, []byte("Claude-only version two guidance\n"))
	if err := Admit(d, in); err == nil {
		t.Fatal("different versions silently resolved")
	}
	raw, err := os.ReadFile(local)
	if err != nil || string(raw) != "own-lane shared guidance\n" {
		t.Fatalf("own-lane reference selection: %q %v", raw, err)
	}
	// The fixture reads an exact installed path; it makes no native-loader claim.
}

func TestAdmitSelectionAndWarning(t *testing.T) {
	f := newAdmitFixture(t)
	var warned []string
	in := f.admission(&diff.ChangeSet{Diffs: []diff.FileDiff{{Tool: "other", Path: "/x"}}}, nil)
	in.Warn = func(m string) { warned = append(warned, m) }
	d := toyDescriptor(nil)
	d.PlacementWarning = "toy placement is static"
	if err := Admit(d, in); err != nil || len(warned) != 0 {
		t.Fatalf("unselected target admitted with side effects: %v %v", err, warned)
	}
	in.Changes.Diffs = append(in.Changes.Diffs, diff.FileDiff{Tool: "toy+other", Path: filepath.Join(f.project, "a.md")})
	err := Admit(d, in)
	if err == nil || err.Error() != "toy cross-target destination conflict "+filepath.Join(f.project, "a.md")+" (toy+other)" {
		t.Fatalf("composite destination: %v", err)
	}
	in.Changes.Diffs = []diff.FileDiff{{Tool: "toy", Path: filepath.Join(f.project, "a.md"), Action: diff.Create, After: []byte("x")}}
	if err := Admit(d, in); err != nil || strings.Join(warned, ",") != "toy placement is static" {
		t.Fatalf("placement warning: %v %v", err, warned)
	}
	if err := Admit(d, nil); err != nil {
		t.Fatalf("nil admission: %v", err)
	}
}

func TestAdmitPhaseOrder(t *testing.T) {
	f := newAdmitFixture(t)
	skill := filepath.Join(f.project, ".toy/skills/alpha/SKILL.md")
	context := filepath.Join(f.project, "TOY.md")
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{
		{Tool: "toy", Type: "skill", Artifact: "alpha", Path: skill, Action: diff.Create, After: []byte("alpha\n"), Scope: "local"},
		{Tool: "toy", Type: "instruction", Artifact: "alpha", Path: context, Action: diff.Append, Section: &diff.SectionEdit{Name: "alpha"}, After: adapter.AppendSection(nil, "alpha", []byte("x")), Scope: "local"},
		{Tool: "toy", Type: "skill", Artifact: "alpha", Path: filepath.Join(f.project, ".toy/skills/alpha/run.sh"), Action: diff.Exec, Scope: "local"},
	}}
	var log []string
	d := toyDescriptor(toyRootsFunc(f.home))
	d.Validator = recorder{log: &log}
	if err := Admit(d, f.admission(cs, nil)); err != nil {
		t.Fatal(err)
	}
	want := "prepare,before-ownership:SKILL.md,after-ownership:SKILL.md,after-context:SKILL.md,before-ownership:TOY.md,after-ownership:TOY.md,after-context:TOY.md"
	if got := strings.Join(log, ","); got != want {
		t.Fatalf("hook order\n got: %s\nwant: %s", got, want)
	}
	for _, phase := range []string{"prepare", "before-ownership", "after-ownership", "after-context"} {
		d.Validator = recorder{log: &log, fail: map[string]error{phase: os.ErrInvalid}}
		if err := Admit(d, f.admission(cs, nil)); !errors.Is(err, os.ErrInvalid) {
			t.Fatalf("%s error not returned as-is: %v", phase, err)
		}
	}
}

func TestAdmitSafePaths(t *testing.T) {
	f := newAdmitFixture(t)
	real := filepath.Join(f.project, "real")
	link := filepath.Join(f.project, "link")
	if err := os.Mkdir(real, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	unsafe := filepath.Join(link, "a.md")
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Tool: "toy", Path: unsafe, Action: diff.Create, After: []byte("x")}}}
	err := Admit(toyDescriptor(nil), f.admission(cs, nil))
	if err == nil || err.Error() != "toy admission: pi path "+link+": unapproved symlink/redirection" {
		t.Fatalf("prefixed safe path: %v", err)
	}
	cs.Diffs[0].Tool = "lax"
	err = Admit(laxDescriptor(nil), f.admission(cs, nil))
	if err == nil || err.Error() != "pi path "+link+": unapproved symlink/redirection" {
		t.Fatalf("unprefixed safe path: %v", err)
	}
	plain := Descriptor{Name: "plain", ContextFile: "PLAIN.md", GlobalMarker: "~/.plain"}
	cs.Diffs[0].Tool = "plain"
	if err := Admit(plain, f.admission(cs, nil)); err != nil {
		t.Fatalf("SafePathCheck false still checked: %v", err)
	}
}

func TestAdmitOwnershipStrictAndLenient(t *testing.T) {
	f := newAdmitFixture(t)
	path := filepath.Join(f.project, ".toy/skills/alpha/SKILL.md")
	owner := func(tool, artifact string, before []byte) []state.Item {
		return []state.Item{{Artifact: artifact, Tool: tool, Scope: "local", Files: []state.FileState{{Path: path, Checksum: state.Checksum(before), Action: "CREATE"}}}}
	}
	for _, tc := range []struct {
		name   string
		d      Descriptor
		fd     diff.FileDiff
		exists bool
		owners []state.Item
		want   string
	}{
		{"strict unknown existing file", toyDescriptor(nil), diff.FileDiff{Action: diff.Skip, Before: []byte("b")}, true, nil, "toy destination " + path + " has unknown ownership; explicit adoption/migration required"},
		{"strict different owner", toyDescriptor(nil), diff.FileDiff{Action: diff.Skip, Before: []byte("b")}, true, owner("other", "alpha", []byte("b")), "toy destination " + path + " owned by alpha (other), selected alpha"},
		{"strict drift", toyDescriptor(nil), diff.FileDiff{Action: diff.Conflict, Before: []byte("edited")}, true, owner("toy", "alpha", []byte("b")), "toy ownership drift at " + path + "; preserve edits, reconcile before mutation (no --force bypass)"},
		{"strict owned unchanged", toyDescriptor(nil), diff.FileDiff{Action: diff.Skip, Before: []byte("b")}, true, owner("toy", "alpha", []byte("b")), ""},
		{"strict already absent", toyDescriptor(nil), diff.FileDiff{Action: diff.Skip, After: []byte{}}, false, owner("toy", "alpha", []byte("b")), ""},
		{"lenient unknown existing file admitted", laxDescriptor(nil), diff.FileDiff{Action: diff.Skip, Before: []byte("b")}, true, nil, ""},
		{"lenient different owner", laxDescriptor(nil), diff.FileDiff{Action: diff.Skip, Before: []byte("b")}, true, owner("other", "alpha", []byte("b")), "lax destination " + path + " owned by alpha (other), selected alpha"},
		{"lenient conflict unowned", laxDescriptor(nil), diff.FileDiff{Action: diff.Conflict, Before: []byte("b")}, true, nil, "lax resource conflict at " + path + ": existing source and selected alpha; resolve migration before applying selection"},
		{"lenient conflict owned drifted", laxDescriptor(nil), diff.FileDiff{Action: diff.Conflict, Before: []byte("edited")}, true, owner("lax", "alpha", []byte("b")), "lax resource conflict at " + path + ": existing source and selected alpha; resolve migration before applying selection"},
		{"lenient conflict owned unchanged", laxDescriptor(nil), diff.FileDiff{Action: diff.Conflict, Before: []byte("b")}, true, owner("lax", "alpha", []byte("b")), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fd := tc.fd
			fd.Tool, fd.Type, fd.Artifact, fd.Path, fd.Scope = tc.d.Name, "skill", "alpha", path, "local"
			if fd.After == nil {
				fd.After = []byte("alpha\n")
			}
			if tc.exists {
				f.write(t, path, fd.Before)
			} else {
				_ = os.Remove(path)
			}
			err := Admit(tc.d, f.admission(&diff.ChangeSet{Diffs: []diff.FileDiff{fd}}, tc.owners))
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.want {
				t.Fatalf("ownership\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

func TestAdmitContextRules(t *testing.T) {
	f := newAdmitFixture(t)
	path := filepath.Join(f.project, "TOY.md")
	owned := adapter.AppendSection([]byte("User prose\n"), "alpha", []byte("old"))
	ownedLax := adapter.AppendSection([]byte("User prose\n"), "lax:alpha", []byte("old"))
	section := func(d Descriptor, before []byte) diff.FileDiff {
		name := d.SectionPrefix + "alpha"
		return diff.FileDiff{Tool: d.Name, Type: "instruction", Artifact: "alpha", Path: path, Scope: "local", Action: diff.Append, Section: &diff.SectionEdit{Name: name}, Before: before, After: adapter.AppendSection(before, name, []byte("new"))}
	}
	sectionOwner := func(tool, artifact, sectionName, at, action string, before []byte) state.Item {
		return state.Item{Artifact: artifact, Tool: tool, Scope: "local", Files: []state.FileState{{Path: at, Section: sectionName, Checksum: state.Checksum(before), Action: action}}}
	}
	toy, lax := toyDescriptor(nil), laxDescriptor(nil)
	for _, tc := range []struct {
		name   string
		d      Descriptor
		fd     diff.FileDiff
		files  map[string][]byte
		owners []state.Item
		want   string
		review bool
	}{
		{"strict shadow", toy, section(toy, nil), map[string][]byte{filepath.Join(f.project, "TOY.override.md"): []byte("s")}, nil, "toy context " + path + " is shadowed by " + filepath.Join(f.project, "TOY.override.md"), false},
		{"lenient ignores shadow files", lax, section(lax, nil), map[string][]byte{filepath.Join(f.project, "TOY.override.md"): []byte("s")}, nil, "", false},
		{"strict relocation any section", toy, section(toy, nil), nil, []state.Item{sectionOwner("toy", "alpha", "weird", filepath.Join(f.project, "docs/TOY.md"), "APPEND", nil)}, "toy context ownership at " + filepath.Join(f.project, "docs/TOY.md") + " would be shadowed/relocated to " + path, false},
		{"lenient relocation selected section", lax, section(lax, nil), nil, []state.Item{sectionOwner("lax", "alpha", "lax:alpha", filepath.Join(f.project, "docs/TOY.md"), "APPEND", nil)}, "lax context ownership at " + filepath.Join(f.project, "docs/TOY.md") + " would be shadowed/relocated to " + path, false},
		{"lenient relocation ignores other sections", lax, section(lax, nil), nil, []state.Item{sectionOwner("lax", "alpha", "lax:other", filepath.Join(f.project, "docs/TOY.md"), "APPEND", nil)}, "", false},
		{"strict mixed context refused", toy, section(toy, owned), map[string][]byte{path: owned}, []state.Item{sectionOwner("other", "beta", "beta", path, "APPEND", owned)}, "toy mixed-context " + path + " owned by beta (other); explicit migration/consent required", false},
		{"lenient mixed context reviewed", lax, section(lax, ownedLax), map[string][]byte{path: ownedLax}, []state.Item{sectionOwner("lax", "alpha", "lax:alpha", path, "APPEND", ownedLax), sectionOwner("other", "beta", "beta", path, "APPEND", ownedLax)}, "", true},
		{"strict drift", toy, section(toy, owned), map[string][]byte{path: owned}, []state.Item{sectionOwner("toy", "alpha", "alpha", path, "APPEND", []byte("recorded"))}, "toy context drift at " + path + " section alpha: recorded bytes differ; preserve edits and reconcile before apply", false},
		{"lenient drift", lax, section(lax, ownedLax), map[string][]byte{path: ownedLax}, []state.Item{sectionOwner("lax", "alpha", "lax:alpha", path, "APPEND", []byte("recorded"))}, "lax context drift at " + path + " section lax:alpha: recorded bytes differ; preserve edits and reconcile before apply", false},
		{"strict ambiguous action", toy, section(toy, owned), map[string][]byte{path: owned}, []state.Item{sectionOwner("toy", "alpha", "alpha", path, "CREATE", owned)}, "toy context " + path + " has ambiguous section ownership", false},
		{"strict unknown ownership", toy, section(toy, []byte("prose\n")), map[string][]byte{path: []byte("prose\n")}, nil, "toy context " + path + " has unknown ownership; preserve prose and obtain explicit migration/consent", false},
		{"lenient unknown section", lax, section(lax, ownedLax), map[string][]byte{path: ownedLax}, nil, "lax context " + path + " section lax:alpha has unknown ownership; refuse replacement", false},
		{"lenient owned admitted", lax, section(lax, ownedLax), map[string][]byte{path: ownedLax}, []state.Item{sectionOwner("lax", "alpha", "lax:alpha", path, "APPEND", ownedLax)}, "", false},
		{"strict owned admitted", toy, section(toy, owned), map[string][]byte{path: owned}, []state.Item{sectionOwner("toy", "alpha", "alpha", path, "APPEND", owned)}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(path)
			_ = os.Remove(filepath.Join(f.project, "TOY.override.md"))
			for p, b := range tc.files {
				f.write(t, p, b)
			}
			var reviews []ContextReview
			in := f.admission(&diff.ChangeSet{Diffs: []diff.FileDiff{tc.fd}}, tc.owners)
			in.Review = func(r ContextReview) error { reviews = append(reviews, r); return nil }
			err := Admit(tc.d, in)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.want {
				t.Fatalf("context\n got: %s\nwant: %s", got, tc.want)
			}
			if (len(reviews) == 1) != tc.review {
				t.Fatalf("reviews = %+v want review=%v", reviews, tc.review)
			}
			if tc.review && (reviews[0].Path != path || reviews[0].Scope != "local" || len(reviews[0].OtherOwners) != 1 || reviews[0].OtherOwners[0] != "beta (other/local) at "+path) {
				t.Fatalf("review contents: %+v", reviews[0])
			}
		})
	}
	// A refusing Review callback stops the admission.
	in := f.admission(&diff.ChangeSet{Diffs: []diff.FileDiff{section(lax, ownedLax)}}, []state.Item{sectionOwner("lax", "alpha", "lax:alpha", path, "APPEND", ownedLax), sectionOwner("other", "beta", "beta", path, "APPEND", ownedLax)})
	f.write(t, path, ownedLax)
	in.Review = func(ContextReview) error { return os.ErrPermission }
	if err := Admit(lax, in); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("review refusal not propagated: %v", err)
	}
}

func TestAdmitResourceRules(t *testing.T) {
	f := newAdmitFixture(t)
	legacyRoot := filepath.Join(f.home, ".toy-legacy")
	roots := func(in RootInputs) ([]scan.SkillRoot, error) {
		return []scan.SkillRoot{
			{Path: filepath.Join(f.home, ".toy/skills"), Recursive: true, DuplicatePolicy: scan.RefuseDifferentBytesVerifiedException, Name: toyName},
			{Path: legacyRoot, Legacy: true, Recursive: true, DuplicatePolicy: scan.RefuseDifferentBytesVerifiedException, Name: toyName},
		}, nil
	}
	selected := filepath.Join(f.project, ".toy/skills/alpha/SKILL.md")
	other := filepath.Join(f.home, ".toy/skills/alpha/SKILL.md")
	legacy := filepath.Join(legacyRoot, "alpha/SKILL.md")
	skill := func(d Descriptor) diff.FileDiff {
		return diff.FileDiff{Tool: d.Name, Type: "skill", Artifact: "alpha", Path: selected, Scope: "local", Action: diff.Create, After: []byte("alpha\n")}
	}
	toy := toyDescriptor(roots)
	named := toy
	named.SkillName = func(raw []byte, path string) (string, error) { return "", os.ErrInvalid }
	for _, tc := range []struct {
		name   string
		d      Descriptor
		files  map[string][]byte
		owners []state.Item
		want   string
	}{
		{"identical bytes admitted", toy, map[string][]byte{other: []byte("alpha\n")}, nil, ""},
		{"different bytes refused", toy, map[string][]byte{other: []byte("alpha\nv2\n")}, nil, "toy skill identity \"alpha\" conflict: " + other + " () and " + selected + " (selected local); silent shadowing refused"},
		{"different bytes admitted when verified", toy, map[string][]byte{other: []byte("alpha\nv2\n")}, []state.Item{{Artifact: "alpha", Tool: "toy", Scope: "global", Files: []state.FileState{{Path: other, Checksum: state.Checksum([]byte("alpha\nv2\n")), Action: "CREATE"}}}}, ""},
		{"legacy discovered", toy, map[string][]byte{legacy: []byte("alpha\n")}, nil, "toy legacy skill " + legacy + ": automatic migration is Partial; preserve files and reconcile ownership/checksums and retire the old root explicitly before installing alpha"},
		{"legacy ownership", toy, map[string][]byte{legacy: []byte("alpha\n")}, []state.Item{{Artifact: "alpha", Tool: "toy", Scope: "global", Files: []state.FileState{{Path: legacy, Checksum: state.Checksum([]byte("alpha\n")), Action: "CREATE"}}}}, "toy legacy ownership at " + legacy + " requires explicit migration/retirement"},
		{"legacy ownership drift", toy, map[string][]byte{legacy: []byte("alpha\nedited\n")}, []state.Item{{Artifact: "alpha", Tool: "toy", Scope: "global", Files: []state.FileState{{Path: legacy, Checksum: state.Checksum([]byte("alpha\n")), Action: "CREATE"}}}}, "toy legacy ownership checksum drift at " + legacy + "; preserve edits, explicit migration required (no --force bypass)"},
		{"skill name rule", named, nil, nil, os.ErrInvalid.Error()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.RemoveAll(filepath.Join(f.home, ".toy"))
			_ = os.RemoveAll(legacyRoot)
			for p, b := range tc.files {
				f.write(t, p, b)
			}
			err := Admit(tc.d, f.admission(&diff.ChangeSet{Diffs: []diff.FileDiff{skill(tc.d)}}, tc.owners))
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.want {
				t.Fatalf("resources\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
	// A Prepare hook that supplies the inventory is honoured and RefuseAlways
	// refuses identical bytes with the identity text.
	lax := laxDescriptor(func(RootInputs) ([]scan.SkillRoot, error) {
		return []scan.SkillRoot{{Path: filepath.Join(f.home, ".lax/skills"), DuplicatePolicy: scan.RefuseAlways, Name: toyName}}, nil
	})
	lax.Validator = inventoryValidator{inventory: []scan.Resource{{Kind: "skill", Name: "alpha", Path: other, Origin: "native user", Content: []byte("alpha\n")}}}
	err := Admit(lax, f.admission(&diff.ChangeSet{Diffs: []diff.FileDiff{skill(lax)}}, nil))
	want := "lax skill identity \"alpha\" conflict: " + other + " (native user) and " + selected + " (selected local); silent shadowing refused"
	if err == nil || err.Error() != want {
		t.Fatalf("refuse-always identity\n got: %v\nwant: %s", err, want)
	}
}

type inventoryValidator struct{ inventory []scan.Resource }

func (v inventoryValidator) Prepare(_ Descriptor, in *Admission) error {
	in.Inventory, in.InventoryResolved = v.inventory, true
	return nil
}
func (inventoryValidator) BeforeOwnership(Descriptor, *Admission, diff.FileDiff) error { return nil }
func (inventoryValidator) AfterOwnership(Descriptor, *Admission, diff.FileDiff, *ContextFacts) error {
	return nil
}
func (inventoryValidator) AfterContext(Descriptor, *Admission, diff.FileDiff, *ContextFacts) error {
	return nil
}

// HS-D1: the shared admission never names a target.
func TestAdmitNamesNoTarget(t *testing.T) {
	for _, file := range []string{"admit.go", "registry.go", "policies.go", "harness.go"} {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, literal := range []string{`"codex"`, `"pi"`, `"claude"`, `"opencode"`} {
			if strings.Contains(string(raw), literal) {
				t.Fatalf("%s names a target: %s", file, literal)
			}
		}
	}
}

// HS-IR-04: roots supplied for other targets join discovery, so a same-name
// resource another target can see is refused under the selected root's policy.
func TestAdmitForeignRootsJoinDiscovery(t *testing.T) {
	f := newAdmitFixture(t)
	foreign := filepath.Join(f.home, ".foreign/skills")
	f.write(t, filepath.Join(foreign, "alpha/SKILL.md"), []byte("alpha\n"))
	selected := filepath.Join(f.project, ".lax/skills/alpha/SKILL.md")
	lax := laxDescriptor(func(RootInputs) ([]scan.SkillRoot, error) {
		return []scan.SkillRoot{{Path: filepath.Join(f.home, ".lax/skills"), DuplicatePolicy: scan.RefuseAlways, Name: toyName}}, nil
	})
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Tool: "lax", Type: "skill", Artifact: "alpha", Path: selected, Scope: "local", Action: diff.Create, After: []byte("alpha\n")}}}
	if err := Admit(lax, f.admission(cs, nil)); err != nil {
		t.Fatalf("without foreign roots: %v", err)
	}
	in := f.admission(cs, nil)
	in.Roots = []scan.SkillRoot{{Path: foreign, Origin: "foreign user", Recursive: true, DuplicatePolicy: scan.RefuseDifferentBytesVerifiedException, Name: toyName}}
	err := Admit(lax, in)
	want := "lax skill identity \"alpha\" conflict: " + filepath.Join(foreign, "alpha/SKILL.md") + " (foreign user) and " + selected + " (selected local); silent shadowing refused"
	if err == nil || err.Error() != want {
		t.Fatalf("foreign root collision\n got: %v\nwant: %s", err, want)
	}
}

// HS-FR-03: foreign roots join the inventory a Prepare hook pre-resolved.
func TestAdmitForeignRootsJoinPreResolvedInventory(t *testing.T) {
	f := newAdmitFixture(t)
	foreign := filepath.Join(f.home, ".foreign/skills")
	f.write(t, filepath.Join(foreign, "alpha/SKILL.md"), []byte("alpha\n"))
	selected := filepath.Join(f.project, ".lax/skills/alpha/SKILL.md")
	lax := laxDescriptor(func(RootInputs) ([]scan.SkillRoot, error) {
		return []scan.SkillRoot{{Path: filepath.Join(f.home, ".lax/skills"), DuplicatePolicy: scan.RefuseAlways, Name: toyName}}, nil
	})
	lax.Validator = inventoryValidator{inventory: []scan.Resource{}}
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Tool: "lax", Type: "skill", Artifact: "alpha", Path: selected, Scope: "local", Action: diff.Create, After: []byte("alpha\n")}}}
	in := f.admission(cs, nil)
	in.Roots = []scan.SkillRoot{{Path: foreign, Origin: "foreign user", Recursive: true, DuplicatePolicy: scan.RefuseDifferentBytesVerifiedException, Name: toyName}}
	err := Admit(lax, in)
	want := "lax skill identity \"alpha\" conflict: " + filepath.Join(foreign, "alpha/SKILL.md") + " (foreign user) and " + selected + " (selected local); silent shadowing refused"
	if err == nil || err.Error() != want {
		t.Fatalf("pre-resolved inventory ignored foreign roots\n got: %v\nwant: %s", err, want)
	}
	// A foreign walk error surfaces too.
	in = f.admission(cs, nil)
	in.Roots = []scan.SkillRoot{{Path: foreign, Recursive: true, DuplicatePolicy: scan.RefuseAlways, Name: func([]byte, string) (string, error) { return "", os.ErrInvalid }}}
	if err := Admit(lax, in); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("foreign walk error lost: %v", err)
	}
}
