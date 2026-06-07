// Package archive is a small, dependency-free tar.gz / zip extractor. It is
// deliberately recipe-agnostic: Phase 4's recipe FETCH step uses it to pull a
// binary out of a release archive, and the later remote-registry phases (6–8)
// reuse the same surface to unpack scaffold tarballs into the registry cache.
// Keeping it standalone means neither caller depends on the other.
//
// Only the Go standard library is used (archive/tar, archive/zip,
// compress/gzip), so it works on every platform with zero prerequisites.
package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path"
	"strings"
)

// Supported format identifiers.
const (
	FormatTarGz = "tar.gz"
	FormatTgz   = "tgz"
	FormatZip   = "zip"
)

// DetectFormat infers an archive format from a filename or URL suffix. It
// returns "" when the name has no recognized archive extension (i.e. a raw,
// un-archived file). Query strings/fragments on a URL are ignored.
func DetectFormat(name string) string {
	n := strings.ToLower(name)
	if i := strings.IndexAny(n, "?#"); i >= 0 {
		n = n[:i]
	}
	switch {
	case strings.HasSuffix(n, ".tar.gz"):
		return FormatTarGz
	case strings.HasSuffix(n, ".tgz"):
		return FormatTgz
	case strings.HasSuffix(n, ".zip"):
		return FormatZip
	default:
		return ""
	}
}

// Extract reads an archive of the given format from r and returns every regular
// file member keyed by its (slash-separated, cleaned) path. Directories and
// non-regular entries are skipped. This is the "unpack everything" surface the
// remote registry uses for scaffold tarballs.
func Extract(r io.Reader, format string) (map[string][]byte, error) {
	switch format {
	case FormatTarGz, FormatTgz:
		return extractTarGz(r)
	case FormatZip:
		return extractZip(r)
	default:
		return nil, fmt.Errorf("archive: unsupported format %q", format)
	}
}

// ExtractFile returns a single member's bytes from an archive. memberPath is
// matched against each entry's cleaned path; a bare filename also matches an
// entry in any directory (so callers need not know the archive's internal
// layout, e.g. "engram" matches "engram-v0.4/engram"). It errors if the member
// is not found. This is the surface the recipe FETCH step uses to pull one
// binary out of a release archive.
func ExtractFile(r io.Reader, format, memberPath string) ([]byte, error) {
	files, err := Extract(r, format)
	if err != nil {
		return nil, err
	}
	want := path.Clean(memberPath)
	if b, ok := files[want]; ok {
		return b, nil
	}
	// Fall back to a basename match so a caller can name just the binary.
	base := path.Base(want)
	for name, b := range files {
		if path.Base(name) == base {
			return b, nil
		}
	}
	return nil, fmt.Errorf("archive: member %q not found", memberPath)
}

func extractTarGz(r io.Reader) (map[string][]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("archive: gzip: %w", err)
	}
	defer gz.Close()

	out := map[string][]byte{}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("archive: tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("archive: tar read %q: %w", hdr.Name, err)
		}
		out[path.Clean(hdr.Name)] = b
	}
	return out, nil
}

func extractZip(r io.Reader) (map[string][]byte, error) {
	// archive/zip needs a ReaderAt + size, so buffer the stream first. Release
	// archives are small (a single binary), so the cost is negligible.
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("archive: read zip: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("archive: zip: %w", err)
	}

	out := map[string][]byte{}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("archive: zip open %q: %w", f.Name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("archive: zip read %q: %w", f.Name, err)
		}
		out[path.Clean(f.Name)] = b
	}
	return out, nil
}
