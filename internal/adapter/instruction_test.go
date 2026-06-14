package adapter

import (
	"bytes"
	"strings"
	"testing"
)

func TestAppendSectionNewFile(t *testing.T) {
	got := AppendSection(nil, "agent-principles", []byte("hello world"))
	want := "<!-- patronus:start agent-principles -->\nhello world\n<!-- patronus:end agent-principles -->\n"
	if string(got) != want {
		t.Errorf("new file:\n got %q\nwant %q", got, want)
	}
}

func TestAppendSectionEmptyExisting(t *testing.T) {
	// Whitespace-only existing is treated as empty.
	got := AppendSection([]byte("  \n\n"), "x", []byte("body"))
	if !strings.HasPrefix(string(got), "<!-- patronus:start x -->") {
		t.Errorf("whitespace existing not treated as empty: %q", got)
	}
}

func TestAppendSectionAppendsPreservingProse(t *testing.T) {
	existing := []byte("# My Notes\n\nsome prose here\n")
	got := AppendSection(existing, "x", []byte("injected"))
	s := string(got)
	if !strings.Contains(s, "# My Notes") || !strings.Contains(s, "some prose here") {
		t.Errorf("surrounding prose lost: %q", s)
	}
	if !strings.Contains(s, "<!-- patronus:start x -->\ninjected\n<!-- patronus:end x -->") {
		t.Errorf("block not appended: %q", s)
	}
	// A separating blank line between prose and the block.
	if !strings.Contains(s, "some prose here\n\n<!-- patronus:start x -->") {
		t.Errorf("missing separating blank line: %q", s)
	}
}

func TestAppendSectionAppendsWhenNoTrailingNewline(t *testing.T) {
	existing := []byte("no newline at end")
	got := AppendSection(existing, "x", []byte("b"))
	if !strings.HasPrefix(string(got), "no newline at end\n\n<!-- patronus:start x -->") {
		t.Errorf("newline normalization failed: %q", got)
	}
}

func TestAppendSectionReplacesInPlace(t *testing.T) {
	existing := []byte("before\n<!-- patronus:start x -->\nOLD\n<!-- patronus:end x -->\nafter\n")
	got := AppendSection(existing, "x", []byte("NEW"))
	s := string(got)
	if strings.Contains(s, "OLD") {
		t.Errorf("old content not replaced: %q", s)
	}
	if !strings.Contains(s, "before\n") || !strings.Contains(s, "\nafter\n") {
		t.Errorf("surrounding prose not preserved: %q", s)
	}
	if !strings.Contains(s, "<!-- patronus:start x -->\nNEW\n<!-- patronus:end x -->") {
		t.Errorf("replacement block wrong: %q", s)
	}
	// Exactly one occurrence of the start marker (no duplication).
	if n := strings.Count(s, "<!-- patronus:start x -->"); n != 1 {
		t.Errorf("expected 1 start marker, got %d", n)
	}
}

func TestAppendSectionIdempotent(t *testing.T) {
	body := []byte("stable body")
	once := AppendSection([]byte("intro\n"), "x", body)
	twice := AppendSection(once, "x", body)
	if !bytes.Equal(once, twice) {
		t.Errorf("not idempotent:\n once: %q\ntwice: %q", once, twice)
	}
}

func TestAppendSectionMalformedStartWithoutEnd(t *testing.T) {
	// A start marker with no matching end: append a fresh well-formed block.
	existing := []byte("text\n<!-- patronus:start x -->\ndangling\n")
	got := AppendSection(existing, "x", []byte("new"))
	if !strings.Contains(string(got), "<!-- patronus:end x -->") {
		t.Errorf("expected a well-formed block appended: %q", got)
	}
}

func TestAppendSectionBodyTrailingNewlinesNormalized(t *testing.T) {
	got := AppendSection(nil, "x", []byte("body\n\n\n"))
	want := "<!-- patronus:start x -->\nbody\n<!-- patronus:end x -->\n"
	if string(got) != want {
		t.Errorf("trailing newlines not normalized:\n got %q\nwant %q", got, want)
	}
}

func TestRemoveSectionRoundTripsAppend(t *testing.T) {
	// Appending then removing must restore the original bytes exactly.
	original := []byte("# My Notes\n\nsome prose here\n")
	appended := AppendSection(original, "x", []byte("injected"))
	got, found := RemoveSection(appended, "x")
	if !found {
		t.Fatal("RemoveSection did not find the appended block")
	}
	if !bytes.Equal(got, original) {
		t.Errorf("append-then-remove not a round-trip:\n got %q\nwant %q", got, original)
	}
}

func TestRemoveSectionPreservesSurroundingProse(t *testing.T) {
	existing := []byte("before\n\n<!-- patronus:start x -->\nBODY\n<!-- patronus:end x -->\n\nafter\n")
	got, found := RemoveSection(existing, "x")
	if !found {
		t.Fatal("expected to find block")
	}
	s := string(got)
	if strings.Contains(s, "BODY") || strings.Contains(s, "patronus:start") {
		t.Errorf("block not fully removed: %q", s)
	}
	if !strings.Contains(s, "before") || !strings.Contains(s, "after") {
		t.Errorf("surrounding prose lost: %q", s)
	}
}

func TestRemoveSectionAbsentIsNoOp(t *testing.T) {
	existing := []byte("# Notes\n\njust prose\n")
	got, found := RemoveSection(existing, "missing")
	if found {
		t.Error("found a section that does not exist")
	}
	if !bytes.Equal(got, existing) {
		t.Errorf("absent removal mutated content:\n got %q\nwant %q", got, existing)
	}
}

func TestRemoveSectionMalformedStartWithoutEnd(t *testing.T) {
	existing := []byte("text\n<!-- patronus:start x -->\ndangling\n")
	got, found := RemoveSection(existing, "x")
	if found {
		t.Error("malformed block should report not found")
	}
	if !bytes.Equal(got, existing) {
		t.Errorf("malformed removal mutated content: %q", got)
	}
}

func TestRemoveSectionOnlyContentYieldsEmpty(t *testing.T) {
	only := AppendSection(nil, "x", []byte("body"))
	got, found := RemoveSection(only, "x")
	if !found {
		t.Fatal("expected to find the only block")
	}
	if len(bytes.TrimSpace(got)) != 0 {
		t.Errorf("expected empty result, got %q", got)
	}
}

func TestSectionBody(t *testing.T) {
	existing := AppendSection([]byte("intro\n"), "x", []byte("the body"))
	body, found := SectionBody(existing, "x")
	if !found {
		t.Fatal("section not found")
	}
	if string(body) != "the body" {
		t.Errorf("got body %q, want %q", body, "the body")
	}
	if _, found := SectionBody(existing, "nope"); found {
		t.Error("found a nonexistent section body")
	}
}
