package root

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func newWorkspaceAuthTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	initProfileFlag(cmd)
	initTargetFlag(cmd)
	initEnvironmentFlag(cmd)
	return cmd
}

func TestMustWorkspaceClientIgnoresBundleWithoutTargetFlag(t *testing.T) {
	testutil.CleanupEnvironment(t)

	rootDir := t.TempDir()
	testutil.Chdir(t, rootDir)

	err := os.WriteFile(filepath.Join(rootDir, "databricks.yml"), []byte(`
bundle:
  name: test
targets:
  dev:
    default: true
    workspace:
      host: https://dev-environment.cloud.databricks.com
`), 0o644)
	require.NoError(t, err)

	t.Setenv("DATABRICKS_HOST", "https://stg-environment.cloud.databricks.com")
	t.Setenv("DATABRICKS_TOKEN", "stg-token")

	cmd := newWorkspaceAuthTestCommand()
	err = MustWorkspaceClient(cmd, nil)
	require.NoError(t, err)

	cfg := cmdctx.ConfigUsed(cmd.Context())
	require.NotNil(t, cfg)
	require.Equal(t, "https://stg-environment.cloud.databricks.com", cfg.Host)
}

func TestMustWorkspaceClientUsesBundleWhenTargetFlagIsSet(t *testing.T) {
	testutil.CleanupEnvironment(t)

	rootDir := t.TempDir()
	testutil.Chdir(t, rootDir)

	err := os.WriteFile(filepath.Join(rootDir, "databricks.yml"), []byte(`
bundle:
  name: test
targets:
  dev:
    default: true
    workspace:
      host: https://dev-environment.cloud.databricks.com
`), 0o644)
	require.NoError(t, err)

	t.Setenv("DATABRICKS_HOST", "https://stg-environment.cloud.databricks.com")
	t.Setenv("DATABRICKS_TOKEN", "stg-token")

	cmd := newWorkspaceAuthTestCommand()
	err = cmd.Flag("target").Value.Set("dev")
	require.NoError(t, err)

	err = MustWorkspaceClient(cmd, nil)
	require.NoError(t, err)

	cfg := cmdctx.ConfigUsed(cmd.Context())
	require.NotNil(t, cfg)
	require.Equal(t, "https://dev-environment.cloud.databricks.com", cfg.Host)
}
