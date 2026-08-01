package manifest

import (
	"strings"
	"testing"
)

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

func TestValidateModeNormalizesEmptyToInline(t *testing.T) {
	a := &Artifact{
		Meta: Meta{APIVersion: "patronus/v2", Family: FamilyArtifact, Name: "x", Version: "1.0.0"},
		Type: TypeInstruction, Entry: "I.md", Targets: []string{"claude"},
	}
	a.Description = "d"
	got, err := finishArtifact(a)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode != "inline" {
		t.Errorf("Mode = %q, want inline", got.Mode)
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
