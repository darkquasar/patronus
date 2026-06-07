package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
)

// fakeFetcher serves canned bytes per URL — no network. Returns the configured
// error when the URL is unknown.
type fakeFetcher struct {
	bodies map[string][]byte
}

func (f fakeFetcher) Fetch(_ context.Context, url string) (io.ReadCloser, error) {
	b, ok := f.bodies[url]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestApplyFetchRawBinary(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "bin", "engram")
	bin := []byte("RAW-BINARY")

	a := &Applier{Fetcher: fakeFetcher{bodies: map[string][]byte{"https://x/engram": bin}}}
	d := diff.FileDiff{
		Path:   dest,
		Action: diff.Fetch,
		Fetch:  &diff.FetchSpec{URL: "https://x/engram", SHA256: sha(bin), Dest: dest, Label: "engram"},
	}
	res, err := a.Apply(cs(d))
	if err != nil {
		t.Fatal(err)
	}
	if read(t, dest) != "RAW-BINARY" {
		t.Errorf("placed content = %q", read(t, dest))
	}
	if len(res.Applied) != 1 {
		t.Errorf("applied = %d, want 1", len(res.Applied))
	}
	// Executable bit set.
	info, _ := os.Stat(dest)
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("binary not executable: %v", info.Mode())
	}
}

func TestApplyFetchArchive(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "bin", "engram")
	bin := "BINARY-IN-TARBALL"

	// Build a tar.gz containing engram-v1/engram.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "engram-v1/engram", Mode: 0o755, Size: int64(len(bin)), Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte(bin))
	tw.Close()
	gz.Close()
	tarball := buf.Bytes()

	a := &Applier{Fetcher: fakeFetcher{bodies: map[string][]byte{"https://x/engram.tgz": tarball}}}
	d := diff.FileDiff{
		Path:   dest,
		Action: diff.Fetch,
		Fetch: &diff.FetchSpec{
			URL: "https://x/engram.tgz", SHA256: sha(tarball), Dest: dest,
			Archive: "tar.gz", BinaryPath: "engram", Label: "engram",
		},
	}
	if _, err := a.Apply(cs(d)); err != nil {
		t.Fatal(err)
	}
	if read(t, dest) != bin {
		t.Errorf("extracted content = %q, want %q", read(t, dest), bin)
	}
}

func TestApplyFetchShaMismatchStops(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "bin", "engram")
	bin := []byte("RAW")

	a := &Applier{Fetcher: fakeFetcher{bodies: map[string][]byte{"https://x/e": bin}}}
	d := diff.FileDiff{
		Path:   dest,
		Action: diff.Fetch,
		Fetch:  &diff.FetchSpec{URL: "https://x/e", SHA256: "deadbeef", Dest: dest, Label: "engram"},
	}
	res, err := a.Apply(cs(d))
	if err == nil {
		t.Fatal("expected sha mismatch error")
	}
	if res.Failed == nil {
		t.Error("expected Failed to be set")
	}
	// Terraform-style: nothing placed on a verify failure.
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Error("binary should not be placed on sha mismatch")
	}
}

func TestApplyFetchNilFetcherFailsLoudly(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "bin", "engram")
	a := &Applier{} // no Fetcher
	d := diff.FileDiff{Path: dest, Action: diff.Fetch, Fetch: &diff.FetchSpec{URL: "https://x/e", Dest: dest}}
	if _, err := a.Apply(cs(d)); err == nil {
		t.Fatal("expected error for FETCH with nil fetcher")
	}
}

func TestApplySkipsExecDiffs(t *testing.T) {
	// EXEC rows are display-only for the applier — never run here.
	a := &Applier{}
	d := diff.FileDiff{Action: diff.Exec, Exec: &diff.ExecSpec{Command: []string{"true"}, Display: "true"}}
	res, err := a.Apply(cs(d))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Applied) != 0 {
		t.Errorf("EXEC should not be applied by the writer, got %d", len(res.Applied))
	}
	if len(res.Skipped) != 1 {
		t.Errorf("EXEC should be skipped, got %d", len(res.Skipped))
	}
}
