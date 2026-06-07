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
