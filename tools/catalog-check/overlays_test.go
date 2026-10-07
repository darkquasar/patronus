package main

import (
	"strings"
	"testing"
)

func overlayFixture(field string, values []any) *catalog {
	return &catalog{items: map[string]*item{
		"invented-role": {
			manifest: document{"type": "agent", "entry": "agent.md", "targets": []any{"pi"}},
			payload:  map[string][]byte{"agent.md": []byte("---\nname: invented-role\ntools: read, bash\nskills: base-guide\n---\nInert role.\n")},
		},
		"overlay": {manifest: document{
			"type": "setting", "targets": []any{"pi"}, "requires": []any{"extra-guide"},
			"setting": document{"path": "subagents.agentOverrides.invented-role." + field, "value": values},
		}},
		"extra-guide": {manifest: document{"type": "skill", "targets": []any{"pi"}}},
	}}
}

func TestNativeAgentOverlaysFollowCurrentBase(t *testing.T) {
	for _, field := range []string{"tools", "skills"} {
		t.Run(field, func(t *testing.T) {
			values := []any{"base-guide", "extra-guide"}
			if field == "tools" {
				values = []any{"read", "bash", "mcp"}
			}
			c := overlayFixture(field, values)
			if err := c.checkAgentOverlays(); err != nil {
				t.Fatal(err)
			}
			// An upstream role update must invalidate a stale complete-list override.
			c.items["invented-role"].payload["agent.md"] = []byte("---\nname: invented-role\ntools: read, bash, write\nskills: base-guide, new-guide\n---\nUpdated role.\n")
			if err := c.checkAgentOverlays(); err == nil || !strings.Contains(err.Error(), "complete ordered") {
				t.Fatalf("stale %s override accepted: %v", field, err)
			}
		})
	}
}

func TestNativeAgentOverlaysRejectBrokenDependencies(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		mutate     func(*catalog)
	}{
		{"missing-role", "missing native agent", func(c *catalog) { delete(c.items, "invented-role") }},
		{"missing-skill", "declared skill dependency", func(c *catalog) { delete(c.items, "extra-guide") }},
		{"undeclared-skill", "declared skill dependency", func(c *catalog) { c.items["overlay"].manifest["requires"] = nil }},
		{"wrong-type", "declared skill dependency", func(c *catalog) { c.items["extra-guide"].manifest["type"] = "instruction" }},
		{"wrong-target", "not Pi-compatible", func(c *catalog) { c.items["extra-guide"].manifest["targets"] = []any{"claude"} }},
		{"missing-frontmatter", "missing agent frontmatter", func(c *catalog) { c.items["invented-role"].payload["agent.md"] = []byte("Inert role.") }},
		{"missing-list", "complete ordered", func(c *catalog) { object(c.items["overlay"].manifest, "setting")["value"] = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := overlayFixture("skills", []any{"base-guide", "extra-guide"})
			tc.mutate(c)
			if err := c.checkAgentOverlays(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("broken overlay accepted: %v", err)
			}
		})
	}
}

func TestNativeAgentOverlaysIgnoreUnrelatedSettings(t *testing.T) {
	c := overlayFixture("skills", nil)
	object(c.items["overlay"].manifest, "setting")["path"] = "unrelated.preference"
	if err := c.checkAgentOverlays(); err != nil {
		t.Fatal(err)
	}
}
