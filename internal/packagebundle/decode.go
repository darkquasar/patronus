package packagebundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/darkquasar/patronus/internal/manifest"
)

const metadataLimit = 1 << 20

// Decode verifies one bounded gzip/USTAR archive and its exact metadata inventory.
// The caller must separately verify the outer archive digest.
func Decode(r io.Reader, expected Identity, limits Limits) (*Bundle, error) {
	if err := validateLimits(limits); err != nil {
		return nil, err
	}
	compressed, err := readBounded(r, limits.CompressedBytes)
	if err != nil {
		return nil, fmt.Errorf("compressed package: %w", err)
	}
	input := bytes.NewReader(compressed)
	gz, err := gzip.NewReader(input)
	if err != nil {
		return nil, fmt.Errorf("gzip header: %w", err)
	}
	gz.Multistream(false)
	raw, err := readBounded(gz, limits.DecodedBytes)
	closeErr := gz.Close()
	if err != nil {
		return nil, fmt.Errorf("gzip payload: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("gzip close: %w", closeErr)
	}
	if input.Len() != 0 {
		return nil, errors.New("trailing compressed data or second gzip stream")
	}
	if err := scanUSTAR(raw, limits); err != nil {
		return nil, err
	}
	tr := tar.NewReader(bytes.NewReader(raw))
	bundle := &Bundle{}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar header: %w", err)
		}
		if h.Format != tar.FormatUSTAR {
			return nil, errors.New("package requires USTAR headers")
		}
		if h.Typeflag == tar.TypeDir {
			continue
		}
		data, err := readBounded(tr, h.Size)
		if err != nil {
			return nil, fmt.Errorf("member %q: %w", h.Name, err)
		}
		mode := uint32(0644)
		if h.Mode&0111 != 0 {
			mode = 0755
		}
		bundle.Files = append(bundle.Files, File{Path: h.Name, Mode: mode, Data: data})
	}
	if err := verifyMetadata(bundle, expected, limits); err != nil {
		return nil, err
	}
	return bundle, nil
}
func validateLimits(l Limits) error {
	if l.CompressedBytes <= 0 || l.CompressedBytes > 32<<20 || l.ExpandedBytes <= 0 || l.ExpandedBytes > 128<<20 || l.FileBytes <= 0 || l.FileBytes > 16<<20 || l.DecodedBytes <= 0 || l.DecodedBytes > 136<<20 || l.Entries <= 0 || l.Entries > 4096 || l.Components <= 0 || l.Components > 16 || l.PathBytes <= 0 || l.PathBytes > 512 {
		return errors.New("invalid package limits")
	}
	return nil
}
func readBounded(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("byte limit %d exceeded", limit)
	}
	return b, nil
}

