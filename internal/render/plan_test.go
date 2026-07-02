package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// TestPlanRendersPluginExecRows proves plugin actions now surface in the standard
// summary table as EXEC rows (install command lines / honest skip lines), rather
// than the deleted mode-based PluginContributions block.
func TestPlanRendersPluginExecRows(t *testing.T) {
	cs := &diff.ChangeSet{DryRun: true, Diffs: []diff.FileDiff{
		{Action: diff.Exec, Type: "plugin", Tool: "claude", Scope: "user",
			Artifact: "superpowers", Path: "claude plugin install superpowers@claude-plugins-official --scope user",
			Exec: &diff.ExecSpec{Display: "claude plugin install superpowers@claude-plugins-official --scope user", Advisory: false}},
		{Action: diff.Exec, Type: "plugin", Tool: "opencode", Scope: "user",
			Artifact: "superpowers", Path: "skipped: opencode has no plugin system",
			Exec: &diff.ExecSpec{Display: "skipped: opencode has no plugin system", Advisory: true}},
	}}
	var b bytes.Buffer
	PrintPlan(&b, cs, testResolver(), false)
	out := b.String()
	if !strings.Contains(out, "superpowers") || !strings.Contains(out, "plugin") {
		t.Errorf("expected plugin rows in plan, got:\n%s", out)
	}
}

func testResolver() toolpath.Resolver {
	env := func(k string) (string, bool) {
		if k == "HOME" {
			return "/home/u", true
		}
		return "", false
	}
	return toolpath.New(env, "/home/u", "/proj")
}

func sampleChangeSet() *diff.ChangeSet {
	return &diff.ChangeSet{
		DryRun: true,
		Diffs: []diff.FileDiff{
			{
				Path: "/home/u/.claude/skills/team-research/SKILL.md", Action: diff.Create,
				After: []byte("body"), Artifact: "team-research", Type: "skill",
				Tool: "claude", Scope: "global", Role: "capability",
			},
			{
				Path: "/home/u/.claude/CLAUDE.md", Action: diff.Append,
				Before: []byte("old\n"), After: []byte("old\n\n<!-- patronus:start ap -->\nx\n<!-- patronus:end ap -->\n"),
				Artifact: "agent-principles", Type: "instruction",
				Tool: "claude", Scope: "global", Role: "instruction", Note: "patronus section: ap",
			},
		},
	}
}

func TestSummaryTableContents(t *testing.T) {
	var buf bytes.Buffer
	PrintSummaryTable(&buf, sampleChangeSet(), testResolver())
	out := buf.String()

	for _, want := range []string{
		"Artifact", "Impacted path(s)", "Operation", "Type", "Role", "Tool", "Scope",
		"team-research", "~/.claude/skills/team-research/SKILL.md", "CREATE", "skill", "capability",
		"agent-principles", "~/.claude/CLAUDE.md", "APPEND", "instruction",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q\n%s", want, out)
		}
	}
	// Home collapsed, no absolute /home/u leak.
	if strings.Contains(out, "/home/u/") {
		t.Errorf("home not collapsed:\n%s", out)
	}
}

func TestSummaryTableGroupsManyFiles(t *testing.T) {
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{
		{Path: "/home/u/.claude/skills/p/SKILL.md", Action: diff.Create, Artifact: "p", Type: "skill", Role: "context", Tool: "claude", Scope: "global"},
		{Path: "/home/u/.claude/skills/p/patterns/a.md", Action: diff.Create, Artifact: "p", Type: "skill", Role: "context", Tool: "claude", Scope: "global"},
		{Path: "/home/u/.claude/skills/p/patterns/b.md", Action: diff.Create, Artifact: "p", Type: "skill", Role: "context", Tool: "claude", Scope: "global"},
	}}
	var buf bytes.Buffer
	PrintSummaryTable(&buf, cs, testResolver())
	out := buf.String()
	// 3 files in one (artifact, op, type, role, tool, scope) group -> compacted.
	if !strings.Contains(out, "files)") {
		t.Errorf("expected file-count compaction:\n%s", out)
	}
}

