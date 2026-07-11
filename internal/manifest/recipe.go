package manifest

import (
	"errors"
	"fmt"
	"strings"
)

// Recipe is an external binary/tool that Patronus delivers (optional fetch+verify)
// and/or wires into each agent (§4). It carries NO file type — its shape is a
// pure function of deliver × wire (see Shape).
type Recipe struct {
	Meta     `yaml:",inline" json:",inline"`
	Summary  string       `yaml:"summary,omitempty" json:"summary,omitempty"`
	Upstream string       `yaml:"upstream,omitempty" json:"upstream,omitempty"`
	License  string       `yaml:"license,omitempty" json:"license,omitempty"`
	Delivery *Delivery    `yaml:"deliver,omitempty" json:"deliver,omitempty"` // nil for wire-only remote MCP
	Scope    *RecipeScope `yaml:"scope,omitempty" json:"scope,omitempty"`
	Wire     Wire         `yaml:"wire" json:"wire"`
	Seed     []string     `yaml:"seed,omitempty" json:"seed,omitempty"`
}

// Header returns the recipe's shared identity header (implements Installable).
func (r *Recipe) Header() Meta { return r.Meta }

// DeliverySource is the closed set of ways a recipe's binary/script is obtained.
type DeliverySource string

const (
	SourceGithubRelease DeliverySource = "github-release"
	SourceDocker        DeliverySource = "docker"
	SourceCargo         DeliverySource = "cargo"
	SourceNpm           DeliverySource = "npm"
	SourceScript        DeliverySource = "script"
	SourceURL           DeliverySource = "url"
)

var deliverySources = map[DeliverySource]bool{
	SourceGithubRelease: true, SourceDocker: true, SourceCargo: true,
	SourceNpm: true, SourceScript: true, SourceURL: true,
}

// pmInstallTemplates renders the global-install command for a package-manager
// delivery source. A package-install recipe (e.g. tdd-guard via npm) surfaces
// this command as a display-only advisory row — Patronus does not silently run a
// global package install; the user (or a future --prefer-system-pkg path) runs it.
var pmInstallTemplates = map[DeliverySource]string{
	SourceNpm:   "npm install -g %s",
	SourceCargo: "cargo install %s",
}

// InstallCommand returns the global package-install command for a package-manager
// delivery (npm/cargo), or "" for a source that is fetched/built differently
// (github-release, docker, script, url). ref defaults to the recipe name when the
// deliver block omits it.
func (d *Delivery) InstallCommand(recipeName string) string {
	tmpl, ok := pmInstallTemplates[d.Source]
	if !ok {
		return ""
	}
	ref := d.Ref
	if ref == "" {
		ref = recipeName
	}
	return fmt.Sprintf(tmpl, ref)
}

// Delivery describes how the recipe's binary/script is obtained (§4b).
type Delivery struct {
	Source    DeliverySource `yaml:"source" json:"source"`                           // github-release | docker | cargo | npm | script | url
	Ref       string         `yaml:"ref,omitempty" json:"ref,omitempty"`             // package name for a PM source (npm/cargo); defaults to the recipe name
	Fallbacks []Fallback     `yaml:"fallbacks,omitempty" json:"fallbacks,omitempty"` // opt-in system-PM alternatives (--prefer-system-pkg)
	InstallTo string         `yaml:"installTo,omitempty" json:"installTo,omitempty"`
	Binary    string         `yaml:"binary,omitempty" json:"binary,omitempty"` // installed binary filename (defaults to recipe name)
	Assets    []Asset        `yaml:"assets,omitempty" json:"assets,omitempty"`

	// url source: a single platform-independent artifact (e.g. a shell script).
	// There is no per-OS/arch matrix — Platforms gates which hosts it runs on.
	URL       string   `yaml:"url,omitempty" json:"url,omitempty"`
	SHA256    string   `yaml:"sha256,omitempty" json:"sha256,omitempty"`       // hex digest; pinned
	Platforms []string `yaml:"platforms,omitempty" json:"platforms,omitempty"` // GOOS allow-list; empty = unrestricted
}

// PinnedURL is the single pinned download of a `url` delivery. It exists so
// ResolveURL returns one named value rather than two bare strings: a transposed
// (url, sha256) pair would silently disarm the trust anchor.
type PinnedURL struct {
	URL    string
	SHA256 string // hex digest; pinned
}

