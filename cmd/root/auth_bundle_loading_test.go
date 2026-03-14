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

const (
	devHost = "https://dev-environment.cloud.databricks.com"
	stgHost = "https://stg-environment.cloud.databricks.com"
)

func newWorkspaceAuthTestCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	initProfileFlag(cmd)
	initTargetFlag(cmd)
	initEnvironmentFlag(cmd)
	return cmd
}

func setupWorkspaceAuthFixture(t *testing.T) {
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
  stg:
    workspace:
      host: https://stg-environment.cloud.databricks.com
`), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(rootDir, ".databrickscfg"), []byte(`
[DEV]
host = https://dev-environment.cloud.databricks.com
token = dev-token

[STG]
host = https://stg-environment.cloud.databricks.com
token = stg-token
`), 0o644)
	require.NoError(t, err)

	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(rootDir, ".databrickscfg"))
}

func assertConfigUsedHost(t *testing.T, cmd *cobra.Command, expectedHost string) {
	cfg := cmdctx.ConfigUsed(cmd.Context())
	require.NotNil(t, cfg)
	require.Equal(t, expectedHost, cfg.Host)
}

func TestMustWorkspaceClientUsesBundleDefaultWithoutExplicitOverride(t *testing.T) {
	setupWorkspaceAuthFixture(t)

	cmd := newWorkspaceAuthTestCommand()
	err := MustWorkspaceClient(cmd, nil)
	require.NoError(t, err)

	assertConfigUsedHost(t, cmd, devHost)
}

func TestMustWorkspaceClientIgnoresBundleWhenHostEnvIsSetWithoutTargetFlag(t *testing.T) {
	setupWorkspaceAuthFixture(t)

	t.Setenv("DATABRICKS_HOST", stgHost)
	t.Setenv("DATABRICKS_TOKEN", "stg-token")

	cmd := newWorkspaceAuthTestCommand()
	err := MustWorkspaceClient(cmd, nil)
	require.NoError(t, err)

	assertConfigUsedHost(t, cmd, stgHost)
}

func TestMustWorkspaceClientIgnoresBundleWhenProfileEnvIsSetWithoutTargetFlag(t *testing.T) {
	setupWorkspaceAuthFixture(t)

	t.Setenv("DATABRICKS_CONFIG_PROFILE", "STG")

	cmd := newWorkspaceAuthTestCommand()
	err := MustWorkspaceClient(cmd, nil)
	require.NoError(t, err)

	assertConfigUsedHost(t, cmd, stgHost)
}

func TestMustWorkspaceClientUsesBundleWhenTargetFlagIsSetWithExplicitEnv(t *testing.T) {
	setupWorkspaceAuthFixture(t)

	t.Setenv("DATABRICKS_HOST", stgHost)
	t.Setenv("DATABRICKS_TOKEN", "stg-token")
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "DEV")

	cmd := newWorkspaceAuthTestCommand()
	err := cmd.Flag("target").Value.Set("dev")
	require.NoError(t, err)

	err = MustWorkspaceClient(cmd, nil)
	require.NoError(t, err)

	assertConfigUsedHost(t, cmd, devHost)
}

func TestMustWorkspaceClientUsesBundleWhenEnvironmentFlagIsSetWithExplicitEnv(t *testing.T) {
	setupWorkspaceAuthFixture(t)

	t.Setenv("DATABRICKS_HOST", stgHost)
	t.Setenv("DATABRICKS_TOKEN", "stg-token")
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "DEV")

	cmd := newWorkspaceAuthTestCommand()
	err := cmd.Flag("environment").Value.Set("dev")
	require.NoError(t, err)

	err = MustWorkspaceClient(cmd, nil)
	require.NoError(t, err)

	assertConfigUsedHost(t, cmd, devHost)
}

func TestMustWorkspaceClientKeepsBundleDefaultWhenOnlyNonAuthEnvIsSet(t *testing.T) {
	setupWorkspaceAuthFixture(t)

	t.Setenv("DATABRICKS_RATE_LIMIT", "500")

	cmd := newWorkspaceAuthTestCommand()
	err := MustWorkspaceClient(cmd, nil)
	require.NoError(t, err)

	assertConfigUsedHost(t, cmd, devHost)
}

func TestMustWorkspaceClientKeepsBundleDefaultWhenOnlyCliPathEnvIsSet(t *testing.T) {
	setupWorkspaceAuthFixture(t)

	t.Setenv("DATABRICKS_CLI_PATH", "/usr/local/bin/databricks")

	cmd := newWorkspaceAuthTestCommand()
	err := MustWorkspaceClient(cmd, nil)
	require.NoError(t, err)

	assertConfigUsedHost(t, cmd, devHost)
}
