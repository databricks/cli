package root

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyHttpRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := emptyHttpRequest(ctx)
	assert.Equal(t, req.Context(), ctx)
}

type promptFn func(ctx context.Context, cfg *config.Config, retry bool) (any, error)

var accountPromptFn = func(ctx context.Context, cfg *config.Config, retry bool) (any, error) {
	return accountClientOrPrompt(ctx, cfg, retry)
}

var workspacePromptFn = func(ctx context.Context, cfg *config.Config, retry bool) (any, error) {
	return workspaceClientOrPrompt(ctx, cfg, retry)
}

func expectPrompts(t *testing.T, fn promptFn, config *config.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Channel to pass errors from the prompting function back to the test.
	errch := make(chan error, 1)

	ctx, io := cmdio.SetupTest(ctx)
	go func() {
		defer close(errch)
		defer cancel()
		_, err := fn(ctx, config, true)
		errch <- err
	}()

	// Expect a prompt
	line, _, err := io.Stderr.ReadLine()
	if assert.NoError(t, err, "Expected to read a line from stderr") {
		assert.Contains(t, string(line), "Search:")
	} else {
		// If there was an error reading from stderr, the prompting function must have terminated early.
		assert.NoError(t, <-errch)
	}
}

func expectReturns(t *testing.T, fn promptFn, config *config.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	ctx, _ = cmdio.SetupTest(ctx)
	client, err := fn(ctx, config, true)
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestAccountClientOrPrompt(t *testing.T) {
	testutil.CleanupEnvironment(t)

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(`
			[account-1111]
			host = https://accounts.azuredatabricks.net/
			account_id = 1111
			token = foobar

			[account-1112]
			host = https://accounts.azuredatabricks.net/
			account_id = 1112
			token = foobar
			`),
		0o755)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("PATH", "/nothing")

	t.Run("Prompt if nothing is specified", func(t *testing.T) {
		expectPrompts(t, accountPromptFn, &config.Config{})
	})

	t.Run("Prompt if a workspace host is specified", func(t *testing.T) {
		expectPrompts(t, accountPromptFn, &config.Config{
			Host:      "https://adb-1234567.89.azuredatabricks.net/",
			AccountID: "1234",
			Token:     "foobar",
		})
	})

	t.Run("Prompt if account ID is not specified", func(t *testing.T) {
		expectPrompts(t, accountPromptFn, &config.Config{
			Host:  "https://accounts.azuredatabricks.net/",
			Token: "foobar",
		})
	})

	t.Run("Prompt if no credential provider can be configured", func(t *testing.T) {
		// The SDK probes all auth types when not specified and this fails for the u2m probe on Windows.
		t.Skip("Skipping as of #2920")

		expectPrompts(t, accountPromptFn, &config.Config{
			Host:      "https://accounts.azuredatabricks.net/",
			AccountID: "1234",

			// Force SDK to not try and lookup the tenant ID from the host.
			// The host above is invalid and will not be reachable.
			AzureTenantID: "nonempty",
		})
	})

	t.Run("Returns if configuration is valid", func(t *testing.T) {
		expectReturns(t, accountPromptFn, &config.Config{
			Host:      "https://accounts.azuredatabricks.net/",
			AccountID: "1234",
			Token:     "foobar",
		})
	})

	t.Run("Returns if a valid profile is specified", func(t *testing.T) {
		expectReturns(t, accountPromptFn, &config.Config{
			Profile: "account-1111",
		})
	})
}

