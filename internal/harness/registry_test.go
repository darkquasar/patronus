package harness

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/scan"
)

func toyRoots(RootInputs) ([]scan.SkillRoot, error) { return nil, nil }

func toyRegistry(t *testing.T) *Registry {
	t.Helper()
	r, err := NewRegistry(
		Descriptor{Name: "toy", Profiles: []string{"toy-profile"}, ShadowFiles: []string{"TOY.override.md"}, SkillRoots: toyRoots, ResourcePhase: ResourceAfterContext, LockTargeted: true, LockCoexistence: true, OwnsEffects: true},
		Descriptor{Name: "lax", Profiles: []string{"lax-profile", "shared-profile"}, LockTargeted: true},
		Descriptor{Name: "plain"},
		Descriptor{Name: "other", Profiles: []string{"shared-profile"}, OwnsEffects: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestNewRegistryRejectsDuplicateZeroNameAndZeroPhase(t *testing.T) {
	for _, tc := range []struct {
		name string
		ds   []Descriptor
		want error
	}{
		{"zero name", []Descriptor{{}}, ErrZeroName},
		{"duplicate", []Descriptor{{Name: "toy"}, {Name: "toy"}}, ErrDuplicateDescriptor},
		{"roots without phase", []Descriptor{{Name: "toy", SkillRoots: toyRoots}}, ErrZeroResourcePhase},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewRegistry(tc.ds...); !errors.Is(err, tc.want) {
				t.Fatalf("NewRegistry = %v want %v", err, tc.want)
			}
		})
	}
	if _, err := NewRegistry(Descriptor{Name: "toy"}, Descriptor{Name: "lax", SkillRoots: toyRoots, ResourcePhase: ResourceBeforeOwnership}); err != nil {
		t.Fatalf("valid registry refused: %v", err)
	}
}

func TestValidateTargetSentinelsAndNames(t *testing.T) {
	r := toyRegistry(t)
	for _, target := range []string{TargetUnset, TargetAll, "toy", "lax", "plain", "other"} {
		if err := r.ValidateTarget(target); err != nil {
			t.Fatalf("ValidateTarget(%q) = %v", target, err)
		}
	}
	err := r.ValidateTarget("invented")
	if err == nil || err.Error() != "unknown target \"invented\"; use lax|other|plain|toy|all" {
		t.Fatalf("unknown target text: %v", err)
	}
}

func TestProfileTargetRule(t *testing.T) {
	r := toyRegistry(t)
	for _, tc := range []struct {
		profile  string
		declared []string
		target   string
		want     string
	}{
		{"toy-profile", nil, "toy", ""},
		{"toy-profile", nil, "other", "toy-profile requires --target toy (got \"other\")"},
		{"toy-profile", nil, "", "toy-profile requires --target toy (got \"\")"},
		{"toy-profile", nil, TargetAll, "toy-profile requires --target toy (got \"all\")"},
		{"shared-profile", nil, "lax", ""},
		{"shared-profile", nil, "other", ""},
		{"shared-profile", nil, "toy", "shared-profile requires --target lax|other (got \"toy\")"},
		{"undeclared", nil, "anything", ""},
		{"", nil, "", ""},
		{"declared-profile", []string{"plain"}, "plain", ""},
		{"declared-profile", []string{"plain"}, "toy", "declared-profile requires --target plain (got \"toy\")"},
		{"toy-profile", []string{"plain", "toy"}, "plain", ""},
		{"toy-profile", []string{"plain"}, "other", "toy-profile requires --target toy|plain (got \"other\")"},
	} {
		t.Run(tc.profile+"/"+tc.target, func(t *testing.T) {
			err := r.ProfileTarget(tc.profile, tc.declared, tc.target)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.want {
				t.Fatalf("ProfileTarget\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// The four prior/target pairs of the historical Codex test over invented
// names: only toy carries LockCoexistence, lax is LockTargeted only.
func TestLockTargetCoexistenceFlag(t *testing.T) {
	r := toyRegistry(t)
	const conflict = "existing lock target %s conflicts with --target %s; use separate project roots or explicit migration"
	for _, tc := range []struct {
		prior, target string
		want          string
	}{
		{"lax", "toy", conflict},
		{"toy", "lax", conflict},
		{"toy", "plain", conflict},
		{"plain", "toy", conflict},
		{"toy", "toy", ""},
		{"lax", "plain", ""}, // neither side names a coexistence descriptor
		{"lax", "other", ""}, // LockTargeted only: accepted
		{"plain", "lax", ""}, // untargeted prior, targeted request: accepted
		{"", "toy", "existing legacy lock target is unknown; preserve patronus.lock and reconcile provenance before Codex migration"},
		{"", "lax", ""},
	} {
		t.Run(tc.prior+"->"+tc.target, func(t *testing.T) {
			prior := &lock.Lock{Version: lock.TargetVersion, Target: tc.prior, Profile: "invented"}
			err := r.LockTarget(prior, tc.target)
			want := tc.want
			if strings.Contains(want, "%s") {
				want = strings.Replace(strings.Replace(want, "%s", tc.prior, 1), "%s", tc.target, 1)
			}
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != want {
				t.Fatalf("LockTarget\n got: %s\nwant: %s", got, want)
			}
		})
	}
	// An empty legacy lock (no profile, no entries) can be crossed into toy.
	if err := r.LockTarget(&lock.Lock{Version: lock.Version}, "toy"); err != nil {
		t.Fatalf("empty legacy lock refused: %v", err)
	}
}

func TestTargetedAndSelected(t *testing.T) {
	r := toyRegistry(t)
	if got := names(r.Targeted()); got != "toy,lax,other" {
		t.Fatalf("Targeted = %s", got)
	}
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Tool: "plain"}, {Tool: "lax+toy"}, {Tool: "unknown"}}}
	if got := names(r.Selected(cs)); got != "toy,lax,plain" {
		t.Fatalf("Selected = %s", got)
	}
	if got := r.Selected(&diff.ChangeSet{}); got != nil {
		t.Fatalf("Selected(empty) = %+v", got)
	}
}

func TestLookupReturnsCopies(t *testing.T) {
	profiles := []string{"toy-profile"}
	r, err := NewRegistry(Descriptor{Name: "toy", Profiles: profiles, ShadowFiles: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	profiles[0] = "mutated-by-caller"
	d, ok := r.Lookup("toy")
	if !ok || d.Profiles[0] != "toy-profile" {
		t.Fatalf("registry shares the caller's slice: %+v", d)
	}
	d.Profiles[0], d.ShadowFiles[0] = "mutated-by-reader", "b"
	again, _ := r.Lookup("toy")
	if again.Profiles[0] != "toy-profile" || again.ShadowFiles[0] != "a" {
		t.Fatalf("Lookup shares registry state: %+v", again)
	}
	if _, ok := r.Lookup("missing"); ok {
		t.Fatal("missing name found")
	}
	targeted := r.Targeted()
	if targeted != nil {
		t.Fatalf("untargeted descriptor listed: %+v", targeted)
	}
}

func TestNilRegistryFailsClosed(t *testing.T) {
	var r *Registry
	if err := r.ValidateTarget("toy"); !errors.Is(err, ErrNilRegistry) {
		t.Fatalf("ValidateTarget on nil = %v", err)
	}
	if err := r.ProfileTarget("p", nil, "toy"); !errors.Is(err, ErrNilRegistry) {
		t.Fatalf("ProfileTarget on nil = %v", err)
	}
	if err := r.LockTarget(&lock.Lock{}, "toy"); !errors.Is(err, ErrNilRegistry) {
		t.Fatalf("LockTarget on nil = %v", err)
	}
	if _, ok := r.Lookup("toy"); ok {
		t.Fatal("Lookup on nil found a descriptor")
	}
	if r.Targeted() != nil || r.Selected(&diff.ChangeSet{Diffs: []diff.FileDiff{{Tool: "toy"}}}) != nil {
		t.Fatal("nil registry returned descriptors")
	}
}

// HS-O2: the lock package keeps its own target enum; its acceptance through
// lock.Save (which calls validateTarget) must equal ValidateTarget's for the
// named set, while the unset sentinel is accepted by the registry only.
func TestLockEnumParity(t *testing.T) {
	r, err := NewRegistry(Descriptor{Name: "claude"}, Descriptor{Name: "codex"}, Descriptor{Name: "opencode"}, Descriptor{Name: "pi"})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, target := range []string{"all", "claude", "codex", "opencode", "pi", "invented", "toy"} {
		saveErr := lock.Save(filepath.Join(dir, "patronus.lock"), &lock.Lock{Version: lock.TargetVersion, Target: target})
		if (saveErr == nil) != (r.ValidateTarget(target) == nil) {
			t.Fatalf("enum parity broken for %q: lock %v registry %v", target, saveErr, r.ValidateTarget(target))
		}
	}
	if err := lock.Save(filepath.Join(dir, "patronus.lock"), &lock.Lock{Version: lock.TargetVersion}); err == nil {
		t.Fatal("lock accepted the unset sentinel on the target-bearing schema")
	}
	if err := r.ValidateTarget(TargetUnset); err != nil {
		t.Fatalf("registry refused the unset sentinel: %v", err)
	}
}

func names(ds []Descriptor) string {
	var out []string
	for _, d := range ds {
		out = append(out, d.Name)
	}
	return strings.Join(out, ",")
}