// scanUSTAR runs before archive/tar, which otherwise consumes extension headers invisibly.
func scanUSTAR(raw []byte, l Limits) error {
	nodes := make(map[string]pathNode)
	entries := 0
	var expanded int64
	for pos := int64(0); ; {
		if int64(len(raw))-pos < 512 {
			return errors.New("missing tar end blocks")
		}
		h := raw[pos : pos+512]
		if allZero(h) {
			if int64(len(raw))-pos < 1024 || !allZero(raw[pos:]) || (int64(len(raw))-pos)%512 != 0 {
				return errors.New("invalid tar ending or trailing payload")
			}
			return nil
		}
		entries++
		if entries > l.Entries {
			return errors.New("tar entry limit exceeded")
		}
		if string(h[257:263]) != "ustar\x00" || string(h[263:265]) != "00" {
			return errors.New("only USTAR headers are accepted")
		}
		sum, err := octal(h[148:156])
		if err != nil {
			return fmt.Errorf("tar checksum: %w", err)
		}
		var actual int64
		for i, b := range h {
			if i >= 148 && i < 156 {
				actual += 32
			} else {
				actual += int64(b)
			}
		}
		if sum != actual {
			return errors.New("invalid tar checksum")
		}
		kind := h[156]
		if kind != tar.TypeReg && kind != 0 && kind != tar.TypeDir {
			return fmt.Errorf("unsupported tar type %d", kind)
		}
		name, err := tarString(h[:100])
		if err != nil {
			return err
		}
		if _, err := validatePath(name, kind == tar.TypeDir, l.Components, l.PathBytes); err != nil {
			return err
		}
		prefix, err := tarString(h[345:500])
		if err != nil {
			return err
		}
		if prefix != "" {
			if _, err := validatePath(prefix, false, l.Components, l.PathBytes); err != nil {
				return err
			}
			name = prefix + "/" + name
		}
		name, err = validatePath(name, kind == tar.TypeDir, l.Components, l.PathBytes)
		if err != nil {
			return err
		}
		if err := addPath(nodes, name, kind == tar.TypeDir); err != nil {
			return err
		}
		mode, err := octal(h[100:108])
		if err != nil || mode&^0777 != 0 {
			return fmt.Errorf("invalid or privileged mode for %q", name)
		}
		size, err := octal(h[124:136])
		if err != nil {
			return fmt.Errorf("size for %q: %w", name, err)
		}
		if kind == tar.TypeDir && size != 0 {
			return errors.New("directory has payload")
		}
		if size > l.FileBytes || name == "package.json" && size > metadataLimit {
			return fmt.Errorf("member %q exceeds file limit", name)
		}
		expanded += size
		if expanded > l.ExpandedBytes {
			return errors.New("expanded file limit exceeded")
		}
		padded := (size + 511) / 512 * 512
		if padded > int64(len(raw))-pos-512 {
			return errors.New("truncated tar payload")
		}
		pos += 512 + padded
	}
}
func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}
func tarString(b []byte) (string, error) {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		if !allZero(b[i:]) {
			return "", errors.New("nonzero bytes after tar name terminator")
		}
		b = b[:i]
	}
	return string(b), nil
}
func octal(b []byte) (int64, error) {
	s := strings.Trim(string(b), " \x00")
	if s == "" {
		return 0, nil
	}
	for _, c := range s {
		if c < '0' || c > '7' {
			return 0, errors.New("invalid octal field")
		}
	}
	return strconv.ParseInt(s, 8, 64)
}
func verifyMetadata(b *Bundle, expected Identity, l Limits) error {
	files := make(map[string]File, len(b.Files))
	var metadata []byte
	for _, f := range b.Files {
		files[f.Path] = f
		if f.Path == "package.json" {
			metadata = f.Data
		}
	}
	if metadata == nil {
		return errors.New("missing package.json")
	}
	decoder := json.NewDecoder(bytes.NewReader(metadata))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&b.Metadata); err != nil {
		return fmt.Errorf("package.json: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("trailing package.json content")
	}
	m := b.Metadata
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported package schema %d", m.SchemaVersion)
	}
	if !validIdentity(m.Identity) || m.Identity != expected {
		return errors.New("invalid or unexpected package identity")
	}
	if !validDigest(m.SourceSHA256) {
		return errors.New("invalid source SHA256")
	}
	if len(m.Files) != len(files)-1 {
		return errors.New("package inventory does not match payload set")
	}
	previous := ""
	for _, e := range m.Files {
		if _, err := validatePath(e.Path, false, l.Components, l.PathBytes); err != nil {
			return err
		}
		if e.Path == "package.json" || e.Path <= previous {
			return errors.New("inventory must be sorted, unique, and exclude package.json")
		}
		previous = e.Path
		f, ok := files[e.Path]
		if !ok {
			return fmt.Errorf("missing payload %q", e.Path)
		}
		if (e.Mode != 0644 && e.Mode != 0755) || e.Mode != f.Mode {
			return fmt.Errorf("payload mode mismatch for %q", e.Path)
		}
		if !validDigest(e.SHA256) || e.SHA256 != fmt.Sprintf("sha256:%x", sha256.Sum256(f.Data)) {
			return fmt.Errorf("payload digest mismatch for %q", e.Path)
		}
	}
	return nil
}
func validDigest(s string) bool {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") {
		return false
	}
	for _, c := range s[7:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func validIdentity(i Identity) bool {
	return validName(i.Name) && manifest.ValidPackageVersion(i.Version) && (i.OS == "linux" || i.OS == "darwin" || i.OS == "windows") && (i.Arch == "amd64" || i.Arch == "arm64")
}

func validName(name string) bool {
	if name == "" || !(name[0] >= 'a' && name[0] <= 'z' || name[0] >= '0' && name[0] <= '9') {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '.' || c == '-') {
			return false
		}
	}
	return true
}
