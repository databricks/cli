package cmdio_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestContextWithStdout(t *testing.T) {
	ctx, stdout := cmdio.NewTestContextWithStdout(context.Background())

	// Render writes to stdout
	data := map[string]string{"message": "test output"}
	err := cmdio.Render(ctx, data)
	require.NoError(t, err)

	assert.Contains(t, stdout.String(), "test output")
}

func TestNewTestContextWithStderr(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(context.Background())

	require.NotPanics(t, func() {
		cmdio.LogString(ctx, "test message")
	})

	assert.Contains(t, stderr.String(), "test message")
}
