package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/state"
)

const workflowDeliveryName = "fixture-workflow"

type workflowDeliveryFixture struct {
	root, home, agentDir, scope string
}

// Only the adapter is production content. The catalog and all delivered bytes
// are authored here; no package fixtures, registry pins or native installs apply.
func newWorkflowDeliveryFixture(t *testing.T, location string) workflowDeliveryFixture {
	t.Helper()
	source, err := registry.DiscoverRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	f := workflowDeliveryFixture{root: t.TempDir(), home: t.TempDir(), scope: "global"}
	dp06Write(t, filepath.Join(f.root, "adapters/pi.yaml"), string(mustRead(t, filepath.Join(source, "adapters/pi.yaml"))))
	f.agentDir = filepath.Join(f.home, ".pi/agent")
	t.Setenv("HOME", f.home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(f.home, ".config"))
	t.Setenv("PI_CODING_AGENT_DIR", "")
	t.Setenv("PI_PACKAGE_DIR", "")
	t.Setenv("PI_SUBAGENT_EXTRA_AGENT_DIRS", "")
	t.Setenv("PI_MCP_CONFIG_MODE", "")
	t.Setenv("PI_OFFLINE", "true")
	switch location {
	case "local":
		f.scope = "local"
		f.agentDir = filepath.Join(f.root, ".pi")
	case "custom":
		f.agentDir = t.TempDir()
		t.Setenv("PI_CODING_AGENT_DIR", f.agentDir)
	case "global":
	default:
		t.Fatalf("unknown fixture location %q", location)
	}
	if err := os.MkdirAll(f.agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(f.root)
	f.publish(t, "1.0.0")
	return f
}

func workflowDeliveryBytes(version, skillDir string) map[string]string {
	return map[string]string{
		"SKILL.md":            "---\nname: fixture-workflow\ndescription: Invented workflow delivery fixture\n---\nRun " + skillDir + "/workflow.js with " + skillDir + "/request.schema.json.\n",
		"workflow.js":         fmt.Sprintf("// invented workflow %s\nreturn {version: %q, schema: %q};\n", version, version, skillDir+"/request.schema.json"),
		"request.schema.json": fmt.Sprintf("{\"type\":\"object\",\"title\":\"request %s\",\"properties\":{\"task\":{\"type\":\"string\"}}}\n", version),
	}
}

func (f workflowDeliveryFixture) publish(t *testing.T, version string) {
	t.Helper()
	src := filepath.Join(f.root, "artifacts", workflowDeliveryName)
	dp06Write(t, filepath.Join(src, "patronus.yaml"), fmt.Sprintf("apiVersion: patronus/v2\nfamily: artifact\nname: %s\nversion: %s\nrole: capability\ntype: skill\ndescription: Invented workflow\ntargets: [pi]\nentry: SKILL.md\nfiles: [workflow.js, request.schema.json]\n", workflowDeliveryName, version))
	for name, body := range workflowDeliveryBytes(version, "{skillDir}") {
		dp06Write(t, filepath.Join(src, name), body)
	}
}

func (f workflowDeliveryFixture) skillDir() string {
	return filepath.Join(f.agentDir, "skills", workflowDeliveryName)
}
func (f workflowDeliveryFixture) args(name string, extra ...string) []string {
	return append([]string{name, "--target", "pi", "--" + f.scope}, extra...)
}
func (f workflowDeliveryFixture) install(t *testing.T) {
	t.Helper()
	if _, _, err := runInstall(t, f.args(workflowDeliveryName, "--deploy")...); err != nil {
		t.Fatal(err)
	}
}
func (f workflowDeliveryFixture) wantVersion(t *testing.T, version string) {
	t.Helper()
	ref := f.skillDir()
	if f.scope == "local" {
		ref = filepath.Join(".pi", "skills", workflowDeliveryName)
	}
	for name, want := range workflowDeliveryBytes(version, ref) {
		workflowDeliveryWantBytes(t, filepath.Join(f.skillDir(), name), want)
	}
	installed, err := state.Load(removeStatePath(f.scope, f.home, f.root))
	if err != nil {
		t.Fatal(err)
	}
	// Match concrete paths rather than assuming one name/tool/scope row across
	// all custom roots. Check ownership hashes against the deployed bytes.
	owned := make(map[string]int)
	for _, item := range installed.Find(workflowDeliveryName, "pi", f.scope) {
		for _, file := range item.Files {
			if filepath.Dir(file.Path) != f.skillDir() {
				continue
			}
			owned[filepath.Base(file.Path)]++
			if item.ItemVersion != version || file.Checksum != shaState(mustRead(t, file.Path)) {
				t.Errorf("ownership/version does not match deployed %s: %+v", file.Path, item)
			}
		}
	}
	for name := range workflowDeliveryBytes(version, ref) {
		if owned[name] != 1 {
			t.Errorf("%s has %d ownership records, want 1", name, owned[name])
		}
	}
}
func workflowDeliveryWantBytes(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("read %s: %v", path, err)
		return
	}
	if string(got) != want {
		t.Errorf("%s bytes = %q, want %q", path, got, want)
	}
}
func workflowDeliveryWantAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("expected absent %s: %v", path, err)
	}
}

