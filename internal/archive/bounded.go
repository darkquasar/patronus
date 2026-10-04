package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

// Limits bounds acquisition and decoding, including unselected members and tar
// metadata headers. Callers choose their delivery policy; this package owns no
// installer state or package policy.
type Limits struct {
	CompressedBytes, DecodedBytes, ExpandedBytes, FileBytes int64
	Entries                                                 int
}

// ReadBounded refuses oversized input before allocating beyond limit+1 bytes.
func ReadBounded(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 || limit >= 1<<40 {
		return nil, fmt.Errorf("archive: invalid byte limit")
	}
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("archive: byte limit %d exceeded", limit)
	}
	return b, nil
}

// ExtractFileBounded retains exact-path then unique-basename selection, but
// rejects ambiguous, unsafe, duplicate and nonregular members. It validates the
// whole archive, not just the selected member. No member is ever executed.
func ExtractFileBounded(r io.Reader, format, member string, limits Limits) ([]byte, error) {
	if limits.Entries <= 0 || limits.FileBytes <= 0 || limits.ExpandedBytes <= 0 || limits.DecodedBytes <= 0 {
		return nil, fmt.Errorf("archive: invalid limits")
	}
	compressed, err := ReadBounded(r, limits.CompressedBytes)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	seen := map[string]bool{}
	var expanded int64
	add := func(name string, dir bool, size int64, r io.Reader) error {
		name = strings.TrimSuffix(name, "/")
		if name == "" || path.IsAbs(name) || path.Clean(name) != name || strings.ContainsAny(name, "\\:\x00") || name == ".." || strings.HasPrefix(name, "../") {
			return fmt.Errorf("archive: unsafe member %q", name)
		}
		if seen[name] {
			return fmt.Errorf("archive: duplicate member %q", name)
		}
		seen[name] = true
		if size < 0 || size > limits.FileBytes || size > limits.ExpandedBytes-expanded {
			return fmt.Errorf("archive: member/expanded limit exceeded by %s", name)
		}
		expanded += size
		if dir {
			if size != 0 {
				return fmt.Errorf("archive: directory payload")
			}
			return nil
		}
		data, err := ReadBounded(r, limits.FileBytes)
		if err != nil {
			return err
		}
		if int64(len(data)) != size {
			return fmt.Errorf("archive: truncated member %s", name)
		}
		files[name] = data
		return nil
	}
	switch format {
	case FormatTarGz, FormatTgz:
		input := bytes.NewReader(compressed)
		gz, err := gzip.NewReader(input)
		if err != nil {
			return nil, err
		}
		gz.Multistream(false)
		raw, err := ReadBounded(gz, limits.DecodedBytes)
		closeErr := gz.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if input.Len() != 0 {
			return nil, fmt.Errorf("archive: trailing compressed data")
		}
		// archive/tar hides PAX/GNU metadata from Next. Count and bound those raw
		// records before passing any bytes to its decoder.
		if err := boundedTarHeaders(raw, limits); err != nil {
			return nil, err
		}
		tr := tar.NewReader(bytes.NewReader(raw))
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeDir {
				return nil, fmt.Errorf("archive: unsupported member type %d", h.Typeflag)
			}
			for key := range h.PAXRecords {
				if strings.HasPrefix(key, "GNU.sparse") {
					return nil, fmt.Errorf("archive: sparse member unsupported")
				}
			}
			if err := add(h.Name, h.Typeflag == tar.TypeDir, h.Size, tr); err != nil {
				return nil, err
			}
		}
	case FormatZip:
		// Bound central-directory allocation before archive/zip constructs File rows.
		end := bytes.LastIndex(compressed, []byte{'P', 'K', 5, 6})
		if end < 0 || len(compressed)-end < 22 {
			return nil, fmt.Errorf("archive: missing zip end record")
		}
		e := compressed[end:]
		count := int(binary.LittleEndian.Uint16(e[10:12]))
		if count == 65535 || count > limits.Entries || binary.LittleEndian.Uint16(e[4:6]) != 0 || binary.LittleEndian.Uint16(e[6:8]) != 0 {
			return nil, fmt.Errorf("archive: zip entry limit, zip64 or multipart unsupported")
		}
		if int(binary.LittleEndian.Uint16(e[20:22])) != len(e)-22 {
			return nil, fmt.Errorf("archive: invalid zip ending")
		}
		if err := boundedZipDirectory(compressed, e, count); err != nil {
			return nil, err
		}
		zr, err := zip.NewReader(bytes.NewReader(compressed), int64(len(compressed)))
		if err != nil {
			return nil, err
		}
		if len(zr.File) != count || len(zr.File) > limits.Entries {
			return nil, fmt.Errorf("archive: zip entry count mismatch")
		}
		for _, f := range zr.File {
			if !f.Mode().IsRegular() && !f.FileInfo().IsDir() {
				return nil, fmt.Errorf("archive: nonregular member %s", f.Name)
			}
			if f.UncompressedSize64 > uint64(limits.FileBytes) {
				return nil, fmt.Errorf("archive: member limit exceeded")
			}
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			err = add(f.Name, f.FileInfo().IsDir(), int64(f.UncompressedSize64), rc)
			closeErr := rc.Close()
			if err != nil {
				return nil, err
			}
			if closeErr != nil {
				return nil, closeErr
			}
		}
	default:
		return nil, fmt.Errorf("archive: unsupported format %q", format)
	}
	if data, ok := files[member]; ok {
		return data, nil
	}
	var found []byte
	matches := 0
	for name, data := range files {
		if path.Base(name) == path.Base(member) {
			found = data
			matches++
		}
	}
	if matches != 1 {
		return nil, fmt.Errorf("archive: member %q missing or ambiguous (%d matches)", member, matches)
	}
	return found, nil
}

