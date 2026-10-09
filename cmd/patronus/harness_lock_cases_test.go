package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/lock"
)

// Target enum parity: dp06Target (install/remove/lock/update flag) and the lock
// schema's validateTarget (through lock.Save) accept the same named set; only
// dp06Target accepts the unset sentinel, and lock.Save checks the target only
// on the target-bearing schema versions.
func TestHarnessTargetEnumParity(t *testing.T) {
	for _, tc := range []struct {
		target        string
		dp06Accepted  bool
		lockAccepted  bool
		dp06ErrorText string
	}{
		{"", true, false, ""},
		{"all", true, true, ""},
		{"claude", true, true, ""},
		{"codex", true, true, ""},
		{"opencode", true, true, ""},
		{"pi", true, true, ""},
		{"invented", false, false, "unknown target \"invented\"; use claude|codex|opencode|pi|all"},
	} {
		t.Run("target="+tc.target, func(t *testing.T) {
			err := dp06Target(testRegistry(t), tc.target)
			if (err == nil) != tc.dp06Accepted {
				t.Fatalf("dp06Target(testRegistry(t), %q) = %v", tc.target, err)
			}
			if err != nil && err.Error() != tc.dp06ErrorText {
				t.Fatalf("dp06Target(testRegistry(t), %q) text %q", tc.target, err)
			}
			path := filepath.Join(t.TempDir(), "patronus.lock")
			err = lock.Save(path, &lock.Lock{Version: lock.TargetVersion, Target: tc.target})
			if (err == nil) != tc.lockAccepted {
				t.Fatalf("lock.Save v3 target %q = %v", tc.target, err)
			}
			if err != nil && err.Error() != "lock schema v3 requires a known nonempty target, got \""+tc.target+"\"" {
				t.Fatalf("lock.Save text %q", err)
			}
			// Versions below the target-bearing schema never check the target.
			if err := lock.Save(path, &lock.Lock{Version: lock.Version, Target: tc.target}); err != nil {
				t.Fatalf("lock.Save v%d target %q = %v", lock.Version, tc.target, err)
			}
		})
	}
}

// harnessLaneCatalog adds an invented profile whose closure includes a native Pi
// delivery recipe, so the only mechanism refusing a foreign lock target for it is
// lock/build.go's "native recipe requires Pi lock target".
func harnessLaneCatalog(t *testing.T) string {
	t.Helper()
	root := fixtureCatalog(t)
	codexWrite(t, filepath.Join(root, "recipes/fixture-native.yaml"), []byte("apiVersion: patronus/v2\nfamily: recipe\nname: fixture-native\nversion: 1.0.0\nrole: tools\ndeliver:\n  via: package-manager\n  install:\n    - manager: pi\n      ref: npm:@invented/plugin@1.2.3\n"))
	codexWrite(t, filepath.Join(root, "profiles/fixture-lane.yaml"), []byte("apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: fixture-lane\nversion: 1.0.0\nlayers:\n  capabilities:\n    - fix-skill\n    - fixture-native\n"))
	return root
}

