package main

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// HS-R5: no admission or target-dispatch literal remains in cmd/patronus or in
// internal/{state,remove,lock}; internal/{adapter,recipe,toolpath,plan,profile,
// scan} carry no unreviewed "opencode" literal (OC-T4). Every remaining target
// literal is enumerated
// in the reviewed allowlist fixture with a stable site identity and a class:
// target-capability (tool pi|codex), descriptor-construction, adapter-data or
// legacy-zero-value. The test fails on an unlisted site, a stale row, a
// multiplicity mismatch and on any capability row naming opencode.

const harnessLiteralFixture = "testdata/harness-literal-allowlist.yaml"

// harnessLiteralFixtureSHA256 pins the reviewed fixture bytes: a class change
// is a review event, not a silent edit.
const harnessLiteralFixtureSHA256 = "bfb67c71f9fd6d38d2d9955d5dafd03cd6f4181c68c4e9b667b17e69fd1d2c40"

type literalSite struct {
	File    string `yaml:"file"`
	Decl    string `yaml:"decl"`
	ASTPath string `yaml:"astPath"`
	Index   int    `yaml:"index"`
	Literal string `yaml:"literal"`
	Class   string `yaml:"class,omitempty"`
	Tool    string `yaml:"tool,omitempty"`
}

func (s literalSite) identity() string {
	return s.File + "|" + s.Decl + "|" + s.ASTPath + "|" + strconv.Itoa(s.Index) + "|" + s.Literal
}

func harnessTargetLiteral(value string) bool {
	switch value {
	case "pi", "codex", "claude", "opencode":
		return true
	}
	return false
}

// literalSites collects every target literal in one parsed file.
func literalSites(file string, f *ast.File) []literalSite {
	var sites []literalSite
	for _, decl := range f.Decls {
		name := "?"
		switch d := decl.(type) {
		case *ast.FuncDecl:
			name = d.Name.Name
			if d.Recv != nil && len(d.Recv.List) == 1 {
				name = receiverName(d.Recv.List[0].Type) + "." + name
			}
		case *ast.GenDecl:
			var names []string
			for _, spec := range d.Specs {
				switch sp := spec.(type) {
				case *ast.ValueSpec:
					for _, n := range sp.Names {
						names = append(names, n.Name)
					}
				case *ast.TypeSpec:
					names = append(names, sp.Name.Name)
				}
			}
			name = d.Tok.String() + ":" + strings.Join(names, ",")
		}
		counts := map[string]int{}
		var path []string
		ast.Inspect(decl, func(n ast.Node) bool {
			if n == nil {
				path = path[:len(path)-1]
				return false
			}
			path = append(path, strings.TrimPrefix(fmt.Sprintf("%T", n), "*ast."))
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil || !harnessTargetLiteral(value) {
				return true
			}
			sites = append(sites, literalSite{File: file, Decl: name, ASTPath: strings.Join(path, "/"), Index: counts[value], Literal: value})
			counts[value]++
			return true
		})
	}
	return sites
}

func receiverName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return "?"
}

func harnessLiteralDirs(t *testing.T) []string {
	t.Helper()
	root := applicationSourceRoot(t)
	return []string{filepath.Join(root, "cmd/patronus"), filepath.Join(root, "internal/state"), filepath.Join(root, "internal/remove"), filepath.Join(root, "internal/lock")}
}

// harnessOpenCodeLiteralDirs is the OC-T4 extension of the parse set (OP-13):
// these packages are scanned for the "opencode" literal only, so every
// OpenCode site is a reviewed adapter-data row; their Pi and Codex literals
// remain the HS-O3 follow-up.
func harnessOpenCodeLiteralDirs(t *testing.T) []string {
	t.Helper()
	root := applicationSourceRoot(t)
	var dirs []string
	for _, pkg := range []string{"adapter", "recipe", "toolpath", "plan", "profile", "scan"} {
		dirs = append(dirs, filepath.Join(root, "internal", pkg))
	}
	return dirs
}

func observedLiteralSites(t *testing.T) []literalSite {
	t.Helper()
	sites := parseLiteralDirs(t, harnessLiteralDirs(t), func(literalSite) bool { return true })
	return append(sites, parseLiteralDirs(t, harnessOpenCodeLiteralDirs(t), func(s literalSite) bool { return s.Literal == "opencode" })...)
}

func parseLiteralDirs(t *testing.T, dirs []string, keep func(literalSite) bool) []literalSite {
	t.Helper()
	root := applicationSourceRoot(t)
	var sites []literalSite
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			rel, _ := filepath.Rel(root, path)
			for _, s := range literalSites(filepath.ToSlash(rel), f) {
				if keep(s) {
					sites = append(sites, s)
				}
			}
		}
	}
	return sites
}