func TestPiWorkflowDeliveryInstallPreviewAndUpdate(t *testing.T) {
	for _, location := range []string{"global", "local", "custom"} {
		t.Run(location, func(t *testing.T) {
			f := newWorkflowDeliveryFixture(t, location)
			beforeHome, beforeRoot := dp01SnapshotFiles(t, f.home), dp01SnapshotFiles(t, f.root)
			beforeAgent := dp01SnapshotFiles(t, f.agentDir)
			if _, _, err := runInstall(t, f.args(workflowDeliveryName)...); err != nil {
				t.Fatal(err)
			}
			dp06SameSnapshot(t, f.home, beforeHome)
			dp06SameSnapshot(t, f.root, beforeRoot)
			dp06SameSnapshot(t, f.agentDir, beforeAgent)
			f.install(t)
			f.wantVersion(t, "1.0.0")
			f.publish(t, "2.0.0")
			beforeHome, beforeRoot = dp01SnapshotFiles(t, f.home), dp01SnapshotFiles(t, f.root)
			beforeAgent = dp01SnapshotFiles(t, f.agentDir)
			if _, _, err := runUpdate(t, f.args(workflowDeliveryName)...); err != nil {
				t.Fatal(err)
			}
			dp06SameSnapshot(t, f.home, beforeHome)
			dp06SameSnapshot(t, f.root, beforeRoot)
			dp06SameSnapshot(t, f.agentDir, beforeAgent)
			if _, _, err := runUpdate(t, f.args(workflowDeliveryName, "--deploy")...); err != nil {
				t.Fatal(err)
			}
			f.wantVersion(t, "2.0.0")
		})
	}
}

func TestPiWorkflowDeliveryEditedUpdatePreservesBatch(t *testing.T) {
	for _, location := range []string{"global", "local", "custom"} {
		t.Run(location, func(t *testing.T) {
			f := newWorkflowDeliveryFixture(t, location)
			f.install(t)
			dp06Write(t, filepath.Join(f.skillDir(), "workflow.js"), "// operator edit\n")
			f.publish(t, "2.0.0")
			beforeHome, beforeRoot := dp01SnapshotFiles(t, f.home), dp01SnapshotFiles(t, f.root)
			beforeAgent := dp01SnapshotFiles(t, f.agentDir)
			// Pi refuses an edited owned path before applying the batch. It must not
			// update the unedited schema or advance state past the retained script.
			if _, _, err := runUpdate(t, f.args(workflowDeliveryName, "--deploy")...); err == nil {
				t.Error("edited workflow update succeeded")
			}
			dp06SameSnapshot(t, f.home, beforeHome)
			dp06SameSnapshot(t, f.root, beforeRoot)
			dp06SameSnapshot(t, f.agentDir, beforeAgent)
		})
	}
}

func TestPiWorkflowDeliverySelectiveRemoval(t *testing.T) {
	for _, location := range []string{"global", "local", "custom"} {
		for _, editedFile := range []string{"workflow.js", "request.schema.json"} {
			t.Run(location+"/"+editedFile, func(t *testing.T) {
				f := newWorkflowDeliveryFixture(t, location)
				f.install(t)
				edited := filepath.Join(f.skillDir(), editedFile)
				dp06Write(t, edited, "operator-owned edit\n")
				sibling := filepath.Join(f.agentDir, "skills", "unrelated", "workflow.js")
				untracked := filepath.Join(f.skillDir(), "operator-notes.txt")
				dp06Write(t, sibling, "unrelated sibling\n")
				dp06Write(t, untracked, "untracked notes\n")
				beforeHome, beforeRoot := dp01SnapshotFiles(t, f.home), dp01SnapshotFiles(t, f.root)
				beforeAgent := dp01SnapshotFiles(t, f.agentDir)
				// A preview may report incomplete removal, but must never mutate bytes/state.
				_, _, _ = execRemove(t, f.args(workflowDeliveryName)...)
				dp06SameSnapshot(t, f.home, beforeHome)
				dp06SameSnapshot(t, f.root, beforeRoot)
				dp06SameSnapshot(t, f.agentDir, beforeAgent)
				if _, _, err := execRemove(t, f.args(workflowDeliveryName, "--deploy")...); err == nil {
					t.Error("edited removal must report incomplete")
				}
				workflowDeliveryWantBytes(t, edited, "operator-owned edit\n")
				for name := range workflowDeliveryBytes("1.0.0", "") {
					if name != editedFile {
						workflowDeliveryWantAbsent(t, filepath.Join(f.skillDir(), name))
					}
				}
				workflowDeliveryWantBytes(t, sibling, "unrelated sibling\n")
				workflowDeliveryWantBytes(t, untracked, "untracked notes\n")
				if _, _, err := execRemove(t, f.args(workflowDeliveryName, "--deploy", "--force")...); err != nil {
					t.Errorf("force removal: %v", err)
				}
				for name := range workflowDeliveryBytes("1.0.0", "") {
					workflowDeliveryWantAbsent(t, filepath.Join(f.skillDir(), name))
				}
				workflowDeliveryWantBytes(t, sibling, "unrelated sibling\n")
				workflowDeliveryWantBytes(t, untracked, "untracked notes\n")
			})
		}
	}
}

