package plugin

import (
	"bytes"
	"encoding/json"
	"io"
)

// pluginRecord is the union of fields Claude and Codex emit in `plugin list
// --json`. Both tools key a plugin by a name and a marketplace; Codex also flags
// installed/enabled and uses "marketplaceName".
type pluginRecord struct {
	Name            string `json:"name"`
	Marketplace     string `json:"marketplace"`
	MarketplaceName string `json:"marketplaceName"`
	Installed       *bool  `json:"installed"` // pointer: absent (Claude) => treated as installed
}

func (r pluginRecord) id() string {
	mkt := r.Marketplace
	if mkt == "" {
		mkt = r.MarketplaceName
	}
	return r.Name + "@" + mkt
}

func (r pluginRecord) isInstalled() bool {
	// Claude lists only installed plugins (no flag) -> installed when absent.
	return r.Installed == nil || *r.Installed
}

// DetectInstalled parses a tool's `plugin list --json` output into the set of
// installed "<plugin>@<marketplace>" ids. Empty input yields an empty set (the
// tool reported nothing); malformed JSON is an error. tool is accepted for future
// per-tool shape divergence but both current shapes parse uniformly.
func DetectInstalled(tool string, r io.Reader) (map[string]bool, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return out, nil
	}

	// Two shapes: a bare array (Claude) or an object with an "installed" array
	// (Codex). Try the array first, then the wrapped form.
	var arr []pluginRecord
	if err := json.Unmarshal(trimmed, &arr); err == nil {
		for _, rec := range arr {
			if rec.isInstalled() {
				out[rec.id()] = true
			}
		}
		return out, nil
	}
	var wrapped struct {
		Installed []pluginRecord `json:"installed"`
		Plugins   []pluginRecord `json:"plugins"`
	}
	if err := json.Unmarshal(trimmed, &wrapped); err != nil {
		return nil, err
	}
	for _, rec := range append(wrapped.Installed, wrapped.Plugins...) {
		if rec.isInstalled() {
			out[rec.id()] = true
		}
	}
	return out, nil
}