func TestWorkspaceClientOrPrompt(t *testing.T) {
	testutil.CleanupEnvironment(t)

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(`
			[workspace-1111]
			host = https://adb-1111.11.azuredatabricks.net/
			token = foobar

			[workspace-1112]
			host = https://adb-1112.12.azuredatabricks.net/
			token = foobar
			`),
		0o755)
	require.NoError(t, err)
	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	t.Setenv("PATH", "/nothing")

	t.Run("Prompt if nothing is specified", func(t *testing.T) {
		expectPrompts(t, workspacePromptFn, &config.Config{})
	})

	t.Run("Prompt if an account host is specified", func(t *testing.T) {
		expectPrompts(t, workspacePromptFn, &config.Config{
			Host:      "https://accounts.azuredatabricks.net/",
			AccountID: "1234",
			Token:     "foobar",
		})
	})

	t.Run("Prompt if no credential provider can be configured", func(t *testing.T) {
		// The SDK probes all auth types when not specified and this fails for the u2m probe on Windows.
		t.Skip("Skipping as of #2920")

		expectPrompts(t, workspacePromptFn, &config.Config{
			Host: "https://adb-1111.11.azuredatabricks.net/",

			// Force SDK to not try and lookup the tenant ID from the host.
			// The host above is invalid and will not be reachable.
			AzureTenantID: "nonempty",
		})
	})

	t.Run("Returns if configuration is valid", func(t *testing.T) {
		expectReturns(t, workspacePromptFn, &config.Config{
			Host:  "https://adb-1111.11.azuredatabricks.net/",
			Token: "foobar",
		})
	})

	t.Run("Returns if a valid profile is specified", func(t *testing.T) {
		expectReturns(t, workspacePromptFn, &config.Config{
			Profile: "workspace-1111",
		})
	})
}

func TestMustAccountClientWorksWithDatabricksCfg(t *testing.T) {
	testutil.CleanupEnvironment(t)

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(`
			[account-1111]
			host = https://accounts.azuredatabricks.net/
			account_id = 1111
			token = foobar
			`),
		0o755)
	require.NoError(t, err)

	cmd := New(context.Background())

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	err = MustAccountClient(cmd, []string{})
	require.NoError(t, err)
}

func TestMustAccountClientWorksWithNoDatabricksCfgButEnvironmentVariables(t *testing.T) {
	testutil.CleanupEnvironment(t)

	ctx, tt := cmdio.SetupTest(context.Background())
	t.Cleanup(tt.Done)
	cmd := New(ctx)
	t.Setenv("DATABRICKS_HOST", "https://accounts.azuredatabricks.net/")
	t.Setenv("DATABRICKS_TOKEN", "foobar")
	t.Setenv("DATABRICKS_ACCOUNT_ID", "1111")

	err := MustAccountClient(cmd, []string{})
	require.NoError(t, err)
}

func TestMustAccountClientErrorsWithNoDatabricksCfg(t *testing.T) {
	testutil.CleanupEnvironment(t)

	ctx, tt := cmdio.SetupTest(context.Background())
	t.Cleanup(tt.Done)
	cmd := New(ctx)

	err := MustAccountClient(cmd, []string{})
	require.ErrorContains(t, err, "no configuration file found at")
}

func TestMustAnyClientCanCreateWorkspaceClient(t *testing.T) {
	testutil.CleanupEnvironment(t)

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(`
			[workspace-1111]
			host = https://adb-1111.11.azuredatabricks.net/
			token = foobar
			`),
		0o755)
	require.NoError(t, err)

	ctx, tt := cmdio.SetupTest(context.Background())
	t.Cleanup(tt.Done)
	cmd := New(ctx)

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	isAccount, err := MustAnyClient(cmd, []string{})
	require.False(t, isAccount)
	require.NoError(t, err)

	w := cmdctx.WorkspaceClient(cmd.Context())
	require.NotNil(t, w)
}

func TestMustAnyClientCanCreateAccountClient(t *testing.T) {
	testutil.CleanupEnvironment(t)

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(`
			[account-1111]
			host = https://accounts.azuredatabricks.net/
			account_id = 1111
			token = foobar
			`),
		0o755)
	require.NoError(t, err)

	ctx, tt := cmdio.SetupTest(context.Background())
	t.Cleanup(tt.Done)
	cmd := New(ctx)

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)
	isAccount, err := MustAnyClient(cmd, []string{})
	require.NoError(t, err)
	require.True(t, isAccount)

	a := cmdctx.AccountClient(cmd.Context())
	require.NotNil(t, a)
}

func TestMustAnyClientWithEmptyDatabricksCfg(t *testing.T) {
	testutil.CleanupEnvironment(t)

	dir := t.TempDir()
	configFile := filepath.Join(dir, ".databrickscfg")
	err := os.WriteFile(
		configFile,
		[]byte(""), // empty file
		0o755)
	require.NoError(t, err)

	ctx, tt := cmdio.SetupTest(context.Background())
	t.Cleanup(tt.Done)
	cmd := New(ctx)

	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

	_, err = MustAnyClient(cmd, []string{})
	require.ErrorContains(t, err, "does not contain account profiles")
}
