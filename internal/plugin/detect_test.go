package plugin

import (
	"strings"
	"testing"
)

func TestDetectInstalledClaudeShape(t *testing.T) {
	// Claude `plugin list --json`: array of installed plugins.
	js := `[
	  {"name":"superpowers","marketplace":"claude-plugins-official","version":"6.0.3"},
	  {"name":"other","marketplace":"acme","version":"1.0.0"}
	]`
	got, err := DetectInstalled("claude", strings.NewReader(js))
	if err != nil {
		t.Fatal(err)
	}
	if !got["superpowers@claude-plugins-official"] || !got["other@acme"] {
		t.Errorf("missing ids: %v", got)
	}
}

func TestDetectInstalledCodexShape(t *testing.T) {
	// Codex `plugin list --json`: objects with installed/enabled + marketplaceName.
	js := `{"installed":[
	  {"name":"superpowers","marketplaceName":"openai-curated","installed":true},
	  {"name":"ghost","marketplaceName":"openai-curated","installed":false}
	]}`
	got, err := DetectInstalled("codex", strings.NewReader(js))
	if err != nil {
		t.Fatal(err)
	}
	if !got["superpowers@openai-curated"] {
		t.Errorf("expected superpowers installed, got %v", got)
	}
	if got["ghost@openai-curated"] {
		t.Errorf("ghost is installed:false and must not appear: %v", got)
	}
}

func TestDetectInstalledEmptyOrGarbled(t *testing.T) {
	if got, err := DetectInstalled("claude", strings.NewReader("")); err != nil || len(got) != 0 {
		t.Errorf("empty input should yield empty set no error, got %v err %v", got, err)
	}
	if _, err := DetectInstalled("claude", strings.NewReader("{not json")); err == nil {
		t.Error("garbled JSON should error")
	}
}
