package cmd_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/stretchr/testify/assert"
)

func TestUnknownCommand(t *testing.T) {
	ctx := context.Background()
	stdout, stderr, err := testcli.RequireErrorRun(t, ctx, "unknown-command")

	assert.Error(t, err, "unknown command", `unknown command "unknown-command" for "databricks"`)
	assert.Equal(t, "", stdout.String())
	assert.Contains(t, stderr.String(), "unknown command")
}
