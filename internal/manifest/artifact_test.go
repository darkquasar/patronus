package manifest

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The catalog build re-marshals each manifest into its tarball, so the marshalled
// form of an unchanged, mode-less instruction must be byte-stable across releases —
// in particular it must NOT sprout a `mode:` line. This is the exact regression that
// tripped the R2 immutability guard when finishArtifact normalized "" -> inline.
func TestMarshalOmitsEmptyMode(t *testing.T) {
	a := &Artifact{
		Meta: Meta{APIVersion: "patronus/v2", Family: FamilyArtifact, Name: "agents-spine", Version: "1.0.0"},
		Type: TypeInstruction, Entry: "INSTRUCTIONS.md", Targets: []string{"claude"},
	}
	a.Description = "d"
	got, err := finishArtifact(a)
	if err != nil {
		t.Fatal(err)
	}
	b, err := yaml.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "mode:") {
		t.Errorf("marshalled mode-less instruction must not contain a mode: line:\n%s", b)
	}
}

func pointerArtifact() *Artifact {
	return &Artifact{
		Meta:    Meta{APIVersion: "patronus/v2", Family: FamilyArtifact, Name: "ticket", Version: "1.0.0"},
		Type:    TypeInstruction,
		Mode:    "pointer",
		Pointer: &InstructionPointer{Trigger: "for multi-session work"},
		Entry:   "INSTRUCTIONS.md",
		Targets: []string{"claude"},
	}
	// Description is set per-test.
}

// An instruction with no mode: must stay mode-empty after load — NOT normalized to
// "inline". Normalizing would materialize `mode: inline` into the re-marshalled
// manifest the catalog build packs, changing every instruction's tarball bytes with
// no authored change and tripping the R2 immutability guard. Empty already means
// inline at every read site.
func TestValidateModeStaysEmptyWhenUnset(t *testing.T) {
	a := &Artifact{
		Meta: Meta{APIVersion: "patronus/v2", Family: FamilyArtifact, Name: "x", Version: "1.0.0"},
		Type: TypeInstruction, Entry: "I.md", Targets: []string{"claude"},
	}
	a.Description = "d"
	got, err := finishArtifact(a)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != "" {
		t.Errorf("Mode = %q, want empty (inline is implicit, not materialized)", got.Mode)
	}
}

func TestValidateModeOnlyOnInstruction(t *testing.T) {
	a := &Artifact{
		Meta: Meta{APIVersion: "patronus/v2", Family: FamilyArtifact, Name: "x", Version: "1.0.0"},
		Type: TypeSkill, Mode: "pointer", Entry: "S.md", Targets: []string{"claude"},
	}
	a.Description = "d"
	err := a.Validate()
	if err == nil || !strings.Contains(err.Error(), "only valid on instruction") {
		t.Errorf("want instruction-only error, got %v", err)
	}
}

func TestValidateModeEnum(t *testing.T) {
	a := pointerArtifact()
	a.Description = "d"
	a.Mode = "bogus"
	err := a.Validate()
	if err == nil || !strings.Contains(err.Error(), "invalid instruction mode") {
		t.Errorf("want enum error, got %v", err)
	}
}

func TestValidatePointerRequiresTrigger(t *testing.T) {
	a := pointerArtifact()
	a.Description = "d"
	a.Pointer = &InstructionPointer{Trigger: ""}
	err := a.Validate()
	if err == nil || !strings.Contains(err.Error(), "pointer.trigger is required") {
		t.Errorf("want trigger-required error, got %v", err)
	}
}

func TestValidatePointerBlockRequiredInPointerMode(t *testing.T) {
	a := pointerArtifact()
	a.Description = "d"
	a.Pointer = nil
	err := a.Validate()
	if err == nil || !strings.Contains(err.Error(), "pointer.trigger is required") {
		t.Errorf("want pointer-block-required error, got %v", err)
	}
}

func TestValidateStrayPointerBlockRejected(t *testing.T) {
	a := &Artifact{
		Meta: Meta{APIVersion: "patronus/v2", Family: FamilyArtifact, Name: "x", Version: "1.0.0"},
		Type: TypeInstruction, Mode: "inline",
		Pointer: &InstructionPointer{Trigger: "t"},
		Entry:   "I.md", Targets: []string{"claude"},
	}
	a.Description = "d"
	err := a.Validate()
	if err == nil || !strings.Contains(err.Error(), "pointer is only valid with mode: pointer") {
		t.Errorf("want stray-pointer error, got %v", err)
	}
}

func TestValidatePointerModeValid(t *testing.T) {
	a := pointerArtifact()
	a.Description = "d"
	if err := a.Validate(); err != nil {
		t.Errorf("valid pointer artifact rejected: %v", err)
	}
}

func TestInstructionLayoutPointerDirForScope(t *testing.T) {
	l := &InstructionLayout{
		PointerDirGlobal:  PathTarget{Path: "~/g/{name}.md", Set: true},
		PointerDirProject: PathTarget{Path: "p/{name}.md", Set: true},
	}
	if got := l.PointerDirFor("global"); got.Path != "~/g/{name}.md" {
		t.Errorf("global pointerDir = %q", got.Path)
	}
	if got := l.PointerDirFor("local"); got.Path != "p/{name}.md" {
		t.Errorf("local pointerDir = %q", got.Path)
	}
}
