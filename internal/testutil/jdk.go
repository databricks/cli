package testutil

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/process"
	"github.com/stretchr/testify/require"
)

func RequireJDK(t *testing.T, ctx context.Context, version string) {
	var stderr bytes.Buffer
	err := process.Forwarded(ctx, []string{"javac", "-version"}, nil, nil, &stderr)
	require.NoError(t, err, "Unable to run javac -version")

	// Get the first line of the output
	line := strings.Split(stderr.String(), "\n")[0]
	require.Contains(t, line, version, "Expected JDK version %s, got %s", version, line)
}
