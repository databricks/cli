package auth_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/require"
)

func TestAuthDescribeSuccess(t *testing.T) {
	t.Skipf("Skipping because of https://github.com/databricks/cli/issues/2010")

	stdout, _ := testcli.RequireSuccessfulRun(t, "auth", "describe")
	outStr := stdout.String()

	w, err := databricks.NewWorkspaceClient(&databricks.Config{})
	require.NoError(t, err)

	require.NotEmpty(t, outStr)
	require.Contains(t, outStr, fmt.Sprintf("Host: %s", w.Config.Host))

	me, err := w.CurrentUser.Me(context.Background())
	require.NoError(t, err)
	require.Contains(t, outStr, fmt.Sprintf("User: %s", me.UserName))
	require.Contains(t, outStr, fmt.Sprintf("Authenticated with: %s", w.Config.AuthType))
	require.Contains(t, outStr, "Current configuration:")
	require.Contains(t, outStr, fmt.Sprintf("✓ host: %s", w.Config.Host))
	require.Contains(t, outStr, "✓ profile: default")
}

func TestAuthDescribeFailure(t *testing.T) {
	t.Skipf("Skipping because of https://github.com/databricks/cli/issues/2010")

	stdout, _ := testcli.RequireSuccessfulRun(t, "auth", "describe", "--profile", "nonexistent")
	outStr := stdout.String()

	require.NotEmpty(t, outStr)
	require.Contains(t, outStr, "Unable to authenticate: resolve")
	require.Contains(t, outStr, "has no nonexistent profile configured")
	require.Contains(t, outStr, "Current configuration:")

	w, err := databricks.NewWorkspaceClient(&databricks.Config{})
	require.NoError(t, err)

	require.Contains(t, outStr, fmt.Sprintf("✓ host: %s", w.Config.Host))
	require.Contains(t, outStr, "✓ profile: nonexistent (from --profile flag)")
}
