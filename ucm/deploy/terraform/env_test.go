package terraform

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestSetTempDirEnvVarsUnixInheritsTMPDIR(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only branch")
	}
	u, _ := newRenderUcm(t)
	ctx := env.Set(t.Context(), "TMPDIR", "/custom/tmp")

	out := map[string]string{}
	require.NoError(t, setTempDirEnvVars(ctx, out, u))

	assert.Equal(t, "/custom/tmp", out["TMPDIR"])
}

func TestSetTempDirEnvVarsUnixOmitsWhenUnset(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only branch")
	}
	u, _ := newRenderUcm(t)
	// Ensure TMPDIR is truly absent from both the context and the process env.
	t.Setenv("TMPDIR", "")
	require.NoError(t, os.Unsetenv("TMPDIR"))
	ctx := t.Context()

	out := map[string]string{}
	require.NoError(t, setTempDirEnvVars(ctx, out, u))

	_, ok := out["TMPDIR"]
	assert.False(t, ok)
}

func TestSetTempDirEnvVarsWindowsInheritsTMP(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only branch")
	}
	u, _ := newRenderUcm(t)
	ctx := env.Set(t.Context(), "TMP", `C:\custom\tmp`)

	out := map[string]string{}
	require.NoError(t, setTempDirEnvVars(ctx, out, u))

	assert.Equal(t, `C:\custom\tmp`, out["TMP"])
}

func TestSetTempDirEnvVarsWindowsFallsBackToLocalStateDir(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only branch")
	}
	u, root := newRenderUcm(t)
	ctx := t.Context()

	out := map[string]string{}
	require.NoError(t, setTempDirEnvVars(ctx, out, u))

	want := filepath.Join(root, ".databricks", "ucm", "dev", "tmp")
	assert.Equal(t, want, out["TMP"])
}

func TestSetProxyEnvVarsUpperCase(t *testing.T) {
	ctx := env.Set(t.Context(), "HTTP_PROXY", "http://up:3128")
	ctx = env.Set(ctx, "HTTPS_PROXY", "http://up:3129")
	ctx = env.Set(ctx, "NO_PROXY", "localhost")

	out := map[string]string{}
	require.NoError(t, setProxyEnvVars(ctx, out))

	assert.Equal(t, "http://up:3128", out["HTTP_PROXY"])
	assert.Equal(t, "http://up:3129", out["HTTPS_PROXY"])
	assert.Equal(t, "localhost", out["NO_PROXY"])
}

func TestSetProxyEnvVarsLowerCaseNormalizedToUpper(t *testing.T) {
	ctx := env.Set(t.Context(), "http_proxy", "http://low:3128")
	ctx = env.Set(ctx, "https_proxy", "http://low:3129")
	ctx = env.Set(ctx, "no_proxy", "127.0.0.1")

	out := map[string]string{}
	require.NoError(t, setProxyEnvVars(ctx, out))

	assert.Equal(t, "http://low:3128", out["HTTP_PROXY"])
	assert.Equal(t, "http://low:3129", out["HTTPS_PROXY"])
	assert.Equal(t, "127.0.0.1", out["NO_PROXY"])
}

func TestSetProxyEnvVarsOmitsUnset(t *testing.T) {
	out := map[string]string{}
	require.NoError(t, setProxyEnvVars(t.Context(), out))

	for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"} {
		_, ok := out[k]
		assert.False(t, ok, "%s should not be set when neither case is on env", k)
	}
}
