package profile

import (
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
)

// These pin the L6 design decision that the profile `sandbox` slot is a LIST (not a
// scalar) so ONE profile can carry per-tool sandbox flavours — native on
// claude/codex, sandbox-runtime on opencode — and the resolver picks exactly the
// one matching --tool. The recipes themselves land in P7.5 (they need a
// settings-merge primitive); this is the resolver-level proof the slot is ready.

// hardenedLike models the eventual `hardened` profile's sandbox slot: three
// per-tool flavours of the L6 recipe, no bare fallback.
func hardenedLike() *registry.Catalog {
	p := &manifest.Profile{
		Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "hardened"},
		Layers: manifest.ProfileLayers{
			Sandbox: manifest.StringList{
				"native-sandbox@claude",
				"native-sandbox@codex",
				"sandbox-runtime@opencode",
			},
		},
	}
	return fakeCatalog(nil, []string{"native-sandbox", "sandbox-runtime"}, p)
}

func TestSandboxSlotFlavourSelectsOnePerTool(t *testing.T) {
	cat := hardenedLike()
	for _, tc := range []struct {
		tool, want string
	}{
		{"claude", "native-sandbox"},
		{"codex", "native-sandbox"},
		{"opencode", "sandbox-runtime"},
	} {
		r, err := Resolve(cat, "hardened", tc.tool)
		if err != nil {
			t.Fatalf("%s: %v", tc.tool, err)
		}
		// Exactly one sandbox resolves for a concrete tool — never two.
		if len(r.Items) != 1 {
			t.Fatalf("%s: got %d sandbox items, want exactly 1: %v", tc.tool, len(r.Items), r.Names())
		}
		if r.Items[0].Name != tc.want {
			t.Errorf("%s: sandbox = %q, want %q", tc.tool, r.Items[0].Name, tc.want)
		}
		if r.Items[0].Slot != "sandbox" {
			t.Errorf("%s: slot = %q, want sandbox", tc.tool, r.Items[0].Slot)
		}
	}
}

func TestSandboxSlotAllToolDropsEveryFlavour(t *testing.T) {
	cat := hardenedLike()
	// The tool-agnostic baseline (lock default) keeps only bare names; every sandbox
	// entry here is @tool-flavoured, so NONE resolve — the lock pins no sandbox.
	for _, tool := range []string{"all", ""} {
		r, err := Resolve(cat, "hardened", tool)
		if err != nil {
			t.Fatalf("%q: %v", tool, err)
		}
		if len(r.Items) != 0 {
			t.Errorf("%q: got %v, want no sandbox items (flavours drop on the agnostic baseline)", tool, r.Names())
		}
	}
}

// TestSandboxSlotExtendsAppends proves a child profile's sandbox flavours APPEND to
// the parent's across `extends` (list-slot semantics), rather than replacing — so a
// base can set some tools' sandbox and a derived profile add another's.
func TestSandboxSlotExtendsAppends(t *testing.T) {
	parent := &manifest.Profile{
		Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "base"},
		Layers: manifest.ProfileLayers{
			Sandbox: manifest.StringList{"native-sandbox@claude", "native-sandbox@codex"},
		},
	}
	child := &manifest.Profile{
		Meta:    manifest.Meta{Family: manifest.FamilyProfile, Name: "derived"},
		Extends: "base",
		Layers: manifest.ProfileLayers{
			Sandbox: manifest.StringList{"sandbox-runtime@opencode"}, // appends, not replaces
		},
	}
	cat := fakeCatalog(nil, []string{"native-sandbox", "sandbox-runtime"}, parent, child)

	// opencode resolves the child-added flavour…
	r, err := Resolve(cat, "derived", "opencode")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Items) != 1 || r.Items[0].Name != "sandbox-runtime" {
		t.Fatalf("opencode = %v, want [sandbox-runtime] (child append survived)", r.Names())
	}
	// …and claude still resolves the parent's flavour (append didn't drop the parent).
	r, _ = Resolve(cat, "derived", "claude")
	if len(r.Items) != 1 || r.Items[0].Name != "native-sandbox" {
		t.Fatalf("claude = %v, want [native-sandbox] (parent flavour inherited)", r.Names())
	}
}

// TestSandboxSlotDedupsBareAndFlavoured guards the dedup-on-base invariant for the
// sandbox slot specifically: a bare name plus its @tool flavour install once.
func TestSandboxSlotDedupsBareAndFlavoured(t *testing.T) {
	p := &manifest.Profile{
		Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "p"},
		Layers: manifest.ProfileLayers{
			Sandbox: manifest.StringList{"native-sandbox", "native-sandbox@claude"},
		},
	}
	cat := fakeCatalog(nil, []string{"native-sandbox"}, p)
	r, _ := Resolve(cat, "p", "claude")
	if len(r.Items) != 1 || r.Items[0].Name != "native-sandbox" {
		t.Fatalf("bare + @claude sandbox should dedup to one base, got %v", r.Names())
	}
}
