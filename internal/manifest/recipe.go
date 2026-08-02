package manifest

import (
	"errors"
	"fmt"
	"strings"
)

// Recipe is an external binary/tool that Patronus delivers (optional fetch+verify)
// and/or wires into each agent (§4). It carries NO file type — its shape is a
// pure function of deliver × wire (see Shape).
//
// A recipe IS versioned like every manifest (the required Meta.Version, ADR-0004),
// but its shape stays COMPUTED — versioning is an identity concern (which revision),
// shape a structural one (what it writes). Adding the first does not make a recipe
// declare a `type:`; there is no ArtifactType on recipes (see Shape/RecipeShape).
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

// DeliverVia is the MECHANISM by which a recipe's payload is obtained. It
// separates the mechanism (how) from the manager (which tool), which the old
// DeliverySource enum conflated.
type DeliverVia string

const (
	ViaFetch          DeliverVia = "fetch"           // a pinned binary/archive (per-OS Assets) or single URL artifact
	ViaPackageManager DeliverVia = "package-manager" // installed via a PackageManager candidate list
	ViaDocker         DeliverVia = "docker"          // a container image
	ViaScript         DeliverVia = "script"          // a self-installing script
)

var deliverVias = map[DeliverVia]bool{ViaFetch: true, ViaPackageManager: true, ViaDocker: true, ViaScript: true}

// PackageManager is the shared vocabulary of package managers, used by every
// InstallCandidate (blessed or fallback). One enum, one detection map.
type PackageManager string

const (
	PMNpm    PackageManager = "npm"
	PMCargo  PackageManager = "cargo"
	PMUv     PackageManager = "uv"
	PMBrew   PackageManager = "brew"
	PMScoop  PackageManager = "scoop"
	PMWinget PackageManager = "winget"
	PMAur    PackageManager = "aur"
)

var packageManagers = map[PackageManager]bool{
	PMNpm: true, PMCargo: true, PMUv: true, PMBrew: true, PMScoop: true, PMWinget: true, PMAur: true,
}

// pmInstallTemplates renders the user-scope install command per manager. A
// manager absent here has no known install command and cannot render one.
// NOTE: npm uses -g (user-scope on nvm/brew node; fails with a permission error
// on a system-prefix rather than escalating — Patronus never runs sudo).
var pmInstallTemplates = map[PackageManager]string{
	PMNpm:   "npm install -g %s",
	PMCargo: "cargo install %s",
	PMUv:    "uv tool install %s",
}

// InstallCandidate is one (manager, ref) way to install the recipe. The list is
// ordered by Preference (lower first); the first candidate whose manager is on
// PATH wins. A fallback is simply a lower-preference candidate — there is no
// separate Fallback concept.
type InstallCandidate struct {
	Manager    PackageManager `yaml:"manager" json:"manager"`
	Ref        string         `yaml:"ref,omitempty" json:"ref,omitempty"` // package name in that manager; defaults to recipe name
	Preference int            `yaml:"preference,omitempty" json:"preference,omitempty"`
}

// InstallCommand renders this candidate's user-scope install command, or "" if
// the manager has no template. ref defaults to recipeName.
func (c InstallCandidate) InstallCommand(recipeName string) string {
	tmpl, ok := pmInstallTemplates[c.Manager]
	if !ok {
		return ""
	}
	ref := c.Ref
	if ref == "" {
		ref = recipeName
	}
	return fmt.Sprintf(tmpl, ref)
}

