package render

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/plugin"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// RenderPluginContributions prints the per-target disposition for one plugin so
// the dry-run states the trust decision (native/verbatim vs translate/flagged vs
// unsupported/skipped) before deploy.
func RenderPluginContributions(w io.Writer, name string, contribs []plugin.Contribution) {
	fmt.Fprintf(w, "plugin %s\n", name)
	for _, c := range contribs {
		switch c.Mode {
		case plugin.ModeNative:
			fmt.Fprintf(w, "  %-8s → native    (%s)   verbatim\n", c.Tool, c.Ecosystem)
		case plugin.ModeTranslate:
			fmt.Fprintf(w, "  %-8s → translate (from %s)   ⚠ not verbatim\n", c.Tool, c.Ecosystem)
		case plugin.ModeUnsupported:
			fmt.Fprintf(w, "  %-8s → unsupported (no plugin construct)   skipped\n", c.Tool)
		}
	}
}

// PrintPlan renders a dry-run change set in a fixed order: the artifact-centric
// summary table first, then the ASCII file tree, then (only with --verbose) the
// per-artifact unified diffs, and finally a footer tally.
func PrintPlan(w io.Writer, cs *diff.ChangeSet, r toolpath.Resolver, verbose bool) {
	if len(visibleDiffs(cs)) == 0 {
		fmt.Fprintln(w, "No changes — everything is already up to date.")
		return
	}
	PrintSummaryTable(w, cs, r)
	fmt.Fprintln(w)
	PrintChangeTree(w, cs, r)
	if verbose {
		fmt.Fprintln(w)
		printVerboseDiffs(w, cs, r)
	}
	printPlanFooter(w, cs)
}

// PrintSummaryTable renders the artifact-centric summary:
//
//	Artifact | Impacted path(s) | Operation | Type | Role | Tool | Scope
//
// Type and Role are the two ontology axes, one per column (no mixed-axis
// "Capability" column): Type is the item's shape (artifact type or recipe
// Shape()), Role is the layer it fills. Rows are grouped per
// (artifact, operation, type, role, tool, scope); the impacted paths for that
// group are listed compactly (a directory + count when many files share a root).
func PrintSummaryTable(w io.Writer, cs *diff.ChangeSet, r toolpath.Resolver) {
	type key struct{ artifact, op, typ, role, tool, scope, uniq string }
	type group struct {
		key   key
		paths []string
	}
	var order []key
	groups := map[key]*group{}

	for _, d := range cs.Diffs {
		if d.IsDir {
			continue
		}
		k := key{d.Artifact, string(d.Action), d.Type, d.Role, d.Tool, d.Scope, ""}
		// EXEC rows are distinct commands, not files sharing a directory — keep
		// each on its own row instead of collapsing them via summarizePaths.
		if d.Action == diff.Exec {
			k.uniq = d.Path
		}
		g, ok := groups[k]
		if !ok {
			g = &group{key: k}
			groups[k] = g
			order = append(order, k)
		}
		g.paths = append(g.paths, displayPath(r, d.Path))

		// Sections from other artifacts folded into this composed APPEND file each
		// get their own row, so the table shows every contributor (not just the
		// first) writing to the shared file.
		for _, c := range d.Contrib {
			ck := key{c.Artifact, string(diff.Append), d.Type, d.Role, d.Tool, d.Scope, ""}
			cg, ok := groups[ck]
			if !ok {
				cg = &group{key: ck}
				groups[ck] = cg
				order = append(order, ck)
			}
			cg.paths = append(cg.paths, displayPath(r, d.Path))
		}
	}

	headers := []string{"Artifact", "Impacted path(s)", "Operation", "Type", "Role", "Tool", "Scope"}
	var rows [][]string
	for _, k := range order {
		g := groups[k]
		rows = append(rows, []string{
			orDash(k.artifact),
			summarizePaths(g.paths),
			k.op,
			orDash(k.typ),
			orDash(k.role),
			orDash(k.tool),
			orDash(k.scope),
		})
	}
	printBox(w, headers, rows)
}

