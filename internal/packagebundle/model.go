// Package packagebundle validates static directory packages without filesystem or process effects.
package packagebundle

// Identity identifies one package release and host platform.
type Identity struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

// File contains verified member bytes and normalized permissions.
type File struct {
	Path string `json:"path"`
	Mode uint32 `json:"mode"`
	Data []byte `json:"data"`
}

// Entry records a payload's digest and normalized permissions.
type Entry struct {
	Path   string `json:"path"`
	Mode   uint32 `json:"mode"`
	SHA256 string `json:"sha256"`
}

// Metadata is the package.json inventory, excluding package.json itself.
type Metadata struct {
	SchemaVersion int      `json:"schemaVersion"`
	Identity      Identity `json:"identity"`
	SourceSHA256  string   `json:"sourceSHA256"`
	Files         []Entry  `json:"files"`
}

// Bundle contains all verified regular files, including package.json.
type Bundle struct {
	Metadata Metadata `json:"metadata"`
	Files    []File   `json:"files"`
}

// Limits bounds compressed input, all decompressed bytes, payloads, and names.
// Values must be positive and may only tighten the fixed format ceilings.
type Limits struct {
	CompressedBytes int64 `json:"compressedBytes"`
	ExpandedBytes   int64 `json:"expandedBytes"`
	FileBytes       int64 `json:"fileBytes"`
	DecodedBytes    int64 `json:"decodedBytes"`
	Entries         int   `json:"entries"`
	Components      int   `json:"components"`
	PathBytes       int   `json:"pathBytes"`
}

// DefaultLimits contains the directory package format ceilings.
var DefaultLimits = Limits{CompressedBytes: 32 << 20, ExpandedBytes: 128 << 20, FileBytes: 16 << 20, DecodedBytes: 136 << 20, Entries: 4096, Components: 16, PathBytes: 512}