func TestPiWorkflowDeliveryCustomRootsIndependent(t *testing.T) {
	f := newWorkflowDeliveryFixture(t, "custom")
	f.install(t)
	first := f.skillDir()
	before := dp01SnapshotFiles(t, f.agentDir)
	other := t.TempDir()
	t.Setenv("PI_CODING_AGENT_DIR", other)
	f.agentDir = other
	f.install(t)
	f.wantVersion(t, "1.0.0")
	if _, _, err := execRemove(t, f.args(workflowDeliveryName, "--deploy", "--force")...); err != nil {
		t.Errorf("second root removal: %v", err)
	}
	workflowDeliveryWantAbsent(t, f.skillDir())
	dp06SameSnapshot(t, filepath.Dir(filepath.Dir(first)), before)
	// The first root's ownership must survive too, not just its bytes.
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Dir(filepath.Dir(first)))
	if _, _, err := execRemove(t, f.args(workflowDeliveryName, "--deploy")...); err != nil {
		t.Errorf("first root ownership lost: %v", err)
	}
	workflowDeliveryWantAbsent(t, first)
}

func TestPiWorkflowDeliverySharedProfileRemoval(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprintf("force=%t", force), func(t *testing.T) {
			f := newWorkflowDeliveryFixture(t, "custom")
			for _, profile := range []string{"workflow-team-a", "workflow-team-b"} {
				dp06Write(t, filepath.Join(f.root, "profiles", profile+".yaml"), fmt.Sprintf("apiVersion: patronus/v2\nfamily: profile\nname: %s\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  capabilities: [%s@pi]\n", profile, workflowDeliveryName))
				if _, _, err := runInstall(t, "--profile", profile, "--target", "pi", "--global", "--deploy"); err != nil {
					t.Fatal(err)
				}
			}
			f.wantVersion(t, "1.0.0")
			args := f.args("workflow-team-a", "--deploy")
			if force {
				args = append(args, "--force")
			}
			_, _, err := execRemove(t, args...)
			if force {
				if err != nil {
					t.Errorf("forced stored-profile removal: %v", err)
				}
				workflowDeliveryWantAbsent(t, f.skillDir())
			} else {
				if err == nil || strings.Contains(err.Error(), "not installed") {
					t.Errorf("shared profile removal must recognize and retain membership: %v", err)
				}
				f.wantVersion(t, "1.0.0")
				// Retained membership must remain actionable for a later forced removal.
				if _, _, err := execRemove(t, f.args("workflow-team-a", "--deploy", "--force")...); err != nil {
					t.Errorf("retained profile removal: %v", err)
				}
				workflowDeliveryWantAbsent(t, f.skillDir())
			}
		})
	}
}

func TestPiWorkflowDeliveryUneditedRemoval(t *testing.T) {
	for _, location := range []string{"global", "local", "custom"} {
		t.Run(location, func(t *testing.T) {
			f := newWorkflowDeliveryFixture(t, location)
			f.install(t)
			f.wantVersion(t, "1.0.0")
			sibling := filepath.Join(f.agentDir, "skills", "unrelated", "SKILL.md")
			dp06Write(t, sibling, "unrelated skill\n")
			if _, _, err := execRemove(t, f.args(workflowDeliveryName, "--deploy")...); err != nil {
				t.Fatal(err)
			}
			workflowDeliveryWantAbsent(t, f.skillDir())
			workflowDeliveryWantBytes(t, sibling, "unrelated skill\n")
		})
	}
}

func TestPiWorkflowDeliveryRemovalDoesNotAliasCustomRoot(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprintf("force=%t", force), func(t *testing.T) {
			f := newWorkflowDeliveryFixture(t, "custom")
			f.install(t)
			before := dp01SnapshotFiles(t, f.agentDir)
			// Selecting an empty root must not select the same name in the old root's
			// state. This isolates removal aliasing from second-install admission.
			t.Setenv("PI_CODING_AGENT_DIR", t.TempDir())
			args := f.args(workflowDeliveryName, "--deploy")
			if force {
				args = append(args, "--force")
			}
			_, _, _ = execRemove(t, args...)
			dp06SameSnapshot(t, f.agentDir, before)
			t.Setenv("PI_CODING_AGENT_DIR", f.agentDir)
			if _, _, err := execRemove(t, f.args(workflowDeliveryName, "--deploy")...); err != nil {
				t.Errorf("original root ownership lost: %v", err)
			}
			workflowDeliveryWantAbsent(t, f.skillDir())
		})
	}
}
