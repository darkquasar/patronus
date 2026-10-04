package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"testing"
)

func dp06Archive(t *testing.T, format string, names []string, bodies []string) []byte {
	t.Helper()
	var out bytes.Buffer
	if format == FormatZip {
		zw := zip.NewWriter(&out)
		for i, name := range names {
			w, err := zw.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte(bodies[i])); err != nil {
				t.Fatal(err)
			}
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		gz := gzip.NewWriter(&out)
		tw := tar.NewWriter(gz)
		for i, name := range names {
			if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(bodies[i]))}); err != nil {
				t.Fatal(err)
			}
			if _, err := tw.Write([]byte(bodies[i])); err != nil {
				t.Fatal(err)
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return out.Bytes()
}
func dp06Limits() Limits {
	return Limits{CompressedBytes: 1 << 20, DecodedBytes: 1 << 20, ExpandedBytes: 1000, FileBytes: 100, Entries: 10}
}

func TestExtractFileBounded(t *testing.T) {
	for _, format := range []string{FormatTarGz, FormatZip} {
		t.Run(format, func(t *testing.T) {
			input := dp06Archive(t, format, []string{"release/bin", "notes"}, []string{"inert", "documentation"})
			for _, member := range []string{"bin", "release/bin"} {
				got, err := ExtractFileBounded(bytes.NewReader(input), format, member, dp06Limits())
				if err != nil || string(got) != "inert" {
					t.Fatalf("safe selection %s: %q %v", member, got, err)
				}
			}
			for _, bound := range []string{"compressed", "decoded", "expanded", "member", "entries"} {
				if format == FormatZip && bound == "decoded" {
					continue
				}
				t.Run(bound, func(t *testing.T) {
					l := dp06Limits()
					switch bound {
					case "compressed":
						l.CompressedBytes = 10
					case "decoded":
						l.DecodedBytes = 1024
					case "expanded":
						l.ExpandedBytes = 6
					case "member":
						l.FileBytes = 6
					case "entries":
						l.Entries = 1
					}
					if _, err := ExtractFileBounded(bytes.NewReader(input), format, "bin", l); err == nil {
						t.Fatal("accepted exceeded limit")
					}
				})
			}
			for _, kind := range []string{"truncated", "corrupt", "missing", "ambiguous", "unsafe", "duplicate"} {
				t.Run(kind, func(t *testing.T) {
					data := append([]byte(nil), input...)
					member := "bin"
					switch kind {
					case "truncated":
						data = data[:len(data)/2]
					case "corrupt":
						data[0] ^= 0xff
					case "missing":
						member = "absent"
					case "ambiguous":
						data = dp06Archive(t, format, []string{"a/bin", "b/bin"}, []string{"one", "two"})
					case "unsafe":
						data = dp06Archive(t, format, []string{"release/bin", "../unsafe"}, []string{"one", "two"})
					case "duplicate":
						data = dp06Archive(t, format, []string{"bin", "bin"}, []string{"one", "two"})
					}
					if _, err := ExtractFileBounded(bytes.NewReader(data), format, member, dp06Limits()); err == nil {
						t.Fatal("accepted invalid archive")
					}
				})
			}
		})
	}
}

func TestExtractFileBoundedCountsHiddenMetadata(t *testing.T) {
	var out bytes.Buffer
	gz := gzip.NewWriter(&out)
	tw := tar.NewWriter(gz)
	h := &tar.Header{Name: "bin", Mode: 0644, Size: 3, Format: tar.FormatPAX, PAXRecords: map[string]string{"comment": "inert metadata"}}
	if err := tw.WriteHeader(h); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	l := dp06Limits()
	l.Entries = 1
	if _, err := ExtractFileBounded(bytes.NewReader(out.Bytes()), FormatTarGz, "bin", l); err == nil {
		t.Fatal("hidden PAX header escaped entry bound")
	}
	// A forged EOCD count must not permit allocation for extra central rows.
	data := dp06Archive(t, FormatZip, []string{"bin", "extra"}, []string{"one", "two"})
	end := bytes.LastIndex(data, []byte{'P', 'K', 5, 6})
	binary.LittleEndian.PutUint16(data[end+8:end+10], 1)
	binary.LittleEndian.PutUint16(data[end+10:end+12], 1)
	if _, err := ExtractFileBounded(bytes.NewReader(data), FormatZip, "bin", dp06Limits()); err == nil {
		t.Fatal("forged zip count admitted")
	}
}
