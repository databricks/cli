package terraform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnvVarWithMatchingVersionUnsetReturnsEmpty(t *testing.T) {
	got, err := getEnvVarWithMatchingVersion(t.Context(), "UCM_TEST_ENV", "UCM_TEST_VER", "1.0")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestGetEnvVarWithMatchingVersionPathMissingReturnsEmpty(t *testing.T) {
	ctx := env.Set(t.Context(), "UCM_TEST_ENV", "/definitely/not/a/real/path/xyzzy")
	got, err := getEnvVarWithMatchingVersion(ctx, "UCM_TEST_ENV", "UCM_TEST_VER", "1.0")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestGetEnvVarWithMatchingVersionNoVersionReturnsValue(t *testing.T) {
	f := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))
	ctx := env.Set(t.Context(), "UCM_TEST_ENV", f)
	got, err := getEnvVarWithMatchingVersion(ctx, "UCM_TEST_ENV", "UCM_TEST_VER", "1.0")
	require.NoError(t, err)
	assert.Equal(t, f, got)
}

func TestGetEnvVarWithMatchingVersionMatchingVersionReturnsValue(t *testing.T) {
	f := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))
	ctx := env.Set(t.Context(), "UCM_TEST_ENV", f)
	ctx = env.Set(ctx, "UCM_TEST_VER", "1.0")
	got, err := getEnvVarWithMatchingVersion(ctx, "UCM_TEST_ENV", "UCM_TEST_VER", "1.0")
	require.NoError(t, err)
	assert.Equal(t, f, got)
}

func TestGetEnvVarWithMatchingVersionMismatchReturnsEmpty(t *testing.T) {
	f := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))
	ctx := env.Set(t.Context(), "UCM_TEST_ENV", f)
	ctx = env.Set(ctx, "UCM_TEST_VER", "0.9")
	got, err := getEnvVarWithMatchingVersion(ctx, "UCM_TEST_ENV", "UCM_TEST_VER", "1.0")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestInheritEnvVarsPassesThroughEnvCopy(t *testing.T) {
	ctx := t.Context()
	for _, k := range []string{
		"HOME",
		"USERPROFILE",
		"PATH",
		"AZURE_CONFIG_DIR",
		"TF_CLI_CONFIG_FILE",
		"USE_SDK_V2_RESOURCES",
		"USE_SDK_V2_DATA_SOURCES",
	} {
		ctx = env.Set(ctx, k, "val-"+k)
	}

	out := map[string]string{}
	require.NoError(t, inheritEnvVars(ctx, out))

	for _, k := range []string{
		"HOME",
		"USERPROFILE",
		"PATH",
		"AZURE_CONFIG_DIR",
		"TF_CLI_CONFIG_FILE",
		"USE_SDK_V2_RESOURCES",
		"USE_SDK_V2_DATA_SOURCES",
	} {
		assert.Equal(t, "val-"+k, out[k], "expected %s pass-through", k)
	}
}

func TestInheritEnvVarsDirectOidcToken(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_OIDC_TOKEN", "direct-token")

	out := map[string]string{}
	require.NoError(t, inheritEnvVars(ctx, out))

	assert.Equal(t, "direct-token", out["DATABRICKS_OIDC_TOKEN"])
	_, hasIndirect := out["DATABRICKS_OIDC_TOKEN_ENV"]
	assert.False(t, hasIndirect)
}

func TestInheritEnvVarsIndirectOidcToken(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_OIDC_TOKEN_ENV", "CUSTOM_OIDC_VAR")
	ctx = env.Set(ctx, "CUSTOM_OIDC_VAR", "custom-token")

	out := map[string]string{}
	require.NoError(t, inheritEnvVars(ctx, out))

	assert.Equal(t, "CUSTOM_OIDC_VAR", out["DATABRICKS_OIDC_TOKEN_ENV"])
	assert.Equal(t, "custom-token", out["CUSTOM_OIDC_VAR"])
}

func TestInheritEnvVarsAzureDevOpsSystemVars(t *testing.T) {
	ctx := t.Context()
	vars := []string{
		"SYSTEM_ACCESSTOKEN",
		"SYSTEM_COLLECTIONID",
		"SYSTEM_COLLECTIONURI",
		"SYSTEM_DEFINITIONID",
		"SYSTEM_HOSTTYPE",
		"SYSTEM_JOBID",
		"SYSTEM_OIDCREQUESTURI",
		"SYSTEM_PLANID",
		"SYSTEM_TEAMFOUNDATIONCOLLECTIONURI",
		"SYSTEM_TEAMPROJECT",
		"SYSTEM_TEAMPROJECTID",
	}
	for _, k := range vars {
		ctx = env.Set(ctx, k, "val-"+k)
	}

	out := map[string]string{}
	require.NoError(t, inheritEnvVars(ctx, out))

	for _, k := range vars {
		assert.Equal(t, "val-"+k, out[k], "expected %s forward", k)
	}
}

func TestInheritEnvVarsMapsCliConfigFileWhenVersionMatches(t *testing.T) {
	f := filepath.Join(t.TempDir(), "cliconfig")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))

	ctx := env.Set(t.Context(), CliConfigPathEnv, f)
	ctx = env.Set(ctx, ProviderVersionEnv, ProviderVersion)

	out := map[string]string{}
	require.NoError(t, inheritEnvVars(ctx, out))

	assert.Equal(t, f, out["TF_CLI_CONFIG_FILE"])
}

func TestInheritEnvVarsSkipsCliConfigFileOnVersionMismatch(t *testing.T) {
	f := filepath.Join(t.TempDir(), "cliconfig")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))

	ctx := env.Set(t.Context(), CliConfigPathEnv, f)
	ctx = env.Set(ctx, ProviderVersionEnv, "0.0.0-mismatch")

	out := map[string]string{}
	require.NoError(t, inheritEnvVars(ctx, out))

	_, ok := out["TF_CLI_CONFIG_FILE"]
	assert.False(t, ok, "mismatched version must not emit TF_CLI_CONFIG_FILE")
}
