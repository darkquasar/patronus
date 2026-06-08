package archive

import (
	"bytes"
	"testing"
)

func TestCreateTarGzRoundTrip(t *testing.T) {
	in := map[string][]byte{
		"patronus.yaml":           []byte("kind: Skill\nname: x\n"),
		"SKILL.md":                []byte("# body"),
		"patterns/pattern-001.md": []byte("pat one"),
	}
	tgz, err := CreateTarGz(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Extract(bytes.NewReader(tgz), FormatTarGz)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(in) {
		t.Fatalf("got %d members, want %d", len(out), len(in))
	}
	for name, want := range in {
		if !bytes.Equal(out[name], want) {
			t.Errorf("%s: got %q, want %q", name, out[name], want)
		}
	}
}

func TestCreateTarGzDeterministic(t *testing.T) {
	in := map[string][]byte{
		"b.md": []byte("two"),
		"a.md": []byte("one"),
		"c.md": []byte("three"),
	}
	a, err := CreateTarGz(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := CreateTarGz(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("CreateTarGz output is not deterministic")
	}
}