// ResolveURL returns the pinned artifact for a `url` delivery, or a clear error
// when the host GOOS is outside the recipe's Platforms allow-list. It is the
// `url`-source analogue of ResolveAsset: the caller (fetchDiff) turns the error
// into a warning and emits no FETCH, rather than downloading something the host
// cannot execute. An empty Platforms list means unrestricted.
func (d *Delivery) ResolveURL(goos string) (*PinnedURL, error) {
	if !d.supportsOS(goos) {
		return nil, fmt.Errorf("deliver: not supported on %s (platforms: %s)",
			goos, strings.Join(d.Platforms, ", "))
	}
	return &PinnedURL{URL: d.URL, SHA256: d.SHA256}, nil
}

// supportsOS reports whether goos is in the Platforms allow-list. An empty list
// means unrestricted (the common case: a real cross-platform binary).
func (d *Delivery) supportsOS(goos string) bool {
	if len(d.Platforms) == 0 {
		return true
	}
	for _, p := range d.Platforms {
		if p == goos {
			return true
		}
	}
	return false
}

// Fallback is one opt-in system-package-manager alternative to the blessed
// delivery source, consulted only under --prefer-system-pkg. A package manager
// that doesn't (yet) carry the recipe simply isn't listed — there is no
// "false" placeholder (which is what the old map[string]interface{} encoded).
type Fallback struct {
	PM  string `yaml:"pm" json:"pm"`   // brew | scoop | winget | cargo | aur | npm | ...
	Ref string `yaml:"ref" json:"ref"` // package name / install ref in that PM
}

// Asset is one pinned per-OS/arch github-release artifact (§4b floor, pinned
// trust model). Archive/BinaryPath are set when the asset is a tar.gz/zip rather
// than a bare binary; the FETCH step extracts BinaryPath.
type Asset struct {
	OS         string `yaml:"os" json:"os"`     // GOOS: linux | darwin | windows
	Arch       string `yaml:"arch" json:"arch"` // GOARCH: amd64 | arm64
	URL        string `yaml:"url" json:"url"`
	SHA256     string `yaml:"sha256" json:"sha256"`                             // hex digest; pinned
	Archive    string `yaml:"archive,omitempty" json:"archive,omitempty"`       // "" | tar.gz | tgz | zip
	BinaryPath string `yaml:"binaryPath,omitempty" json:"binaryPath,omitempty"` // member to extract from the archive
}

// ResolveAsset returns the asset matching the given GOOS/GOARCH (injected so the
// caller — and tests — control the host). It errors clearly when no asset is
// pinned for the host, which is also how the ai-memory "experimental Windows"
// caveat (§5c) surfaces: a missing windows asset is an explicit, actionable error
// rather than a silent failure.
func (d *Delivery) ResolveAsset(goos, goarch string) (*Asset, error) {
	for i := range d.Assets {
		if d.Assets[i].OS == goos && d.Assets[i].Arch == goarch {
			return &d.Assets[i], nil
		}
	}
	if len(d.Assets) == 0 {
		return nil, fmt.Errorf("deliver: no assets pinned (upstream not yet resolved)")
	}
	return nil, fmt.Errorf("deliver: no asset for %s/%s", goos, goarch)
}

// RecipeScope captures per-repo isolation markers and the global store location.
type RecipeScope struct {
	Marker string `yaml:"marker,omitempty" json:"marker,omitempty"`
	Global string `yaml:"global,omitempty" json:"global,omitempty"`
}

// WireMode is the SINGLE source of wiring truth — replacing the old
// SelfWiring bool + implicit (which-of-Mcp/PostInstall-is-set) signalling.
type WireMode string

const (
	WireModeMcp  WireMode = "mcp"  // Patronus performs the MCP-config MERGE itself
	WireModeRun  WireMode = "run"  // Patronus runs the commands we specify (EXEC)
	WireModeSelf WireMode = "self" // the recipe's own installer wires it (EXEC, self-managing)
)

var wireModes = map[WireMode]bool{WireModeMcp: true, WireModeRun: true, WireModeSelf: true}

// Wire describes how the recipe is bound to each agent. Mode is the single
// discriminator; the mode-specific field (Mcp for mcp, Run for run/self) carries
// the payload.
type Wire struct {
	Mode  WireMode `yaml:"mode" json:"mode"`
	Tools []string `yaml:"tools,omitempty" json:"tools,omitempty"`
	Mcp   *WireMcp `yaml:"mcp,omitempty" json:"mcp,omitempty"` // present iff mode == mcp
	Run   []string `yaml:"run,omitempty" json:"run,omitempty"` // present iff mode == run or self (was: postInstall)
}

