package versionbump

import "testing"

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		change  Change
		violate bool
	}{
		{
			name:    "content changed without bump fails",
			change:  Change{Name: "skills/foo", ContentChanged: true, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: "1.0.0"},
			violate: true,
		},
		{
			name:    "content changed with bump passes",
			change:  Change{Name: "skills/foo", ContentChanged: true, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: "1.1.0"},
			violate: false,
		},
		{
			name:    "version-only change passes",
			change:  Change{Name: "skills/foo", ContentChanged: false, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: "1.1.0"},
			violate: false,
		},
		{
			name:    "new artifact has no base to compare",
			change:  Change{Name: "skills/new", ContentChanged: true, ExistedInBase: false, BaseVersion: "", HeadVersion: "1.0.0"},
			violate: false,
		},
		{
			name:    "deleted artifact does not violate",
			change:  Change{Name: "skills/gone", ContentChanged: true, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: ""},
			violate: false,
		},
		{
			name:    "no content change is clean",
			change:  Change{Name: "skills/foo", ContentChanged: false, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: "1.0.0"},
			violate: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Check([]Change{tt.change})
			if tt.violate && len(got) != 1 {
				t.Fatalf("want 1 violation, got %d: %v", len(got), got)
			}
			if !tt.violate && len(got) != 0 {
				t.Fatalf("want 0 violations, got %d: %v", len(got), got)
			}
			if tt.violate && got[0].Name != tt.change.Name {
				t.Errorf("violation names %q, want %q", got[0].Name, tt.change.Name)
			}
		})
	}
}

func TestCheckPreservesOrderAndReportsAll(t *testing.T) {
	changes := []Change{
		{Name: "a", ContentChanged: true, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: "1.0.0"},
		{Name: "b", ContentChanged: true, ExistedInBase: true, BaseVersion: "2.0.0", HeadVersion: "2.1.0"}, // bumped, clean
		{Name: "c", ContentChanged: true, ExistedInBase: true, BaseVersion: "3.0.0", HeadVersion: "3.0.0"},
	}
	got := Check(changes)
	if len(got) != 2 {
		t.Fatalf("want 2 violations, got %d: %v", len(got), got)
	}
	if got[0].Name != "a" || got[1].Name != "c" {
		t.Errorf("violations = [%s %s], want [a c]", got[0].Name, got[1].Name)
	}
	if got[0].Version != "1.0.0" {
		t.Errorf("violation a version = %q, want 1.0.0", got[0].Version)
	}
}

func TestCheckEmpty(t *testing.T) {
	if got := Check(nil); got != nil {
		t.Errorf("Check(nil) = %v, want nil", got)
	}
}