func boundedTarHeaders(raw []byte, l Limits) error {
	count := 0
	var expanded int64
	for pos := int64(0); ; {
		if int64(len(raw))-pos < 512 {
			return fmt.Errorf("archive: truncated tar headers")
		}
		h := raw[pos : pos+512]
		if bytes.Equal(h, make([]byte, 512)) {
			tail := raw[pos:]
			if len(tail) < 1024 || len(tail)%512 != 0 || !bytes.Equal(tail, make([]byte, len(tail))) {
				return fmt.Errorf("archive: invalid tar ending")
			}
			return nil
		}
		count++
		if count > l.Entries {
			return fmt.Errorf("archive: entry limit exceeded")
		}
		// POSIX octal sizes are sufficient for the admitted bounded delivery size.
		size, err := strconv.ParseInt(strings.Trim(string(h[124:136]), " \x00"), 8, 64)
		if err != nil || size < 0 || size > l.FileBytes || size > l.ExpandedBytes-expanded {
			return fmt.Errorf("archive: invalid size or member/expanded limit")
		}
		expanded += size
		padded := (size + 511) / 512 * 512
		if padded > int64(len(raw))-pos-512 {
			return fmt.Errorf("archive: truncated tar member")
		}
		pos += 512 + padded
	}
}

// Check every central-directory header before archive/zip allocates its File
// inventory. Count fields alone can lie; variable-length metadata is included.
func boundedZipDirectory(data, end []byte, count int) error {
	offset := int64(binary.LittleEndian.Uint32(end[16:20]))
	size := int64(binary.LittleEndian.Uint32(end[12:16]))
	stop := offset + size
	if offset < 0 || stop > int64(len(data)-len(end)) {
		return fmt.Errorf("archive: invalid zip directory range")
	}
	seen := 0
	for offset < stop {
		if stop-offset < 46 || !bytes.Equal(data[offset:offset+4], []byte{'P', 'K', 1, 2}) {
			return fmt.Errorf("archive: invalid zip directory")
		}
		h := data[offset : offset+46]
		seen++
		if seen > count {
			return fmt.Errorf("archive: zip entry count exceeded")
		}
		width := int64(46) + int64(binary.LittleEndian.Uint16(h[28:30])) + int64(binary.LittleEndian.Uint16(h[30:32])) + int64(binary.LittleEndian.Uint16(h[32:34]))
		if width > stop-offset {
			return fmt.Errorf("archive: truncated zip metadata")
		}
		offset += width
	}
	if seen != count {
		return fmt.Errorf("archive: zip entry count mismatch")
	}
	return nil
}
