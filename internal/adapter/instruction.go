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
