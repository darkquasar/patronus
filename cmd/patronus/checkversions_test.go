package main

import "testing"

func TestCheckVersions(t *testing.T) {
	tests := []struct {
		name    string
		change  artifactChange
		violate bool
	}{
		{
			name:    "content changed without bump fails",
			change:  artifactChange{Name: "skills/foo", ContentChanged: true, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: "1.0.0"},
			violate: true,
		},
		{
			name:    "content changed with bump passes",
			change:  artifactChange{Name: "skills/foo", ContentChanged: true, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: "1.1.0"},
			violate: false,
		},
		{
			name:    "version-only change passes",
			change:  artifactChange{Name: "skills/foo", ContentChanged: false, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: "1.1.0"},
			violate: false,
		},
		{
			name:    "new artifact has no base to compare",
			change:  artifactChange{Name: "skills/new", ContentChanged: true, ExistedInBase: false, BaseVersion: "", HeadVersion: "1.0.0"},
			violate: false,
		},
		{
			name:    "deleted artifact does not violate",
			change:  artifactChange{Name: "skills/gone", ContentChanged: true, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: ""},
			violate: false,
		},
		{
			name:    "no content change is clean",
			change:  artifactChange{Name: "skills/foo", ContentChanged: false, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: "1.0.0"},
			violate: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkVersions([]artifactChange{tt.change})
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

func TestCheckVersionsPreservesOrderAndReportsAll(t *testing.T) {
	changes := []artifactChange{
		{Name: "a", ContentChanged: true, ExistedInBase: true, BaseVersion: "1.0.0", HeadVersion: "1.0.0"},
		{Name: "b", ContentChanged: true, ExistedInBase: true, BaseVersion: "2.0.0", HeadVersion: "2.1.0"}, // bumped, clean
		{Name: "c", ContentChanged: true, ExistedInBase: true, BaseVersion: "3.0.0", HeadVersion: "3.0.0"},
	}
	got := checkVersions(changes)
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

func TestCheckVersionsEmpty(t *testing.T) {
	if got := checkVersions(nil); got != nil {
		t.Errorf("checkVersions(nil) = %v, want nil", got)
	}
}

// TestReconcileRecipeChange covers the pure reconciler that maps a recipe's
// before/after bytes into an artifactChange. A recipe is a FLAT single file where
// the manifest IS the whole content, so content-changed is manifestContentChanged
// over the two revisions — there is no sibling-vs-manifest split. checkVersions then
// judges it by the same rule as an artifact directory.
func TestReconcileRecipeChange(t *testing.T) {
	const path = "recipes/demo.yaml"
	tests := []struct {
		name          string
		base          string
		head          string
		existedInBase bool
		wantViolation bool
	}{
		{
			name:          "content change without a bump violates",
			base:          "name: demo\nversion: 1.0.0\nrole: tools\n",
			head:          "name: demo\nversion: 1.0.0\nrole: memory\n",
			existedInBase: true,
			wantViolation: true,
		},
		{
			name:          "content change with a bump passes",
			base:          "name: demo\nversion: 1.0.0\nrole: tools\n",
			head:          "name: demo\nversion: 1.1.0\nrole: memory\n",
			existedInBase: true,
			wantViolation: false,
		},
		{
			name:          "version-only edit passes",
			base:          "name: demo\nversion: 1.0.0\nrole: tools\n",
			head:          "name: demo\nversion: 1.1.0\nrole: tools\n",
			existedInBase: true,
			wantViolation: false,
		},
		{
			name:          "new recipe has no base to compare",
			base:          "",
			head:          "name: demo\nversion: 1.0.0\nrole: tools\n",
			existedInBase: false,
			wantViolation: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := reconcileRecipeChange(path, []byte(tt.base), tt.existedInBase, []byte(tt.head))
			got := checkVersions([]artifactChange{c})
			if tt.wantViolation && (len(got) != 1 || got[0].Name != path) {
				t.Fatalf("want a violation on %q, got %v", path, got)
			}
			if !tt.wantViolation && len(got) != 0 {
				t.Fatalf("want no violation, got %v", got)
			}
		})
	}
}

func TestManifestContentChanged(t *testing.T) {
	tests := []struct {
		name string
		base string
		head string
		want bool
	}{
		{
			name: "version-only edit is not content",
			base: "name: demo\nversion: 1.0.0\nrole: context\n",
			head: "name: demo\nversion: 1.1.0\nrole: context\n",
			want: false,
		},
		{
			name: "identical is not a change",
			base: "name: demo\nversion: 1.0.0\nrole: context\n",
			head: "name: demo\nversion: 1.0.0\nrole: context\n",
			want: false,
		},
		{
			name: "description edit is content",
			base: "name: demo\nversion: 1.0.0\ndescription: old\n",
			head: "name: demo\nversion: 1.0.0\ndescription: new\n",
			want: true,
		},
		{
			name: "requires edit is content",
			base: "name: demo\nversion: 1.0.0\nrequires: []\n",
			head: "name: demo\nversion: 1.0.0\nrequires: [serena]\n",
			want: true,
		},
		{
			name: "role edit is content",
			base: "name: demo\nversion: 1.0.0\nrole: context\n",
			head: "name: demo\nversion: 1.0.0\nrole: capability\n",
			want: true,
		},
		{
			name: "version line missing on one side, rest equal, is not content",
			base: "name: demo\nrole: context\n",
			head: "name: demo\nversion: 1.0.0\nrole: context\n",
			want: false,
		},
		{
			name: "content edit alongside a version bump is still content",
			base: "name: demo\nversion: 1.0.0\ndescription: old\n",
			head: "name: demo\nversion: 1.1.0\ndescription: new\n",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := manifestContentChanged([]byte(tt.base), []byte(tt.head)); got != tt.want {
				t.Errorf("manifestContentChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}