// Delivery describes how the recipe's payload is obtained (§4b). Via is the
// mechanism; the mechanism-specific fields carry the rest.
type Delivery struct {
	Via DeliverVia `yaml:"via" json:"via"` // fetch | package-manager | docker | script
	// Install is the ordered candidate list for via: package-manager (first
	// present-on-PATH wins). Empty for fetch/docker/script.
	Install   []InstallCandidate `yaml:"install,omitempty" json:"install,omitempty"`
	InstallTo string             `yaml:"installTo,omitempty" json:"installTo,omitempty"`
	Binary    string             `yaml:"binary,omitempty" json:"binary,omitempty"` // installed binary filename (defaults to recipe name)
	Assets    []Asset            `yaml:"assets,omitempty" json:"assets,omitempty"` // via: fetch, per-OS matrix

	// via: fetch single artifact — a single platform-independent artifact (e.g. a
	// shell script). There is no per-OS/arch matrix — Platforms gates which hosts
	// it runs on.
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

// WireMethod is WHAT KIND of wiring operation a recipe performs (one axis).
type WireMethod string

const (
	WireMerge WireMethod = "merge" // Patronus merges structured config (MCP block, settings, hook array)
	WireExec  WireMethod = "exec"  // the wiring is one or more shell commands
	WireNone  WireMethod = ""      // no wiring — delivering the package is the whole job
)

// WireActor is WHO performs the wiring (the orthogonal axis).
type WireActor string

const (
	ActorPatronus WireActor = "patronus" // Patronus runs it and owns the reversal (state-tracked)
	ActorExternal WireActor = "external" // something outside Patronus does it; surfaced as advisory, remove reports manual-cleanup
)

var wireMethods = map[WireMethod]bool{WireMerge: true, WireExec: true, WireNone: true}
var wireActors = map[WireActor]bool{ActorPatronus: true, ActorExternal: true}

// Wire describes how the recipe is bound to each agent. Method and Actor are the
// two orthogonal discriminators; the method-specific field carries the payload.
type Wire struct {
	Method WireMethod `yaml:"method,omitempty" json:"method,omitempty"`
	Actor  WireActor  `yaml:"actor,omitempty" json:"actor,omitempty"`
	Tools  []string   `yaml:"tools,omitempty" json:"tools,omitempty"`
	Mcp    *WireMcp   `yaml:"mcp,omitempty" json:"mcp,omitempty"` // present iff method == merge
	Run    []string   `yaml:"run,omitempty" json:"run,omitempty"` // present iff method == exec
}

// WireMcp is the MCP-config entry Patronus merges for a method:merge recipe.
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

// Shape derives the recipe's type from deliver × wire method. Actor does not
// affect shape (a run and a self-wiring command are both fetch+run) — it only
// affects whether the emitted EXEC is advisory (see internal/recipe).
func (r *Recipe) Shape() RecipeShape {
	switch {
	case r.Delivery == nil:
		return ShapeWireOnly
	case r.Wire.Method == WireNone:
		return ShapeInstall // deliver a package and stop; something else (a hook) wires it
	case r.Wire.Method == WireExec:
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
	// An empty wire.method is the install-only recipe (deliver a package, wire
	// nothing — a hook or another item does the wiring). It is valid ONLY with a
	// deliver block; a recipe that neither delivers nor wires does nothing.
	if r.Wire.Method == WireNone {
		if r.Delivery == nil {
			return fmt.Errorf("recipe does nothing: needs a wire.method or a deliver block")
		}
		return nil
	}
	if !wireMethods[r.Wire.Method] {
		return fmt.Errorf("invalid wire.method %q (want merge|exec, or omit for an install-only deliver recipe)", r.Wire.Method)
	}
	if !wireActors[r.Wire.Actor] {
		return fmt.Errorf("wire.method %s requires wire.actor (patronus|external)", r.Wire.Method)
	}
	switch r.Wire.Method {
	case WireMerge:
		if r.Wire.Actor != ActorPatronus {
			return fmt.Errorf("wire.method merge requires actor: patronus (Patronus performs the merge)")
		}
		if r.Wire.Mcp == nil {
			return fmt.Errorf("wire.method merge requires a wire.mcp block")
		}
	case WireExec:
		if len(r.Wire.Run) == 0 {
			return fmt.Errorf("wire.method exec requires wire.run commands")
		}
	}
	return nil
}

// validateDelivery checks the mechanism-specific shape of a deliver block. The
// via enum is closed; each mechanism then has its own required fields.
func validateDelivery(d *Delivery) error {
	if !deliverVias[d.Via] {
		return fmt.Errorf("invalid deliver.via %q (want fetch|package-manager|docker|script)", d.Via)
	}
	switch d.Via {
	case ViaPackageManager:
		if len(d.Install) == 0 {
			return errors.New("deliver.via package-manager requires at least one install candidate")
		}
		for _, c := range d.Install {
			if !packageManagers[c.Manager] {
				return fmt.Errorf("invalid install candidate manager %q", c.Manager)
			}
		}
	case ViaFetch:
		// A single-URL fetch is a pinned artifact needing url+sha256; an
		// asset-matrix fetch carries per-OS assets. The two are distinguished by
		// which field is set (URL vs Assets), not a separate via. A fetch with
		// NEITHER is a stub whose upstream isn't pinned yet (the sandbox recipe):
		// still valid, it simply emits no FETCH for any host until assets land —
		// the pre-refactor github-release behaviour.
		if d.URL != "" {
			if d.SHA256 == "" {
				return errors.New("deliver.via fetch with a url requires a sha256 (the pinned trust anchor)")
			}
			if len(d.Assets) > 0 {
				return errors.New("deliver.via fetch takes a single url OR assets, not both")
			}
		}
	}
	return nil
}
