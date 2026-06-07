package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"testing"
)

// makeTarGz builds an in-memory .tar.gz from name->content pairs.
func makeTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// makeZip builds an in-memory .zip from name->content pairs.
func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDetectFormat(t *testing.T) {
	cases := map[string]string{
		"engram-v0.4-linux-amd64.tar.gz":           FormatTarGz,
		"engram.tgz":                               FormatTgz,
		"engram-windows.zip":                       FormatZip,
		"https://x/y/engram.tar.gz?token=abc#frag": FormatTarGz,
		"engram":     "", // raw binary, no archive
		"engram.exe": "",
	}
	for name, want := range cases {
		if got := DetectFormat(name); got != want {
			t.Errorf("DetectFormat(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestExtractTarGzRoundTrip(t *testing.T) {
	data := makeTarGz(t, map[string]string{
		"engram-v0.4/engram":    "BINARY",
		"engram-v0.4/README.md": "docs",
	})
	files, err := Extract(bytes.NewReader(data), FormatTarGz)
	if err != nil {
		t.Fatal(err)
	}
	if string(files["engram-v0.4/engram"]) != "BINARY" {
		t.Errorf("got %q", files["engram-v0.4/engram"])
	}
	if len(files) != 2 {
		t.Errorf("expected 2 members, got %d", len(files))
	}
}

func TestExtractZipRoundTrip(t *testing.T) {
	data := makeZip(t, map[string]string{"bin/engram.exe": "WINBIN"})
	files, err := Extract(bytes.NewReader(data), FormatZip)
	if err != nil {
		t.Fatal(err)
	}
	if string(files["bin/engram.exe"]) != "WINBIN" {
		t.Errorf("got %q", files["bin/engram.exe"])
	}
}

func TestExtractFileBasenameMatch(t *testing.T) {
	// A caller naming just the binary should match it inside any directory.
	data := makeTarGz(t, map[string]string{"engram-v0.4/engram": "BINARY"})
	b, err := ExtractFile(bytes.NewReader(data), FormatTarGz, "engram")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "BINARY" {
		t.Errorf("got %q", b)
	}
}

func TestExtractFileExactMatch(t *testing.T) {
	data := makeZip(t, map[string]string{"a/engram": "A", "b/engram": "B"})
	b, err := ExtractFile(bytes.NewReader(data), FormatZip, "b/engram")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "B" {
		t.Errorf("got %q, want B", b)
	}
}

func TestExtractFileMissingMember(t *testing.T) {
	data := makeTarGz(t, map[string]string{"engram-v0.4/engram": "BINARY"})
	if _, err := ExtractFile(bytes.NewReader(data), FormatTarGz, "nope"); err == nil {
		t.Fatal("expected error for missing member")
	}
}

func TestExtractCorruptStream(t *testing.T) {
	if _, err := Extract(bytes.NewReader([]byte("not a gzip stream")), FormatTarGz); err == nil {
		t.Fatal("expected error for corrupt gzip")
	}
	if _, err := Extract(bytes.NewReader([]byte("not a zip")), FormatZip); err == nil {
		t.Fatal("expected error for corrupt zip")
	}
}

func TestExtractUnsupportedFormat(t *testing.T) {
	if _, err := Extract(bytes.NewReader(nil), "rar"); err == nil {
		t.Fatal("expected error for unsupported format")
	}
}
