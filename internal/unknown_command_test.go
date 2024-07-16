package internal

import (
	"testing"

	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestUnknownCommand(t *testing.T) {
	stdout, stderr, err := RequireErrorRun(t, "unknown-command")

	assert.Error(t, err, "unknown command", `unknown command "unknown-command" for "databricks"`)
	assert.Equal(t, "", stdout.String())
	assert.Contains(t, stderr.String(), "unknown command")
}