// WireMcp is the MCP-config entry Patronus merges for a mode: mcp recipe.
type WireMcp struct {
	Transport string   `yaml:"transport" json:"transport"` // http | stdio
	URL       string   `yaml:"url,omitempty" json:"url,omitempty"`
	Command   string   `yaml:"command,omitempty" json:"command,omitempty"`
	Args      []string `yaml:"args,omitempty" json:"args,omitempty"`
}

// RecipeShape is the COMPUTED type of a recipe (§4c) — never authored, so it
// cannot contradict the deliver/wire structure.
type RecipeShape string

const (
	ShapeWireOnly  RecipeShape = "wire-only"    // no delivery, just a config MERGE (github)
	ShapeFetchWire RecipeShape = "fetch+wire"   // fetch a binary, then MERGE config (engram)
	ShapeFetchRun  RecipeShape = "fetch+run"    // fetch (or docker) + EXEC commands (ai-memory, script)
	ShapeInstall   RecipeShape = "install-only" // deliver a package, no wiring (tdd-guard via npm)
)

// Shape derives the recipe's type from deliver × wire. It is a pure function with
// no ambiguous case — honest only because Delivery is nil-or-present and
// Wire.Mode is a single enum (possibly empty for an install-only recipe).
func (r *Recipe) Shape() RecipeShape {
	switch {
	case r.Delivery == nil:
		return ShapeWireOnly
	case r.Wire.Mode == "":
		return ShapeInstall // deliver a package and stop; something else (a hook) wires it
	case r.Wire.Mode == WireModeRun || r.Wire.Mode == WireModeSelf:
		return ShapeFetchRun
	default:
		return ShapeFetchWire
	}
}

// LoadRecipe reads and validates a recipe manifest.
func LoadRecipe(path string) (*Recipe, error) {
	var r Recipe
	if err := decodeFile(path, &r); err != nil {
		return nil, err
	}
	if err := validateRecipe(&r); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &r, nil
}

// DecodeRecipe parses+validates a recipe manifest from raw YAML bytes — used for
// an https: sourced manifest that never lands on a local path.
func DecodeRecipe(data []byte) (*Recipe, error) {
	var r Recipe
	if err := decodeBytes(data, &r); err != nil {
		return nil, err
	}
	if err := validateRecipe(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func validateRecipe(r *Recipe) error {
	if err := validateMeta(r.Meta, FamilyRecipe); err != nil {
		return err
	}
	if r.Role == "" {
		return fmt.Errorf("missing role")
	}
	if r.Delivery != nil {
		if err := validateDelivery(r.Delivery); err != nil {
			return err
		}
	}
	// An empty wire.mode is the install-only recipe (deliver a package, wire
	// nothing — a hook or another item does the wiring). It is valid ONLY with a
	// deliver block; a recipe that neither delivers nor wires does nothing.
	if r.Wire.Mode == "" {
		if r.Delivery == nil {
			return fmt.Errorf("recipe does nothing: needs a wire.mode or a deliver block")
		}
		return nil
	}
	if !wireModes[r.Wire.Mode] {
		return fmt.Errorf("invalid wire.mode %q (want mcp|run|self, or omit for an install-only deliver recipe)", r.Wire.Mode)
	}
	switch r.Wire.Mode {
	case WireModeMcp:
		if r.Wire.Mcp == nil {
			return fmt.Errorf("wire.mode mcp requires a wire.mcp block")
		}
	case WireModeRun, WireModeSelf:
		if len(r.Wire.Run) == 0 {
			return fmt.Errorf("wire.mode %s requires wire.run commands", r.Wire.Mode)
		}
	}
	return nil
}

// validateDelivery checks the source-specific shape of a deliver block. The
// source enum itself is closed; each source then has its own required fields.
func validateDelivery(d *Delivery) error {
	if !deliverySources[d.Source] {
		return fmt.Errorf("invalid deliver.source %q", d.Source)
	}
	if d.Source != SourceURL {
		return nil
	}
	// A url delivery is a single pinned artifact, not a per-OS/arch matrix.
	if d.URL == "" {
		return errors.New("deliver: source url requires a url")
	}
	if d.SHA256 == "" {
		return errors.New("deliver: source url requires a sha256 (the pinned trust anchor)")
	}
	if len(d.Assets) > 0 {
		return errors.New("deliver: source url takes a single url, not assets")
	}
	return nil
}
