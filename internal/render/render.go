// Package render produces human-readable and JSON output for the catalog and
// scan inventory, keeping the cmd layer thin.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/scan"
)

// CatalogView selects which sections PrintCatalog emits, and whether profile
// layers are expanded.
type CatalogView struct {
	Artifacts bool
	Recipes   bool
	Profiles  bool
	Plugins   bool
	Layers    bool
	// Description switches the artifact section from the compact table (which omits
	// the description so a row fits the screen) to a block/list view where each
	// artifact is one record with its FULL untruncated description, blocks
	// separated by a `---` rule.
	Description bool
	// Artifact, when set, restricts the artifact section to the single named item,
	// shown as its full block (implies the block view). Empty means all artifacts.
	Artifact string
}

// JSON writes v as indented JSON.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

const descWidth = 60

// PrintCatalog renders the catalog as aligned text sections.
func PrintCatalog(w io.Writer, cat *registry.Catalog, view CatalogView) {
	if view.Artifacts {
		printArtifacts(w, cat.Artifacts, view)
	}
	if view.Recipes {
		printRecipes(w, cat.Recipes)
	}
	if view.Profiles {
		printProfiles(w, cat.Profiles, view.Layers)
	}
	if view.Plugins {
		printPlugins(w, cat.Plugins)
	}
}

func printPlugins(w io.Writer, entries []registry.PluginEntry) {
	fmt.Fprintln(w, "PLUGINS")
	if len(entries) == 0 {
		fmt.Fprintln(w, "  (none)")
		fmt.Fprintln(w)
		return
	}
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tDESCRIPTION")
	for _, e := range entries {
		m := e.Manifest
		fmt.Fprintf(tw, "  %s\t%s\n", m.Name, truncate(m.Description, descWidth))
	}
	tw.Flush()
	fmt.Fprintln(w)
}

// printArtifacts renders the artifact section. The default is a compact table
// (NAME/TYPE/ROLE/TARGETS, no description). With --description (or a single
// --artifact lookup) it switches to a block/list view carrying the full
// description.
func printArtifacts(w io.Writer, entries []registry.ArtifactEntry, view CatalogView) {
	fmt.Fprintln(w, "ARTIFACTS")

	// Single-artifact lookup: filter to the one named item (block view).
	if view.Artifact != "" {
		for _, e := range entries {
			if e.Manifest.Name == view.Artifact {
				printArtifactBlocks(w, []registry.ArtifactEntry{e})
				return
			}
		}
		fmt.Fprintf(w, "  no artifact named %q\n\n", view.Artifact)
		return
	}

	if len(entries) == 0 {
		fmt.Fprintln(w, "  (none)")
		fmt.Fprintln(w)
		return
	}

	if view.Description {
		printArtifactBlocks(w, entries)
		return
	}

	// Compact default: drop the description column so a row fits the terminal.
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tTYPE\tROLE\tTARGETS")
	for _, e := range entries {
		m := e.Manifest
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", m.Name, m.Type, m.Role, joinList(m.Targets))
	}
	tw.Flush()
	fmt.Fprintln(w)
}

// printArtifactBlocks renders one record per artifact with its FULL description,
// blocks separated by a `---` rule so a long description is readable in full.
func printArtifactBlocks(w io.Writer, entries []registry.ArtifactEntry) {
	for _, e := range entries {
		m := e.Manifest
		fmt.Fprintln(w, "---")
		fmt.Fprintf(w, "%s\n", m.Name)
		fmt.Fprintf(w, "  type:        %s\n", m.Type)
		fmt.Fprintf(w, "  role:        %s\n", m.Role)
		fmt.Fprintf(w, "  targets:     %s\n", joinList(m.Targets))
		fmt.Fprintf(w, "  description: %s\n", m.Description)
	}
	fmt.Fprintln(w, "---")
	fmt.Fprintln(w)
}

