package packagebundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
)

type member struct {
	name string
	mode int64
	kind byte
	data []byte
}

func testIdentity() Identity {
	return Identity{Name: "kit", Version: "1.0.0", OS: "linux", Arch: "amd64"}
}
func digest(data []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(data)) }
func fixture(t *testing.T) []member {
	t.Helper()
	data := []byte("echo must-not-run\n")
	m := Metadata{SchemaVersion: 1, Identity: testIdentity(), SourceSHA256: digest([]byte("source")), Files: []Entry{{Path: "install.sh", Mode: 0755, SHA256: digest(data)}}}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return []member{{"package.json", 0644, tar.TypeReg, b}, {"install.sh", 0755, tar.TypeReg, data}}
}
func rawTar(t *testing.T, members []member) []byte {
	t.Helper()
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	for _, m := range members {
		h := &tar.Header{Name: m.name, Mode: m.mode, Typeflag: m.kind, Size: int64(len(m.data)), Format: tar.FormatUSTAR}
		if m.kind == tar.TypeSymlink || m.kind == tar.TypeLink {
			h.Linkname = "target"
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(m.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func zipTar(t *testing.T, b []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	z := gzip.NewWriter(&out)
	if _, err := z.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
func TestDecodeVerifiedBytes(t *testing.T) {
	b, err := Decode(bytes.NewReader(zipTar(t, rawTar(t, fixture(t)))), testIdentity(), DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Files) != 2 || b.Files[1].Path != "install.sh" || string(b.Files[1].Data) != "echo must-not-run\n" || b.Files[1].Mode != 0755 {
		t.Fatalf("unexpected bundle: %+v", b)
	}
}
func TestDecodeHostileMembers(t *testing.T) {
	for _, m := range []member{{"link", 0777, tar.TypeSymlink, nil}, {"hard", 0644, tar.TypeLink, nil}, {"dev", 0644, tar.TypeChar, nil}, {"block", 0644, tar.TypeBlock, nil}, {"fifo", 0644, tar.TypeFifo, nil}, {"install.sh", 0755, tar.TypeReg, nil}, {"INSTALL.SH", 0644, tar.TypeReg, nil}, {"install.sh/x", 0644, tar.TypeReg, nil}, {"package.json", 0644, tar.TypeReg, nil}, {"a", 04755, tar.TypeReg, nil}, {"../x", 0644, tar.TypeReg, nil}, {"a//b", 0644, tar.TypeReg, nil}, {"extra", 0644, tar.TypeReg, nil}} {
		t.Run(fmt.Sprintf("%s-%d", m.name, m.kind), func(t *testing.T) {
			members := append(fixture(t), m)
			if _, err := Decode(bytes.NewReader(zipTar(t, rawTar(t, members))), testIdentity(), DefaultLimits); err == nil {
				t.Fatal("accepted hostile member")
			}
		})
	}
}
func TestDecodeMetadata(t *testing.T) {
	for _, change := range []func(*Metadata){func(m *Metadata) { m.SchemaVersion = 2 }, func(m *Metadata) { m.Identity.Version = "2.0.0" }, func(m *Metadata) { m.Files = nil }, func(m *Metadata) { m.Files[0].SHA256 = digest(nil) }, func(m *Metadata) { m.Files[0].Mode = 0644 }, func(m *Metadata) { m.Files = append(m.Files, m.Files[0]) }, func(m *Metadata) { m.SourceSHA256 = "bad" }} {
		members := fixture(t)
		var m Metadata
		if err := json.Unmarshal(members[0].data, &m); err != nil {
			t.Fatal(err)
		}
		change(&m)
		members[0].data, _ = json.Marshal(m)
		if _, err := Decode(bytes.NewReader(zipTar(t, rawTar(t, members))), testIdentity(), DefaultLimits); err == nil {
			t.Fatal("accepted malformed metadata")
		}
	}
	for _, data := range []string{"null", "{}", "{", string(fixture(t)[0].data) + " {}"} {
		members := fixture(t)
		members[0].data = []byte(data)
		if _, err := Decode(bytes.NewReader(zipTar(t, rawTar(t, members))), testIdentity(), DefaultLimits); err == nil {
			t.Fatal("accepted invalid JSON")
		}
	}
}
func TestDecodeLimits(t *testing.T) {
	b := zipTar(t, rawTar(t, fixture(t)))
	for _, change := range []func(*Limits){func(l *Limits) { l.CompressedBytes = int64(len(b) - 1) }, func(l *Limits) { l.DecodedBytes = 512 }, func(l *Limits) { l.FileBytes = 10 }, func(l *Limits) { l.ExpandedBytes = 10 }, func(l *Limits) { l.Entries = 1 }, func(l *Limits) { l.PathBytes = 5 }} {
		l := DefaultLimits
		change(&l)
		if _, err := Decode(bytes.NewReader(b), testIdentity(), l); err == nil {
			t.Fatal("accepted over budget")
		}
	}
}

func checksumHeader(h []byte) {
	for i := 148; i < 156; i++ {
		h[i] = ' '
	}
	sum := 0
	for _, b := range h {
		sum += int(b)
	}
	copy(h[148:156], fmt.Sprintf("%06o\x00 ", sum))
}
func TestDecodeRawHeaderAndStreamAttacks(t *testing.T) {
	base := rawTar(t, fixture(t))
	for name, mutate := range map[string]func([]byte) []byte{
		"pax":           func(b []byte) []byte { b[156] = tar.TypeXHeader; checksumHeader(b[:512]); return b },
		"global-pax":    func(b []byte) []byte { b[156] = tar.TypeXGlobalHeader; checksumHeader(b[:512]); return b },
		"gnu-name":      func(b []byte) []byte { b[156] = tar.TypeGNULongName; checksumHeader(b[:512]); return b },
		"gnu-link":      func(b []byte) []byte { b[156] = tar.TypeGNULongLink; checksumHeader(b[:512]); return b },
		"unknown":       func(b []byte) []byte { b[156] = 'Z'; checksumHeader(b[:512]); return b },
		"bad-checksum":  func(b []byte) []byte { b[0] = 'z'; return b },
		"bad-magic":     func(b []byte) []byte { b[257] = 'x'; checksumHeader(b[:512]); return b },
		"base256-size":  func(b []byte) []byte { b[124] = 0x80; checksumHeader(b[:512]); return b },
		"negative-size": func(b []byte) []byte { copy(b[124:136], "-0000000001\x00"); checksumHeader(b[:512]); return b },
		"huge-size":     func(b []byte) []byte { copy(b[124:136], "77777777777\x00"); checksumHeader(b[:512]); return b },
		"name-tail":     func(b []byte) []byte { b[90] = 'x'; checksumHeader(b[:512]); return b },
		"unsafe-prefix": func(b []byte) []byte { copy(b[345:500], "../"); checksumHeader(b[:512]); return b },
		"absolute-name-with-prefix": func(b []byte) []byte {
			copy(b[:100], make([]byte, 100))
			copy(b[:100], "/x")
			copy(b[345:500], "dir")
			checksumHeader(b[:512])
			return b
		},
		"one-end-block":    func(b []byte) []byte { return b[:len(b)-512] },
		"no-end-blocks":    func(b []byte) []byte { return b[:len(b)-1024] },
		"trailing-payload": func(b []byte) []byte { return append(b, bytes.Repeat([]byte{1}, 512)...) },
		"second-tar":       func(b []byte) []byte { return append(b, base...) },
		"partial-padding":  func(b []byte) []byte { return append(b, 0) },
	} {
		t.Run(name, func(t *testing.T) {
			raw := mutate(bytes.Clone(base))
			if _, err := Decode(bytes.NewReader(zipTar(t, raw)), testIdentity(), DefaultLimits); err == nil {
				t.Fatal("accepted hostile raw archive")
			}
		})
	}
	compressed := zipTar(t, base)
	for name, b := range map[string][]byte{"second-gzip": append(bytes.Clone(compressed), compressed...), "trailing-byte": append(bytes.Clone(compressed), 0), "truncated-gzip": compressed[:len(compressed)-1]} {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode(bytes.NewReader(b), testIdentity(), DefaultLimits); err == nil {
				t.Fatal("accepted malformed stream")
			}
		})
	}
	compressed[len(compressed)-8] ^= 1
	if _, err := Decode(bytes.NewReader(compressed), testIdentity(), DefaultLimits); err == nil {
		t.Fatal("accepted invalid gzip checksum")
	}
}
func TestDecodeDirectoriesAndNormalization(t *testing.T) {
	members := fixture(t)
	members[1].name = "config/install.sh"
	members[1].mode = 0700
	var m Metadata
	if err := json.Unmarshal(members[0].data, &m); err != nil {
		t.Fatal(err)
	}
	m.Files[0].Path = members[1].name
	members[0].data, _ = json.Marshal(m)
	members = append(members, member{"config/", 0700, tar.TypeDir, nil})
	b, err := Decode(bytes.NewReader(zipTar(t, rawTar(t, members))), testIdentity(), DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Files) != 2 || b.Files[1].Mode != 0755 {
		t.Fatalf("bad normalization: %+v", b)
	}
	for _, extra := range []member{{"Config/other", 0644, tar.TypeReg, nil}, {"config", 0755, tar.TypeDir, nil}, {"config", 0644, tar.TypeReg, nil}} {
		if _, err := Decode(bytes.NewReader(zipTar(t, rawTar(t, append(members, extra)))), testIdentity(), DefaultLimits); err == nil {
			t.Fatal("accepted directory collision")
		}
	}
	l := DefaultLimits
	l.Components = 1
	if _, err := Decode(bytes.NewReader(zipTar(t, rawTar(t, members))), testIdentity(), l); err == nil {
		t.Fatal("accepted component overflow")
	}
}
func TestDecodeMetadataBudgetAndMissingFile(t *testing.T) {
	members := fixture(t)
	members[0].data = bytes.Repeat([]byte{' '}, (1<<20)+1)
	if _, err := Decode(bytes.NewReader(zipTar(t, rawTar(t, members))), testIdentity(), DefaultLimits); err == nil {
		t.Fatal("accepted metadata overflow")
	}
	for _, members := range [][]member{fixture(t)[:1], fixture(t)[1:]} {
		if _, err := Decode(bytes.NewReader(zipTar(t, rawTar(t, members))), testIdentity(), DefaultLimits); err == nil {
			t.Fatal("accepted missing member")
		}
	}
}
func TestDecodePAXFloodBounded(t *testing.T) {
	var raw bytes.Buffer
	tw := tar.NewWriter(&raw)
	h := &tar.Header{Name: "install.sh", Mode: 0755, Typeflag: tar.TypeReg, Format: tar.FormatPAX, PAXRecords: map[string]string{"comment": string(bytes.Repeat([]byte{'x'}, 4096))}}
	if err := tw.WriteHeader(h); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	l := DefaultLimits
	l.DecodedBytes = 2048
	if _, err := Decode(bytes.NewReader(zipTar(t, raw.Bytes())), testIdentity(), l); err == nil {
		t.Fatal("accepted metadata flood")
	}
	if _, err := Decode(bytes.NewReader(zipTar(t, raw.Bytes())), testIdentity(), DefaultLimits); err == nil {
		t.Fatal("accepted valid PAX extension")
	}
}

func TestDecodeTrailingExtensionAndSmallPAX(t *testing.T) {
	base := rawTar(t, fixture(t))
	for _, kind := range []byte{tar.TypeXHeader, tar.TypeXGlobalHeader, tar.TypeGNULongName, tar.TypeGNULongLink} {
		t.Run(fmt.Sprintf("type-%d", kind), func(t *testing.T) {
			extension := rawTar(t, []member{{"extension", 0644, tar.TypeReg, []byte("12 path=x.y\n")}})
			extension[156] = kind
			checksumHeader(extension[:512])
			for _, raw := range [][]byte{append(bytes.Clone(extension[:len(extension)-1024]), base...), append(bytes.Clone(base[:len(base)-1024]), extension...)} {
				if _, err := Decode(bytes.NewReader(zipTar(t, raw)), testIdentity(), DefaultLimits); err == nil {
					t.Fatal("accepted extension header")
				}
			}
		})
	}
}
func TestDecodeValidPaddingAndExactBudgets(t *testing.T) {
	members := fixture(t)
	raw := rawTar(t, members)
	compressed := zipTar(t, raw)
	l := DefaultLimits
	l.CompressedBytes = int64(len(compressed))
	l.DecodedBytes = int64(len(raw))
	l.Entries = len(members)
	l.ExpandedBytes = int64(len(members[0].data) + len(members[1].data))
	l.FileBytes = int64(len(members[0].data))
	if _, err := Decode(bytes.NewReader(compressed), testIdentity(), l); err != nil {
		t.Fatalf("exact budget rejected: %v", err)
	}
	if _, err := Decode(bytes.NewReader(zipTar(t, append(raw, make([]byte, 1024)...))), testIdentity(), DefaultLimits); err != nil {
		t.Fatalf("zero padding rejected: %v", err)
	}
}
