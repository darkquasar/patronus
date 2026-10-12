package install

import (
	"archive/tar"
	"archive/zip"
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

// fetchBinaryBody wraps invented bytes in the supported executable deliveries.
func fetchBinaryBody(t *testing.T, format string, bin []byte) []byte {
	t.Helper()
	if format == "" {
		return bin
	}
	var buf bytes.Buffer
	if format == "zip" {
		zw := zip.NewWriter(&buf)
		w, err := zw.Create("fixture")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(bin); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		if err := tw.WriteHeader(&tar.Header{Name: "fixture", Mode: 0o755, Size: int64(len(bin)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(bin); err != nil {
			t.Fatal(err)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return buf.Bytes()
}

func TestFetchBinaryAboveDirectoryLimit(t *testing.T) {
	bin := bytes.Repeat([]byte("x"), (16<<20)+1)
	for _, format := range []string{"", "tar.gz", "zip"} {
		t.Run("format="+format, func(t *testing.T) {
			body := fetchBinaryBody(t, format, bin)
			dest := filepath.Join(t.TempDir(), "fixture")
			spec := &diff.FetchSpec{URL: "https://fixture.invalid/binary", SHA256: sha(body), Dest: dest, Archive: format, BinaryPath: "fixture"}
			fetcher := fakeFetcher{bodies: map[string][]byte{spec.URL: body}}
			acquired, err := AcquireFetch(context.Background(), spec, fetcher)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(acquired, body) {
				t.Fatal("acquisition changed verified delivery bytes")
			}
			a := &Applier{Fetcher: fetcher}
			res, err := a.Apply(cs(diff.FileDiff{Path: dest, Action: diff.Fetch, Fetch: spec}))
			if err != nil {
				t.Fatal(err)
			}
			placed, err := os.ReadFile(dest)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(placed, bin) || spec.PlacedSHA256 != sha(bin) || len(res.Applied) != 1 {
				t.Fatal("FETCH did not place and record the verified binary")
			}
		})
	}
}

func TestAcquireFetchBinaryAtLimit(t *testing.T) {
	bin := bytes.Repeat([]byte("x"), 32<<20)
	for _, format := range []string{"", "tar.gz", "zip"} {
		t.Run("format="+format, func(t *testing.T) {
			body := fetchBinaryBody(t, format, bin)
			spec := &diff.FetchSpec{URL: "https://fixture.invalid/binary", SHA256: sha(body), Archive: format, BinaryPath: "fixture"}
			if _, err := AcquireFetch(context.Background(), spec, fakeFetcher{bodies: map[string][]byte{spec.URL: body}}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFetchBinaryAboveLimitStops(t *testing.T) {
	bin := bytes.Repeat([]byte("x"), (32<<20)+1)
	for _, format := range []string{"", "tar.gz", "zip"} {
		t.Run("format="+format, func(t *testing.T) {
			body := fetchBinaryBody(t, format, bin)
			dest := filepath.Join(t.TempDir(), "fixture")
			spec := &diff.FetchSpec{URL: "https://fixture.invalid/binary", SHA256: sha(body), Dest: dest, Archive: format, BinaryPath: "fixture"}
			fetcher := fakeFetcher{bodies: map[string][]byte{spec.URL: body}}
			if _, err := AcquireFetch(context.Background(), spec, fetcher); err == nil {
				t.Fatal("acquisition accepted an oversized binary")
			}
			a := &Applier{Fetcher: fetcher}
			res, err := a.Apply(cs(diff.FileDiff{Path: dest, Action: diff.Fetch, Fetch: spec}))
			if err == nil || res.Failed == nil || len(res.Applied) != 0 || spec.PlacedSHA256 != "" {
				t.Fatal("oversized FETCH did not fail without recording a placement")
			}
			if _, err := os.Stat(dest); !os.IsNotExist(err) {
				t.Fatal("oversized FETCH wrote its destination")
			}
		})
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
