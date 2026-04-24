package terraform

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildEnvForwardsAuthFromAuthConfig pins the wire-level DATABRICKS_*
// env vars we expect to forward to terraform — they reach the subprocess
// via the resolved SDK auth config (auth.Env), NOT via parent-env
// passthrough. Mirrors DAB ordering.
func TestBuildEnvForwardsAuthFromAuthConfig(t *testing.T) {
	u, _ := newRenderUcm(t)
	authCfg := &config.Config{
		Host:  "https://example.cloud.databricks.com",
		Token: "resolved-token",
	}

	got, err := buildEnv(t.Context(), u, authCfg)
	require.NoError(t, err)

	assert.Equal(t, "https://example.cloud.databricks.com", got["DATABRICKS_HOST"])
	assert.Equal(t, "resolved-token", got["DATABRICKS_TOKEN"])
}

// TestBuildEnvDropsCloudCreds pins the strict-DAB-alignment decision:
// AWS/Azure/GCP cloud-underlay credentials are NOT forwarded to terraform.
// Revisit when UCM gains resources that actually need them (tracked
// issue).
func TestBuildEnvDropsCloudCreds(t *testing.T) {
	u, _ := newRenderUcm(t)
	ctx := env.Set(t.Context(), "AWS_ACCESS_KEY_ID", "AKIA...")
	ctx = env.Set(ctx, "AZURE_TENANT_ID", "azure-tenant")
	ctx = env.Set(ctx, "GOOGLE_CREDENTIALS", `{"type":"service_account"}`)

	got, err := buildEnv(ctx, u, nil)
	require.NoError(t, err)

	for _, k := range []string{
		"AWS_ACCESS_KEY_ID",
		"AZURE_TENANT_ID",
		"GOOGLE_CREDENTIALS",
	} {
		_, ok := got[k]
		assert.Falsef(t, ok, "%s must not be forwarded (strict DAB alignment)", k)
	}
}

// TestBuildEnvWithoutAuthConfigDropsDatabricksHost pins that parent-env
// DATABRICKS_HOST reaches terraform only via SDK config → auth.Env. UCM
// no longer passes DATABRICKS_HOST through from the parent env directly;
// the SDK's config resolution does that on our behalf.
func TestBuildEnvWithoutAuthConfigDropsDatabricksHost(t *testing.T) {
	u, _ := newRenderUcm(t)
	ctx := env.Set(t.Context(), "DATABRICKS_HOST", "https://parent.cloud.databricks.com")

	got, err := buildEnv(ctx, u, nil)
	require.NoError(t, err)

	_, ok := got["DATABRICKS_HOST"]
	assert.False(t, ok,
		"DATABRICKS_HOST reaches terraform only via auth.Env; parent-env passthrough was dropped for DAB alignment")
}

// TestBuildEnvMapsProxyVarsUppercase verifies setProxyEnvVars is wired
// into buildEnv.
func TestBuildEnvMapsProxyVarsUppercase(t *testing.T) {
	u, _ := newRenderUcm(t)
	ctx := env.Set(t.Context(), "http_proxy", "http://proxy.example:3128")
	ctx = env.Set(ctx, "HTTPS_PROXY", "http://proxy.example:3129")

	got, err := buildEnv(ctx, u, nil)
	require.NoError(t, err)

	assert.Equal(t, "http://proxy.example:3128", got["HTTP_PROXY"])
	assert.Equal(t, "http://proxy.example:3129", got["HTTPS_PROXY"])
}

// TestBuildEnvResolvedAuthOverridesPassthrough pins the DAB ordering:
// auth.Env seeds the map first; inheritEnvVars / other passthrough
// cannot override DATABRICKS_* already set by auth resolution.
func TestBuildEnvResolvedAuthOverridesPassthrough(t *testing.T) {
	u, _ := newRenderUcm(t)
	ctx := env.Set(t.Context(), "DATABRICKS_HOST", "https://stale.cloud.databricks.com")
	authCfg := &config.Config{Host: "https://profile.cloud.databricks.com"}
	got, err := buildEnv(ctx, u, authCfg)
	require.NoError(t, err)

	assert.Equal(t, "https://profile.cloud.databricks.com", got["DATABRICKS_HOST"])
}

// TestBuildEnvAbsolutizesRelativeDatabricksCliPath pins the UCM-local
// fix for the shared DAB bug where DATABRICKS_CLI_PATH can be a
// relative path that fails to resolve from terraform's working dir.
func TestBuildEnvAbsolutizesRelativeDatabricksCliPath(t *testing.T) {
	u, _ := newRenderUcm(t)
	ctx := env.Set(t.Context(), "DATABRICKS_CLI_PATH", "../cli/cli")

	got, err := buildEnv(ctx, u, nil)
	require.NoError(t, err)

	assert.True(t, filepath.IsAbs(got["DATABRICKS_CLI_PATH"]),
		"expected absolute path, got %q", got["DATABRICKS_CLI_PATH"])
}

func TestLockIdentityDerivesPathFromTarget(t *testing.T) {
	u, _ := newRenderUcm(t)
	user, dir := lockIdentity(t.Context(), u)
	require.NotEmpty(t, user)
	assert.Contains(t, dir, ".databricks")
	assert.Contains(t, dir, "ucm")
	assert.Contains(t, dir, "dev")
	assert.Contains(t, dir, "state")
}
