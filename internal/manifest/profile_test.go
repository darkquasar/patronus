package manifest

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestLoadProfileTargetsAreOwnDocumentOnly pins the optional profile `targets:`
// field: it decodes from the profile's own bytes and is never inherited through
// `extends:` (a child that extends a declaring parent without its own `targets:`
// is not a declaring profile). All names are invented (ADR-0002).
func TestLoadProfileTargetsAreOwnDocumentOnly(t *testing.T) {
	cases := []struct {
		name string
		body string
		want StringList
	}{
		{
			name: "declared",
			body: "apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: toy-lane\nversion: 1.0.0\ntargets: [toy]\nlayers:\n  capabilities: [sample]\n",
			want: StringList{"toy"},
		},
		{
			name: "declared-scalar",
			body: "apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: toy-lane\nversion: 1.0.0\ntargets: toy\nlayers:\n  capabilities: [sample]\n",
			want: StringList{"toy"},
		},
		{
			name: "undeclared",
			body: "apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: toy-base\nversion: 1.0.0\nlayers:\n  capabilities: [sample]\n",
			want: nil,
		},
		{
			name: "extends-declaring-parent-without-own-targets",
			body: "apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: toy-child\nversion: 1.0.0\nextends: toy-lane\nlayers:\n  capabilities: [extra]\n",
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tc.name+".yaml")
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			p, err := LoadProfile(path)
			if err != nil {
				t.Fatalf("LoadProfile: %v", err)
			}
			if !reflect.DeepEqual(p.Targets, tc.want) {
				t.Fatalf("Targets = %#v, want %#v", p.Targets, tc.want)
			}
		})
	}
}