// printVerboseDiffs prints a unified diff for each non-skip diff, grouped by
// artifact, above the summary table.
func printVerboseDiffs(w io.Writer, cs *diff.ChangeSet, r toolpath.Resolver) {
	current := ""
	for _, d := range cs.Diffs {
		if d.IsDir || d.Action == diff.Skip {
			continue
		}
		if d.Artifact != current {
			current = d.Artifact
			fmt.Fprintf(w, "── %s ──\n", orDash(current))
		}
		fmt.Fprintf(w, "%s  %s  [%s %s]\n", string(d.Action), displayPath(r, d.Path), d.Tool, d.Scope)
		u := d.Unified()
		if u == "" {
			fmt.Fprintln(w)
			continue
		}
		for _, line := range strings.Split(strings.TrimRight(u, "\n"), "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
		fmt.Fprintln(w)
	}
}

// printPlanFooter prints the action tally and the dry-run reminder.
func printPlanFooter(w io.Writer, cs *diff.ChangeSet) {
	c := cs.Counts()
	parts := []string{}
	for _, a := range []diff.Action{diff.Create, diff.Append, diff.Merge, diff.Fetch, diff.Exec, diff.Delete, diff.Unappend, diff.Restore, diff.Conflict, diff.Skip} {
		if c[a] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c[a], a))
		}
	}
	fmt.Fprintf(w, "\nPlan: %s\n", strings.Join(parts, ", "))
	if cs.DryRun {
		fmt.Fprintln(w, "(dry run — no files were written)")
	}
}

// PrintChangeTree renders the change set as an ASCII file tree grouped by root,
// with per-leaf action annotations and a trailing "# ACTION — role: x" comment.
// Retained as an alternate view (e.g. for --json consumers or debugging).
func PrintChangeTree(w io.Writer, cs *diff.ChangeSet, r toolpath.Resolver) {
	if len(visibleDiffs(cs)) == 0 {
		fmt.Fprintln(w, "(no changes)")
		return
	}
	root := newTreeNode()
	for _, d := range cs.Diffs {
		if d.IsDir || d.Action == diff.Exec {
			continue // EXEC is a command, not a file path — shown in the table only
		}
		root.insert(splitSegments(displayPath(r, d.Path)), d)
	}
	root.render(w)
}

// --- helpers -----------------------------------------------------------------

func visibleDiffs(cs *diff.ChangeSet) []diff.FileDiff {
	var out []diff.FileDiff
	for _, d := range cs.Diffs {
		if !d.IsDir {
			out = append(out, d)
		}
	}
	return out
}

func displayPath(r toolpath.Resolver, p string) string { return r.CollapseHome(p) }

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// summarizePaths compactly describes a set of target paths: a single path as-is;
// many files sharing a common directory as "<dir>/ (+N files)".
func summarizePaths(paths []string) string {
	switch len(paths) {
	case 0:
		return "-"
	case 1:
		return paths[0]
	}
	dir := commonDir(paths)
	if dir != "" {
		return fmt.Sprintf("%s/ (%d files)", dir, len(paths))
	}
	return fmt.Sprintf("%s (+%d more)", paths[0], len(paths)-1)
}

// commonDir returns the longest common slash-delimited directory prefix of the
// given paths, or "" if they do not share one.
func commonDir(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	split := func(p string) []string { return strings.Split(strings.ReplaceAll(p, "\\", "/"), "/") }
	prefix := split(paths[0])
	prefix = prefix[:len(prefix)-1] // drop filename
	for _, p := range paths[1:] {
		segs := split(p)
		segs = segs[:len(segs)-1]
		prefix = commonPrefix(prefix, segs)
		if len(prefix) == 0 {
			return ""
		}
	}
	return strings.Join(prefix, "/")
}

func commonPrefix(a, b []string) []string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}

