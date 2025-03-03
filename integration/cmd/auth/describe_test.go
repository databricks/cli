package auth_test

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/require"
)

func TestAuthDescribeSuccess(t *testing.T) {
	ctx := context.Background()
	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "auth", "describe")
	outStr := stdout.String()

	w, err := databricks.NewWorkspaceClient(&databricks.Config{})
	require.NoError(t, err)

	require.NotEmpty(t, outStr)

	hostWithoutPrefix := strings.TrimPrefix(w.Config.Host, "https://")
	require.Regexp(t, "Host: (?:https://)?"+regexp.QuoteMeta(hostWithoutPrefix), outStr)

	me, err := w.CurrentUser.Me(context.Background())
	require.NoError(t, err)
	require.Contains(t, outStr, "User: "+me.UserName)
	require.Contains(t, outStr, "Authenticated with: "+w.Config.AuthType)
	require.Contains(t, outStr, "Current configuration:")
	require.Contains(t, outStr, "✓ host: "+w.Config.Host)
	require.Contains(t, outStr, "✓ profile: default")
}

func TestAuthDescribeFailure(t *testing.T) {
	// Store the original value of env variable
	originalProfileValue, envProfileExists := os.LookupEnv("DATABRICKS_CONFIG_PROFILE")

	// restore env variable after the test:
	if envProfileExists {
		// Unset the env variable for this test
		err := os.Unsetenv("DATABRICKS_CONFIG_PROFILE")
		require.NoError(t, err)

		t.Cleanup(func() {
			err := os.Setenv("DATABRICKS_CONFIG_PROFILE", originalProfileValue)
			require.NoError(t, err)
		})
	}

	ctx := context.Background()
	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "auth", "describe", "--profile", "nonexistent", "--log-level", "trace")
	outStr := stdout.String()

	require.NotEmpty(t, outStr)
	require.Contains(t, outStr, "Unable to authenticate: resolve")
	require.Contains(t, outStr, "has no nonexistent profile configured")
	require.Contains(t, outStr, "Current configuration:")

	w, err := databricks.NewWorkspaceClient(&databricks.Config{})
	require.NoError(t, err)

	require.Contains(t, outStr, "✓ host: "+w.Config.Host)
	require.Contains(t, outStr, "✓ profile: nonexistent (from --profile flag)")
}
