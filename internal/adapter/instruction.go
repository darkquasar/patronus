package adapter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
)

// transformInstruction produces an APPEND diff: the artifact body is written
// into the tool's instructions file (CLAUDE.md / AGENTS.md) inside an
// idempotent, fenced section keyed by the artifact name. Surrounding prose is
// never touched.
func (e *Engine) transformInstruction(art *manifest.Artifact, ad *manifest.Adapter, scope, srcDir string, readExisting ReadExisting) ([]diff.FileDiff, error) {
	if ad.Layout.Instruction == nil {
		return nil, fmt.Errorf("adapter %q: no Instruction layout", ad.Tool)
	}
	target := ad.Layout.Instruction.ForScope(scope)
	if !target.OK() {
		return nil, fmt.Errorf("adapter %q: Instruction has no %s target", ad.Tool, scope)
	}

	entry := art.Entry
	if entry == "" {
		return nil, fmt.Errorf("adapter: Instruction %q missing entry", art.Name)
	}
	body, err := os.ReadFile(filepath.Join(srcDir, entry))
	if err != nil {
		return nil, fmt.Errorf("adapter: read instruction entry: %w", err)
	}

	path := e.resolver.ResolveMarker(target.File, ad.Tool, scope)
	existing, _, err := readExisting(path)
	if err != nil {
		return nil, fmt.Errorf("adapter: read existing instruction file: %w", err)
	}

	after := AppendSection(existing, art.Name, body)
	return []diff.FileDiff{{
		Path:    path,
		Action:  diff.Append,
		Before:  existing,
		After:   after,
		Tool:    ad.Tool,
		Scope:   scope,
		Role:    string(art.Role),
		Note:    "patronus section: " + art.Name,
		Section: &diff.SectionEdit{Name: art.Name, Body: body},
	}}, nil
}

// sectionMarkers returns the start/end fence lines for a named patronus section.
func sectionMarkers(name string) (start, end string) {
	return "<!-- patronus:start " + name + " -->", "<!-- patronus:end " + name + " -->"
}

// AppendSection inserts or replaces ONLY the fenced block keyed by name within
// existing, preserving all surrounding prose. Behavior:
//   - existing empty: the file becomes exactly the fenced block.
//   - block present: its contents are replaced in place (markers kept).
//   - block absent: the block is appended after a separating blank line.
//
// The result is deterministic and idempotent: re-running with the same body
// yields identical bytes.
func AppendSection(existing []byte, name string, body []byte) []byte {
	start, end := sectionMarkers(name)
	block := buildBlock(start, end, body)

	if len(bytes.TrimSpace(existing)) == 0 {
		return []byte(block + "\n")
	}

	sIdx := bytes.Index(existing, []byte(start))
	if sIdx >= 0 {
		// Replace from the start marker through the end marker (inclusive).
		eIdx := bytes.Index(existing[sIdx:], []byte(end))
		if eIdx >= 0 {
			eIdx = sIdx + eIdx + len(end)
			var buf bytes.Buffer
			buf.Write(existing[:sIdx])
			buf.WriteString(block)
			buf.Write(existing[eIdx:])
			return buf.Bytes()
		}
		// Malformed (start without end): fall through and append a fresh block.
	}

	// Append after the existing content with a separating blank line.
	var buf bytes.Buffer
	buf.Write(existing)
	if !bytes.HasSuffix(existing, []byte("\n")) {
		buf.WriteByte('\n')
	}
	buf.WriteByte('\n')
	buf.WriteString(block)
	buf.WriteByte('\n')
	return buf.Bytes()
}

// RemoveSection is the inverse of AppendSection: it strips ONLY the fenced block
// keyed by name from existing, preserving all surrounding prose, and returns the
// result plus whether a block was found. It shares sectionMarkers with
// AppendSection so the two stay symmetric. Behavior:
//   - block present: the markers and everything between them are removed, along
//     with one separating blank line if one immediately precedes the block (the
//     mirror of the blank line AppendSection inserts), so an append-then-remove
//     round-trips to the original bytes.
//   - block absent (or malformed start-without-end): existing is returned
//     unchanged with found=false.
//   - removing the file's only content yields empty bytes (the caller decides
//     whether to delete the now-empty file).
func RemoveSection(existing []byte, name string) (result []byte, found bool) {
	start, end := sectionMarkers(name)
	sIdx := bytes.Index(existing, []byte(start))
	if sIdx < 0 {
		return existing, false
	}
	rel := bytes.Index(existing[sIdx:], []byte(end))
	if rel < 0 {
		return existing, false // malformed: start without end — leave untouched
	}
	eIdx := sIdx + rel + len(end)

	// Extend the removed range to swallow the separating blank line AppendSection
	// inserts before an appended block (a "\n\n" boundary), so the round-trip is
	// exact. Trim one trailing newline after the end marker too when the block was
	// appended (AppendSection writes the block then a final '\n').
	startCut := sIdx
	if startCut >= 2 && existing[startCut-1] == '\n' && existing[startCut-2] == '\n' {
		startCut-- // drop the blank-line separator that preceded the block
	}
	endCut := eIdx
	if endCut < len(existing) && existing[endCut] == '\n' {
		endCut++ // drop the newline AppendSection wrote after the block
	}

	var buf bytes.Buffer
	buf.Write(existing[:startCut])
	buf.Write(existing[endCut:])
	return buf.Bytes(), true
}

// SectionBody returns the body bytes currently inside the fenced block keyed by
// name (between the markers, trimmed of the surrounding newlines), and whether
// the block was found. Used for drift detection: comparing the on-disk section
// against what Patronus wrote tells remove whether the user edited inside it.
func SectionBody(existing []byte, name string) (body []byte, found bool) {
	start, end := sectionMarkers(name)
	sIdx := bytes.Index(existing, []byte(start))
	if sIdx < 0 {
		return nil, false
	}
	bodyStart := sIdx + len(start)
	rel := bytes.Index(existing[bodyStart:], []byte(end))
	if rel < 0 {
		return nil, false
	}
	inner := existing[bodyStart : bodyStart+rel]
	return bytes.Trim(inner, "\n"), true
}

// buildBlock renders the fenced block with the body between the markers.
func buildBlock(start, end string, body []byte) string {
	trimmed := bytes.TrimRight(body, "\n")
	var buf bytes.Buffer
	buf.WriteString(start)
	buf.WriteByte('\n')
	buf.Write(trimmed)
	buf.WriteByte('\n')
	buf.WriteString(end)
	return buf.String()
}
