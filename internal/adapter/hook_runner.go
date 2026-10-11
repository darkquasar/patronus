package adapter

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/darkquasar/patronus/internal/manifest"
)

// hookRunnerFile is the file name of the adapter-embedded hook runner placed in
// a tool's runner dir.
const hookRunnerFile = "patronus-hook-runner.ts"

// hookRunnerDialectToken is the single placeholder in the runner template that
// receives the adapter's RunnerDialect as JSON.
const hookRunnerDialectToken = "/*patronus:dialect*/null"

//go:embed hook_runner.ts.tmpl
var hookRunnerTemplate []byte

// renderHookRunner renders the runner source for one adapter dialect. The
// dialect is adapter data (toolMap, argMap, schemaVersion), marshalled with
// sorted keys so the bytes are deterministic: every script hook of one tool
// places the identical file, which composition records as one shared placed
// file. The template is the OpenCode plugin API shape; a second harness that
// declares runner dirs would need its own template (stated limit).
func renderHookRunner(dialect manifest.RunnerDialect) ([]byte, error) {
	if dialect.SchemaVersion == 0 || len(dialect.ToolMap) == 0 {
		return nil, fmt.Errorf("runner dialect needs schemaVersion and a toolMap")
	}
	if bytes.Count(hookRunnerTemplate, []byte(hookRunnerDialectToken)) != 1 {
		return nil, fmt.Errorf("runner template must carry the dialect token exactly once")
	}
	encoded, err := json.Marshal(dialect)
	if err != nil {
		return nil, fmt.Errorf("encode runner dialect: %w", err)
	}
	return bytes.Replace(hookRunnerTemplate, []byte(hookRunnerDialectToken), encoded, 1), nil
}