func loadLiteralFixture(t *testing.T) ([]literalSite, []byte) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(applicationSourceRoot(t), "cmd/patronus", harnessLiteralFixture))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Rows []literalSite `yaml:"rows"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Rows, raw
}

// compareLiteralSites reports every difference between observed and allowed
// sites: unlisted observed sites, stale rows, multiplicity mismatches per
// {file, decl, literal} and forbidden rows.
func compareLiteralSites(observed, allowed []literalSite) []string {
	var problems []string
	allowedBy := map[string]literalSite{}
	for _, row := range allowed {
		switch row.Class {
		case "target-capability":
			if row.Tool != "pi" && row.Tool != "codex" {
				problems = append(problems, "capability row without tool pi|codex: "+row.identity())
			}
			if row.Literal == "opencode" || row.Tool == "opencode" {
				problems = append(problems, "capability row names opencode: "+row.identity())
			}
		case "descriptor-construction", "adapter-data", "legacy-zero-value":
		default:
			problems = append(problems, "row with unknown class "+strconv.Quote(row.Class)+": "+row.identity())
		}
		if _, dup := allowedBy[row.identity()]; dup {
			problems = append(problems, "duplicate row: "+row.identity())
		}
		allowedBy[row.identity()] = row
	}
	observedBy := map[string]bool{}
	for _, site := range observed {
		observedBy[site.identity()] = true
		if _, ok := allowedBy[site.identity()]; !ok {
			problems = append(problems, "unlisted target literal: "+site.identity())
		}
	}
	for id := range allowedBy {
		if !observedBy[id] {
			problems = append(problems, "stale fixture row: "+id)
		}
	}
	multiplicity := func(sites []literalSite) map[string]int {
		out := map[string]int{}
		for _, s := range sites {
			out[s.File+"|"+s.Decl+"|"+s.Literal]++
		}
		return out
	}
	want, got := multiplicity(allowed), multiplicity(observed)
	for key, n := range got {
		if want[key] != n {
			problems = append(problems, fmt.Sprintf("multiplicity %s: observed %d allowed %d", key, n, want[key]))
		}
	}
	sort.Strings(problems)
	return problems
}

func TestHarnessNoDispatchLiterals(t *testing.T) {
	allowed, raw := loadLiteralFixture(t)
	if sum := fmt.Sprintf("%x", sha256.Sum256(raw)); sum != harnessLiteralFixtureSHA256 {
		t.Fatalf("fixture bytes changed (sha256 %s); review the allowlist and update harnessLiteralFixtureSHA256", sum)
	}
	observed := observedLiteralSites(t)
	if problems := compareLiteralSites(observed, allowed); len(problems) > 0 {
		var rows strings.Builder
		for _, s := range observed {
			fmt.Fprintf(&rows, "  - {file: %s, decl: %s, astPath: %s, index: %d, literal: %s}\n", s.File, s.Decl, s.ASTPath, s.Index, s.Literal)
		}
		t.Fatalf("%d problem(s):\n%s\nobserved rows:\n%s", len(problems), strings.Join(problems, "\n"), rows.String())
	}
}

// Negative cases over an invented source parsed in memory.
func TestHarnessNoDispatchLiteralsNegative(t *testing.T) {
	const src = `package toy

func capability(tool string) bool { return tool == "pi" }

func other(tool string) bool { return tool == "codex" }
`
	parse := func(code string) []literalSite {
		f, err := parser.ParseFile(token.NewFileSet(), "toy.go", code, 0)
		if err != nil {
			t.Fatal(err)
		}
		return literalSites("toy.go", f)
	}
	baseline := parse(src)
	allowed := make([]literalSite, 0, len(baseline))
	for _, s := range baseline {
		s.Class, s.Tool = "target-capability", s.Literal
		allowed = append(allowed, s)
	}
	if problems := compareLiteralSites(baseline, allowed); len(problems) != 0 {
		t.Fatalf("baseline must match itself: %v", problems)
	}
	// Inserting a dispatch literal into a function that already holds an allowed
	// identical capability literal is caught by the occurrence index/multiplicity.
	inserted := parse(strings.Replace(src, `return tool == "pi" }`, `if tool == "pi" { return true }; return tool == "pi" }`, 1))
	problems := compareLiteralSites(inserted, allowed)
	if len(problems) == 0 || !strings.Contains(strings.Join(problems, "\n"), "multiplicity toy.go|capability|pi") {
		t.Fatalf("inserted literal not detected: %v", problems)
	}
	// Deleting a permitted site leaves a stale row.
	deleted := parse(strings.Replace(src, `tool == "codex"`, `tool == ""`, 1))
	problems = compareLiteralSites(deleted, allowed)
	if len(problems) == 0 || !strings.Contains(strings.Join(problems, "\n"), "stale fixture row: toy.go|other") {
		t.Fatalf("stale row not detected: %v", problems)
	}
	// A capability row may never name opencode.
	bad := append([]literalSite(nil), allowed...)
	bad[0].Tool = "opencode"
	if problems := compareLiteralSites(baseline, bad); len(problems) == 0 || !strings.Contains(problems[0], "opencode") {
		t.Fatalf("opencode capability row accepted: %v", problems)
	}
	// A class change shows as a fixture byte change (hash) and an unknown class fails.
	bad = append([]literalSite(nil), allowed...)
	bad[0].Class = "invented-class"
	if problems := compareLiteralSites(baseline, bad); len(problems) == 0 {
		t.Fatal("unknown class accepted")
	}
	a, _ := yaml.Marshal(allowed)
	b, _ := yaml.Marshal(bad)
	if sha256.Sum256(a) == sha256.Sum256(b) {
		t.Fatal("class change invisible to the fixture hash")
	}
}