func TestPlanSectionOrder(t *testing.T) {
	// Order must be: summary table → tree → (verbose diffs). Use index-of to
	// assert relative positions so the layout can't silently regress.
	var buf bytes.Buffer
	PrintPlan(&buf, sampleChangeSet(), testResolver(), true)
	out := buf.String()

	table := strings.Index(out, "Type") // table header
	tree := strings.Index(out, "└──")   // tree glyph
	diffs := strings.Index(out, "@@")   // unified diff hunk
	if table < 0 || tree < 0 || diffs < 0 {
		t.Fatalf("missing a section (table=%d tree=%d diffs=%d):\n%s", table, tree, diffs, out)
	}
	if !(table < tree && tree < diffs) {
		t.Errorf("wrong order: table=%d tree=%d diffs=%d (want table<tree<diffs)\n%s", table, tree, diffs, out)
	}
}

func TestDefaultPlanShowsTreeAndSummary(t *testing.T) {
	var buf bytes.Buffer
	PrintPlan(&buf, sampleChangeSet(), testResolver(), false)
	out := buf.String()
	// Tree is part of the default output...
	if !strings.Contains(out, "(new)") || !(strings.Contains(out, "├──") || strings.Contains(out, "└──")) {
		t.Errorf("default plan missing tree:\n%s", out)
	}
	// ...alongside the summary table.
	if !strings.Contains(out, "Type") {
		t.Errorf("default plan missing summary table:\n%s", out)
	}
	// But NO unified-diff hunks in non-verbose mode.
	if strings.Contains(out, "@@") {
		t.Errorf("default plan should not contain unified diff hunks:\n%s", out)
	}
}

func TestVerbosePlanShowsUnifiedDiff(t *testing.T) {
	var buf bytes.Buffer
	PrintPlan(&buf, sampleChangeSet(), testResolver(), true)
	out := buf.String()

	// Tree is present in verbose mode too.
	if !(strings.Contains(out, "├──") || strings.Contains(out, "└──")) {
		t.Errorf("verbose plan missing tree:\n%s", out)
	}
	// Per-artifact headers.
	if !strings.Contains(out, "── team-research ──") || !strings.Contains(out, "── agent-principles ──") {
		t.Errorf("missing artifact section headers:\n%s", out)
	}
	// Unified diff markers for the APPEND (content changed).
	if !strings.Contains(out, "@@") {
		t.Errorf("expected unified diff hunk markers:\n%s", out)
	}
	// Summary table still present below.
	if !strings.Contains(out, "Type") {
		t.Errorf("summary table missing in verbose mode:\n%s", out)
	}
	// Footer tally + dry-run note.
	if !strings.Contains(out, "Plan:") || !strings.Contains(out, "dry run") {
		t.Errorf("missing footer:\n%s", out)
	}
}

func TestPlanEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintPlan(&buf, &diff.ChangeSet{DryRun: true}, testResolver(), false)
	if !strings.Contains(buf.String(), "No changes") {
		t.Errorf("expected empty message, got %q", buf.String())
	}
}

func TestPlanAllSkipIsEmptyForVisible(t *testing.T) {
	cs := &diff.ChangeSet{DryRun: true, Diffs: []diff.FileDiff{
		{Path: "/home/u/.claude/x", Action: diff.Skip, Artifact: "a", Tool: "claude", Scope: "global"},
	}}
	var buf bytes.Buffer
	PrintPlan(&buf, cs, testResolver(), false)
	// A SKIP is still a visible (non-dir) row, so the summary table prints it.
	if !strings.Contains(buf.String(), "SKIP") {
		t.Errorf("expected SKIP row:\n%s", buf.String())
	}
}

func TestChangeTreeRendersHierarchy(t *testing.T) {
	var buf bytes.Buffer
	PrintChangeTree(&buf, sampleChangeSet(), testResolver())
	out := buf.String()
	if !strings.Contains(out, "~") || !strings.Contains(out, "└──") && !strings.Contains(out, "├──") {
		t.Errorf("expected tree glyphs:\n%s", out)
	}
	if !strings.Contains(out, "(new)") || !strings.Contains(out, "# CREATE — role: capability") {
		t.Errorf("expected leaf annotations:\n%s", out)
	}
}