func TestHarnessLaneProfileRefusesClaude(t *testing.T) {
	root := harnessLaneCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PI_OFFLINE", "true")
	t.Chdir(root)
	_, _, err := runLock(t, "--profile", "fixture-lane", "--target", "claude")
	if got := harnessErr(err); got != "native recipe requires Pi lock target" {
		t.Fatalf("lane profile lock --target claude: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "patronus.lock")); !os.IsNotExist(err) {
		t.Fatalf("refused lock wrote patronus.lock: %v", err)
	}
	if _, _, err := runLock(t, "--profile", "fixture-lane", "--target", "pi"); err != nil {
		t.Fatalf("lane profile lock --target pi: %v", err)
	}
}

// TestHarnessLockCoexistence records the complete install/update/lock outcome
// matrix over a prior pi or codex lock (HP-02). Today codexLockTarget fires only
// when the prior or the requested target is codex; the lock command refuses any
// changed target over a pi or codex lock; dp06CheckProfileLock refuses a differing
// target only for a matching profile. Accepted rows are asserted as accepted.
func TestHarnessLockCoexistence(t *testing.T) {
	const profile = "fixture-any-profile"
	const (
		codexConflict = "existing lock target %s conflicts with --target %s; use separate project roots or explicit migration"
		lockConflict  = "existing lock target %s conflicts with --target %s; explicit migration required"
		profileLock   = "patronus.lock target %s differs from requested %s"
	)
	type row struct {
		prior, command, target string
		want                   string // "" accepted; %s pairs are (prior, target); other text is the post-lock outcome
	}
	const (
		targetRequired = "--target is required (one of claude|codex|opencode|pi|all) for: fixture-any"
		nothingInstall = "nothing is installed (no state recorded)"
		piNotInstalled = "not installed in selected global root: fixture-any; update cannot relocate ownership"
	)
	rows := []row{
		// prior pi lock: codexLockTarget never fires unless codex is requested.
		{"pi", "install", "claude", ""},
		{"pi", "install", "all", ""},
		{"pi", "install", "codex", codexConflict},
		{"pi", "install", "pi", ""},
		{"pi", "install", "", targetRequired},
		{"pi", "install-profile", "claude", profileLock},
		{"pi", "install-profile", "all", profileLock},
		{"pi", "install-profile", "codex", codexConflict},
		{"pi", "install-profile", "pi", ""},
		{"pi", "install-profile", "", ""}, // target inherited from the matching profile lock
		{"pi", "update", "claude", nothingInstall},
		{"pi", "update", "all", nothingInstall},
		{"pi", "update", "codex", codexConflict},
		{"pi", "update", "pi", piNotInstalled},
		{"pi", "update", "", nothingInstall},
		{"pi", "lock", "claude", lockConflict},
		{"pi", "lock", "all", lockConflict},
		{"pi", "lock", "codex", lockConflict},
		{"pi", "lock", "pi", ""},
		{"pi", "lock", "", ""},
		// prior codex lock: every differing explicit target is refused by codexLockTarget.
		{"codex", "install", "claude", codexConflict},
		{"codex", "install", "all", codexConflict},
		{"codex", "install", "codex", ""},
		{"codex", "install", "pi", codexConflict},
		{"codex", "install", "", targetRequired},
		{"codex", "install-profile", "claude", codexConflict},
		{"codex", "install-profile", "all", codexConflict},
		{"codex", "install-profile", "codex", ""},
		{"codex", "install-profile", "pi", codexConflict},
		{"codex", "install-profile", "", ""},
		{"codex", "update", "claude", codexConflict},
		{"codex", "update", "all", codexConflict},
		{"codex", "update", "codex", nothingInstall},
		{"codex", "update", "pi", codexConflict},
		{"codex", "update", "", nothingInstall},
		{"codex", "lock", "claude", lockConflict},
		{"codex", "lock", "all", lockConflict},
		{"codex", "lock", "codex", ""},
		{"codex", "lock", "pi", lockConflict},
		{"codex", "lock", "", ""},
	}
	for _, tc := range rows {
		t.Run(tc.prior+"/"+tc.command+"/target="+tc.target, func(t *testing.T) {
			root := harnessAnyCatalog(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
			t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, ".pi", "agent"))
			t.Setenv("PI_OFFLINE", "true")
			t.Chdir(root)
			if err := lock.Save(filepath.Join(root, "patronus.lock"), &lock.Lock{Version: lock.TargetVersion, Target: tc.prior, Profile: profile}); err != nil {
				t.Fatal(err)
			}
			targetArgs := []string{}
			if tc.target != "" {
				targetArgs = []string{"--target", tc.target}
			}
			var err error
			switch tc.command {
			case "install":
				err = harnessRun(func() error {
					_, _, e := runInstall(t, append([]string{"fixture-any", "--global"}, targetArgs...)...)
					return e
				})
			case "install-profile":
				err = harnessRun(func() error {
					_, _, e := runInstall(t, append([]string{"--profile", profile, "--global"}, targetArgs...)...)
					return e
				})
			case "update":
				args := []string{"fixture-any"}
				if tc.target == "pi" {
					args = append(args, "--global") // Pi update requires a scope flag before any lock check
				}
				err = harnessRun(func() error { _, _, e := runUpdate(t, append(args, targetArgs...)...); return e })
			case "lock":
				err = harnessRun(func() error {
					_, _, e := runLock(t, append([]string{"--profile", profile}, targetArgs...)...)
					return e
				})
			}
			want := tc.want
			if strings.Contains(want, "%s") {
				want = fmt.Sprintf(want, tc.prior, tc.target)
			}
			if got := harnessErr(err); got != want {
				t.Fatalf("%s prior=%s target=%q\n got: %s\nwant: %s", tc.command, tc.prior, tc.target, got, want)
			}
		})
	}
}

func harnessRun(f func() error) error { return f() }

// harnessAnyCatalog adds an invented artifact admitted for every target and a
// profile holding only it, so the matrix rows reach the lock rules.
func harnessAnyCatalog(t *testing.T) string {
	t.Helper()
	root := fixtureCatalog(t)
	codexWrite(t, filepath.Join(root, "artifacts/skills/fixture-any/patronus.yaml"), []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: capability\nname: fixture-any\ndescription: \"Invented skill admitted for every target.\"\nversion: 1.0.0\nentry: SKILL.md\ntargets: [claude, codex, opencode, pi]\ndefaults:\n  scope: global\n"))
	codexWrite(t, filepath.Join(root, "artifacts/skills/fixture-any/SKILL.md"), []byte("---\nname: fixture-any\ndescription: invented skill\n---\nBody\n"))
	codexWrite(t, filepath.Join(root, "profiles/fixture-any-profile.yaml"), []byte("apiVersion: patronus/v2\nfamily: profile\nrole: lifecycle\nname: fixture-any-profile\nversion: 1.0.0\nlayers:\n  capabilities:\n    - fixture-any\n"))
	return root
}
