package sqlsafe

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/require"

	"github.com/databricks/cli/libs/env"
)

var helper = OverrideHelper{
	FlagName:   "--allow-destructive",
	EnvVar:     "DATABRICKS_CLI_ALLOW_DESTRUCTIVE_SQL",
	ProfileKey: "cli.allow_destructive_sql",
}

func TestOverrideResolveFlagPrecedence(t *testing.T) {
	ctx := context.Background()
	allowed, err := helper.Resolve(ctx, &config.Config{}, true, true)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestOverrideResolveEnv(t *testing.T) {
	ctx := env.Set(context.Background(), helper.EnvVar, "true")
	allowed, err := helper.Resolve(ctx, &config.Config{}, false, false)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestOverrideEnvOverridesFlag(t *testing.T) {
	ctx := env.Set(context.Background(), helper.EnvVar, "false")
	allowed, err := helper.Resolve(ctx, &config.Config{}, true, true)
	require.NoError(t, err)
	require.False(t, allowed)

	ctx = env.Set(context.Background(), helper.EnvVar, "true")
	allowed, err = helper.Resolve(ctx, &config.Config{}, true, false)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestOverrideResolveEnvInvalid(t *testing.T) {
	ctx := env.Set(context.Background(), helper.EnvVar, "nope")
	_, err := helper.Resolve(ctx, &config.Config{}, false, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), helper.EnvVar)
}

func TestOverrideResolveProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "databrickscfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("[DEFAULT]\ncli.allow_destructive_sql = true\n"), 0o600))

	ctx := env.Set(context.Background(), "DATABRICKS_CONFIG_FILE", cfgPath)
	allowed, err := helper.Resolve(ctx, &config.Config{Profile: "DEFAULT"}, false, false)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestOverrideProfileOverridesEnvAndFlag(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "databrickscfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("[DEFAULT]\ncli.allow_destructive_sql = false\n"), 0o600))

	ctx := env.Set(context.Background(), "DATABRICKS_CONFIG_FILE", cfgPath)
	ctx = env.Set(ctx, helper.EnvVar, "true")
	allowed, err := helper.Resolve(ctx, &config.Config{Profile: "DEFAULT"}, true, true)
	require.NoError(t, err)
	require.False(t, allowed)
}

func TestOverrideResolveProfileInvalidBool(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "databrickscfg")
	require.NoError(t, os.WriteFile(cfgPath, []byte("[DEFAULT]\ncli.allow_destructive_sql = maybe\n"), 0o600))

	ctx := env.Set(context.Background(), "DATABRICKS_CONFIG_FILE", cfgPath)
	_, err := helper.Resolve(ctx, &config.Config{Profile: "DEFAULT"}, false, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), helper.ProfileKey)
}

func TestOverrideBlockedErrorMessage(t *testing.T) {
	baseErr := context.DeadlineExceeded
	err := helper.BlockedError(baseErr)
	require.Contains(t, err.Error(), helper.FlagName)
	require.Contains(t, err.Error(), helper.EnvVar)
	require.Contains(t, err.Error(), helper.ProfileKey)
}
