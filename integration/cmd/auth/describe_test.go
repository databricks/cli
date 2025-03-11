package auth_test

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go/config"

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
	testutil.CleanupEnvironment(t)

	// set up a custom config file:
	home := t.TempDir()
	cfg := &config.Config{
		ConfigFile: filepath.Join(home, "customcfg"),
		Profile:    "profile1",
	}
	err := databrickscfg.SaveToProfile(context.Background(), cfg)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(home, "customcfg"))

	// run the command:
	ctx := context.Background()
	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "auth", "describe", "--profile", "nonexistent")
	outStr := stdout.String()

	require.NotEmpty(t, outStr)
	require.Contains(t, outStr, "Unable to authenticate: resolve")
	require.Contains(t, outStr, "has no nonexistent profile configured")
	require.Contains(t, outStr, "Current configuration:")
	require.Contains(t, outStr, "✓ profile: nonexistent (from --profile flag)")
}
