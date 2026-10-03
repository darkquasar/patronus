package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// This invented skill exercises copying, not a shipped skill's file inventory.
func TestPiDeliveryDeclaredHiddenFiles(t *testing.T) {
	for _, scope := range []string{"global", "local"} {
		t.Run(scope, func(t *testing.T) {
			f := dp02Setup(t)
			dp02Artifact(t, f.root, "invented-notebook", "skill", "---\nname: invented-notebook\ndescription: Invented fixture\n---\nRead the notes.\n")
			src := filepath.Join(f.root, "artifacts/invented-notebook")
			manifestPath := filepath.Join(src, "patronus.yaml")
			dp06Write(t, manifestPath, string(mustRead(t, manifestPath))+"files: [.marker, notes]\n")
			want := map[string][]byte{
				".marker":         []byte("invented-version\r\n\x00\xff"),
				"notes/.retained": []byte("hidden nested text\r\n"),
			}
			for name, body := range want {
				dp02File(t, filepath.Join(src, name), string(body))
			}
			dp02File(t, filepath.Join(src, ".undeclared"), "not selected")
			before := dp02Snapshot(t, f.home, f.root)
			if _, _, err := runInstall(t, "invented-notebook", "--target", "pi", "--"+scope); err != nil {
				t.Fatal(err)
			}
			dp07NoEffects(t, f, before)
			if _, _, err := runInstall(t, "invented-notebook", "--target", "pi", "--"+scope, "--deploy"); err != nil {
				t.Fatal(err)
			}
			root := os.Getenv("PI_CODING_AGENT_DIR")
			if scope == "local" {
				root = filepath.Join(f.root, ".pi")
			}
			dest := filepath.Join(root, "skills/invented-notebook")
			for name, body := range want {
				if got := mustRead(t, filepath.Join(dest, name)); !bytes.Equal(got, body) {
					t.Fatalf("hidden file %s changed bytes: got %x, want %x", name, got, body)
				}
			}
			if _, err := os.Stat(filepath.Join(dest, ".undeclared")); !os.IsNotExist(err) {
				t.Fatalf("undeclared hidden file copied: %v", err)
			}
			if _, _, err := execRemove(t, "invented-notebook", "--target", "pi", "--"+scope, "--deploy"); err != nil {
				t.Fatal(err)
			}
			for name := range want {
				if _, err := os.Stat(filepath.Join(dest, name)); !os.IsNotExist(err) {
					t.Fatalf("owned hidden file %s not removed: %v", name, err)
				}
			}
			runner, ok := runnerForCommands.(*fakeRunner)
			if !ok || len(runner.ran) != 0 || f.fetcher.calls != 0 {
				t.Fatal("static skill delivery executed or acquired a payload")
			}
		})
	}
}