// printBox draws a Unicode box table with the given headers and rows.
func printBox(w io.Writer, headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Fprintln(w, "(no changes)")
		return
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = runeLen(h)
	}
	for _, row := range rows {
		for i, c := range row {
			if l := runeLen(c); l > widths[i] {
				widths[i] = l
			}
		}
	}

	line := func(left, mid, right string) {
		var b strings.Builder
		b.WriteString(left)
		for i, wd := range widths {
			b.WriteString(strings.Repeat("─", wd+2))
			if i < len(widths)-1 {
				b.WriteString(mid)
			}
		}
		b.WriteString(right)
		fmt.Fprintln(w, b.String())
	}
	rowOut := func(cells []string) {
		var b strings.Builder
		b.WriteString("│")
		for i, c := range cells {
			pad := widths[i] - runeLen(c)
			fmt.Fprintf(&b, " %s%s ", c, strings.Repeat(" ", pad))
			b.WriteString("│")
		}
		fmt.Fprintln(w, b.String())
	}

	line("┌", "┬", "┐")
	rowOut(headers)
	line("├", "┼", "┤")
	for _, row := range rows {
		rowOut(row)
	}
	line("└", "┴", "┘")
}

func runeLen(s string) int { return len([]rune(s)) }

// --- tree construction -------------------------------------------------------

type treeNode struct {
	children map[string]*treeNode
	order    []string
	leaf     *diff.FileDiff
}

func newTreeNode() *treeNode { return &treeNode{children: map[string]*treeNode{}} }

func (n *treeNode) child(name string) *treeNode {
	c, ok := n.children[name]
	if !ok {
		c = newTreeNode()
		n.children[name] = c
		n.order = append(n.order, name)
	}
	return c
}

func (n *treeNode) insert(segs []string, d diff.FileDiff) {
	cur := n
	for i, s := range segs {
		cur = cur.child(s)
		if i == len(segs)-1 {
			dd := d
			cur.leaf = &dd
		}
	}
}

// render prints top-level roots without branch glyphs; descendants use ├──/└──.
func (n *treeNode) render(w io.Writer) {
	names := append([]string(nil), n.order...)
	sort.Strings(names)
	for _, name := range names {
		c := n.children[name]
		fmt.Fprintln(w, name+leafSuffix(c))
		c.renderChildren(w, "")
	}
}

func (n *treeNode) renderChildren(w io.Writer, prefix string) {
	names := append([]string(nil), n.order...)
	sort.Strings(names)
	for i, name := range names {
		c := n.children[name]
		last := i == len(names)-1
		branch, nextPrefix := "├── ", prefix+"│   "
		if last {
			branch, nextPrefix = "└── ", prefix+"    "
		}
		fmt.Fprintln(w, prefix+branch+name+leafSuffix(c))
		c.renderChildren(w, nextPrefix)
	}
}

func leafSuffix(n *treeNode) string {
	if n.leaf == nil {
		return "/"
	}
	d := n.leaf
	if d.Role != "" {
		return fmt.Sprintf("  %s  # %s — role: %s", annotation(d.Action), d.Action, d.Role)
	}
	return fmt.Sprintf("  %s  # %s", annotation(d.Action), d.Action)
}

func annotation(a diff.Action) string {
	switch a {
	case diff.Create:
		return "(new)"
	case diff.Append:
		return "(appended)"
	case diff.Merge:
		return "(modified)"
	case diff.Skip:
		return "(skip)"
	case diff.Fetch:
		return "(download)"
	case diff.Exec:
		return "(run)"
	case diff.Conflict:
		return "(conflict!)"
	case diff.Delete:
		return "(removed)"
	case diff.Unappend:
		return "(un-appended)"
	case diff.Restore:
		return "(restored)"
	default:
		return ""
	}
}

// splitSegments splits a display path into segments, preserving a leading ~ or /
// marker as the first segment so roots render naturally.
func splitSegments(p string) []string {
	if p == "" {
		return nil
	}
	p = strings.ReplaceAll(p, "\\", "/")
	if strings.HasPrefix(p, "~/") {
		rest := strings.Split(strings.TrimPrefix(p, "~/"), "/")
		return append([]string{"~"}, rest...)
	}
	if strings.HasPrefix(p, "/") {
		rest := strings.Split(strings.TrimPrefix(p, "/"), "/")
		return append([]string{"/" + rest[0]}, rest[1:]...)
	}
	return strings.Split(p, "/")
}