func TestSummarizePaths(t *testing.T) {
	if got := summarizePaths(nil); got != "-" {
		t.Errorf("empty = %q, want -", got)
	}
	if got := summarizePaths([]string{"a/b.md"}); got != "a/b.md" {
		t.Errorf("single = %q", got)
	}
	if got := summarizePaths([]string{"a/b.md", "a/c.md"}); got != "a/ (2 files)" {
		t.Errorf("grouped = %q, want 'a/ (2 files)'", got)
	}
	// Disjoint roots -> "first (+N more)".
	if got := summarizePaths([]string{"x/a.md", "y/b.md"}); !strings.Contains(got, "more") {
		t.Errorf("disjoint = %q", got)
	}
}

func TestChangeTreeProjectRelativeRoot(t *testing.T) {
	// A path with no ~ or absolute prefix still renders.
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{
		{Path: "AGENTS.md", Action: diff.Append, Artifact: "ap", Tool: "codex", Scope: "local", Role: "instruction"},
	}}
	var buf bytes.Buffer
	PrintChangeTree(&buf, cs, testResolver())
	if !strings.Contains(buf.String(), "AGENTS.md") {
		t.Errorf("project-relative path not rendered:\n%s", buf.String())
	}
}

func TestCommonDir(t *testing.T) {
	got := commonDir([]string{"a/b/c.md", "a/b/d.md", "a/b/e/f.md"})
	if got != "a/b" {
		t.Errorf("commonDir = %q, want a/b", got)
	}
	if got := commonDir([]string{"a/x.md", "b/y.md"}); got != "" {
		t.Errorf("disjoint commonDir = %q, want empty", got)
	}
}

func TestSplitSegments(t *testing.T) {
	cases := map[string][]string{
		"~/.claude/x": {"~", ".claude", "x"},
		"/etc/hosts":  {"/etc", "hosts"},
		"rel/path.md": {"rel", "path.md"},
		"":            nil,
	}
	for in, want := range cases {
		got := splitSegments(in)
		if len(got) != len(want) {
			t.Errorf("splitSegments(%q) = %v, want %v", in, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("splitSegments(%q)[%d] = %q, want %q", in, i, got[i], want[i])
			}
		}
	}
}

func TestAnnotationAllActions(t *testing.T) {
	cases := map[diff.Action]string{
		diff.Create:   "(new)",
		diff.Append:   "(appended)",
		diff.Merge:    "(modified)",
		diff.Skip:     "(skip)",
		diff.Conflict: "(conflict!)",
	}
	for a, want := range cases {
		if got := annotation(a); got != want {
			t.Errorf("annotation(%s) = %q, want %q", a, got, want)
		}
	}
}

func TestSummaryConflictAndMergeRows(t *testing.T) {
	cs := &diff.ChangeSet{DryRun: true, Diffs: []diff.FileDiff{
		{Path: "/proj/.mcp.json", Action: diff.Merge, Artifact: "memrec", Type: "wire-only", Role: "tools", Tool: "claude", Scope: "local", Before: []byte("{}"), After: []byte(`{"mcpServers":{}}`)},
		{Path: "/proj/x.md", Action: diff.Conflict, Artifact: "c", Type: "skill", Role: "capability", Tool: "claude", Scope: "local", Before: []byte("a"), After: []byte("b")},
	}}
	var buf bytes.Buffer
	PrintPlan(&buf, cs, testResolver(), false)
	out := buf.String()
	if !strings.Contains(out, "MERGE") || !strings.Contains(out, "CONFLICT") {
		t.Errorf("expected MERGE and CONFLICT rows:\n%s", out)
	}
	if !strings.Contains(out, "(modified)") || !strings.Contains(out, "(conflict!)") {
		t.Errorf("expected tree annotations:\n%s", out)
	}
}

func TestUnifiedMethod(t *testing.T) {
	d := diff.FileDiff{Before: []byte("a\nb\n"), After: []byte("a\nc\n"), Action: diff.Append}
	u := d.Unified()
	if !strings.Contains(u, "@@") || !strings.Contains(u, "-b") || !strings.Contains(u, "+c") {
		t.Errorf("unified diff unexpected:\n%s", u)
	}
	// SKIP yields no diff.
	skip := diff.FileDiff{Before: []byte("a"), After: []byte("a"), Action: diff.Skip}
	if skip.Unified() != "" {
		t.Errorf("skip should yield empty unified diff")
	}
}