func printRecipes(w io.Writer, entries []registry.RecipeEntry) {
	fmt.Fprintln(w, "RECIPES")
	if len(entries) == 0 {
		fmt.Fprintln(w, "  (none)")
		fmt.Fprintln(w)
		return
	}
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tTYPE\tROLE\tSUMMARY")
	for _, e := range entries {
		m := e.Manifest
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", m.Name, m.Shape(), m.Role, truncate(m.Summary, descWidth))
	}
	tw.Flush()
	fmt.Fprintln(w)
}

func printProfiles(w io.Writer, entries []registry.ProfileEntry, layers bool) {
	fmt.Fprintln(w, "PROFILES")
	if len(entries) == 0 {
		fmt.Fprintln(w, "  (none)")
		fmt.Fprintln(w)
		return
	}
	if !layers {
		tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "  NAME\tSTATUS\tSUMMARY")
		for _, e := range entries {
			m := e.Manifest
			fmt.Fprintf(tw, "  %s\t%s\t%s\n", m.Name, statusOrDash(m.Status), truncate(m.Summary, descWidth))
		}
		tw.Flush()
		fmt.Fprintln(w)
		return
	}
	for _, e := range entries {
		printProfileLayers(w, e.Manifest)
	}
	fmt.Fprintln(w)
}

func printProfileLayers(w io.Writer, m *manifest.Profile) {
	header := m.Name
	if m.Status != "" {
		header = fmt.Sprintf("%s (%s)", m.Name, m.Status)
	}
	fmt.Fprintln(w, header)

	type row struct {
		key string
		val string
	}
	var rows []row
	add := func(k string, list []string) {
		if len(list) > 0 {
			rows = append(rows, row{k, joinList(list)})
		}
	}
	addScalar := func(k, v string) {
		if v != "" {
			rows = append(rows, row{k, v})
		}
	}
	L := m.Layers
	add("instructions", L.Instructions)
	add("capabilities", L.Capabilities)
	add("context", L.Context)
	add("tools", L.Tools)
	addScalar("memory", L.Memory)
	add("sandbox", L.Sandbox)
	add("observability", L.Observability)
	add("eval", L.Eval)
	add("guardrails", L.Guardrails)

	for i, r := range rows {
		branch := "├──"
		if i == len(rows)-1 {
			branch = "└──"
		}
		fmt.Fprintf(w, "%s %s: %s\n", branch, r.key, r.val)
	}
	if len(rows) == 0 {
		fmt.Fprintln(w, "└── (no populated layers)")
	}
}

// PrintInventory renders a scan inventory as text.
func PrintInventory(w io.Writer, inv *scan.Inventory) {
	fmt.Fprintf(w, "Scanned project: %s\n", inv.ProjectDir)
	fmt.Fprintf(w, "Home:            %s\n", inv.Home)
	if e := inv.Env; e.CodexHome != "" || e.OpencodeConfigDir != "" || e.XDGConfigHome != "" {
		fmt.Fprintln(w, "Env overrides:")
		if e.CodexHome != "" {
			fmt.Fprintf(w, "  CODEX_HOME=%s\n", e.CodexHome)
		}
		if e.OpencodeConfigDir != "" {
			fmt.Fprintf(w, "  OPENCODE_CONFIG_DIR=%s\n", e.OpencodeConfigDir)
		}
		if e.XDGConfigHome != "" {
			fmt.Fprintf(w, "  XDG_CONFIG_HOME=%s\n", e.XDGConfigHome)
		}
	}
	fmt.Fprintln(w)

	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "TOOL\tSCOPE\tDETECTED\tEVIDENCE")
	for _, t := range inv.Tools {
		printDetection(tw, t.Tool, t.Global)
		printDetection(tw, t.Tool, t.Local)
	}
	tw.Flush()
}

func printDetection(tw io.Writer, tool string, d *scan.Detection) {
	if d == nil {
		return
	}
	status := "no"
	evidence := "-"
	if d.Detected {
		status = "yes"
		evidence = joinList(d.MatchedPaths)
	}
	fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", tool, d.Scope, status, evidence)
}

func statusOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func joinList(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	out := items[0]
	for _, s := range items[1:] {
		out += ", " + s
	}
	return out
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
