package packagebundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// Build produces a deterministic package and verifies it with the delivery decoder.
func Build(identity Identity, sourceSHA256 string, files []File) ([]byte, error) {
	if !validIdentity(identity) || !validDigest(sourceSHA256) {
		return nil, errors.New("invalid package identity or source digest")
	}
	if len(files) >= DefaultLimits.Entries {
		return nil, errors.New("package entry limit exceeded")
	}
	sorted := append([]File(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	metadata := Metadata{SchemaVersion: 1, Identity: identity, SourceSHA256: sourceSHA256, Files: make([]Entry, 0, len(sorted))}
	nodes := make(map[string]pathNode)
	if err := addPath(nodes, "package.json", false); err != nil {
		return nil, err
	}
	var expanded int64
	for _, f := range sorted {
		if _, err := ValidatePath(f.Path, false); err != nil {
			return nil, err
		}
		if err := addPath(nodes, f.Path, false); err != nil {
			return nil, err
		}
		if f.Mode != 0644 && f.Mode != 0755 {
			return nil, fmt.Errorf("invalid mode for %q", f.Path)
		}
		expanded += int64(len(f.Data))
		if int64(len(f.Data)) > DefaultLimits.FileBytes || expanded > DefaultLimits.ExpandedBytes {
			return nil, errors.New("package file size limit exceeded")
		}
		metadata.Files = append(metadata.Files, Entry{Path: f.Path, Mode: f.Mode, SHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(f.Data))})
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	if len(data) > metadataLimit {
		return nil, errors.New("package metadata limit exceeded")
	}
	members := append([]File{{Path: "package.json", Mode: 0644, Data: data}}, sorted...)
	var output bytes.Buffer
	gz := gzip.NewWriter(&output)
	tw := tar.NewWriter(gz)
	for _, f := range members {
		h := &tar.Header{Name: f.Path, Mode: int64(f.Mode), Size: int64(len(f.Data)), Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}
		if err := tw.WriteHeader(h); err != nil {
			return nil, fmt.Errorf("package USTAR header %q: %w", f.Path, err)
		}
		if _, err := tw.Write(f.Data); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	if _, err := Decode(bytes.NewReader(output.Bytes()), identity, DefaultLimits); err != nil {
		return nil, fmt.Errorf("verify built package: %w", err)
	}
	return output.Bytes(), nil
}
