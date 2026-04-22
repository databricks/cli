package terraform

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildEnvPassesAuthAndCloudVars pins the wire-level auth and cloud-cred
// env variables we expect to forward to the terraform subprocess. The test
// uses libs/env's context-backed environment so it doesn't pollute the
// process-wide os.Environ.
func TestBuildEnvPassesAuthAndCloudVars(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_HOST", "https://example.cloud.databricks.com")
	ctx = env.Set(ctx, "DATABRICKS_CLIENT_ID", "sp-client-id")
	ctx = env.Set(ctx, "DATABRICKS_CLIENT_SECRET", "sp-client-secret")
	ctx = env.Set(ctx, "AWS_ACCESS_KEY_ID", "AKIA...")
	ctx = env.Set(ctx, "AWS_SECRET_ACCESS_KEY", "secret")
	ctx = env.Set(ctx, "AZURE_TENANT_ID", "azure-tenant")
	ctx = env.Set(ctx, "GOOGLE_CREDENTIALS", `{"type":"service_account"}`)

	got := buildEnv(ctx, nil)

	for _, key := range []string{
		"DATABRICKS_HOST",
		"DATABRICKS_CLIENT_ID",
		"DATABRICKS_CLIENT_SECRET",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AZURE_TENANT_ID",
		"GOOGLE_CREDENTIALS",
	} {
		_, ok := got[key]
		assert.Truef(t, ok, "expected %s to be passed through", key)
	}

	assert.Equal(t, "https://example.cloud.databricks.com", got["DATABRICKS_HOST"])
	assert.Equal(t, "sp-client-id", got["DATABRICKS_CLIENT_ID"])
}

func TestBuildEnvOmitsUnsetVars(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_HOST", "https://example.cloud.databricks.com")
	got := buildEnv(ctx, nil)

	_, ok := got["DATABRICKS_CLIENT_ID"]
	assert.False(t, ok, "unset var should not leak into env map")
	_, ok = got["AWS_ACCESS_KEY_ID"]
	assert.False(t, ok)
}

func TestBuildEnvMapsProxyVarsUppercase(t *testing.T) {
	ctx := env.Set(t.Context(), "http_proxy", "http://proxy.example:3128")
	ctx = env.Set(ctx, "HTTPS_PROXY", "http://proxy.example:3129")

	got := buildEnv(ctx, nil)
	assert.Equal(t, "http://proxy.example:3128", got["HTTP_PROXY"])
	assert.Equal(t, "http://proxy.example:3129", got["HTTPS_PROXY"])
}

// TestBuildEnvMaterializesResolvedAuth pins the behaviour that makes
// `ucm plan`/`ucm deploy` work when auth comes from ~/.databrickscfg
// instead of DATABRICKS_* env vars. The resolved SDK config must be
// serialised into DATABRICKS_* so the terraform subprocess can auth.
func TestBuildEnvMaterializesResolvedAuth(t *testing.T) {
	authCfg := &config.Config{
		Host:  "https://profile.cloud.databricks.com",
		Token: "resolved-token",
	}
	got := buildEnv(t.Context(), authCfg)
	assert.Equal(t, "https://profile.cloud.databricks.com", got["DATABRICKS_HOST"])
	assert.Equal(t, "resolved-token", got["DATABRICKS_TOKEN"])
}

// TestBuildEnvResolvedAuthOverridesPassthrough pins the overlay ordering:
// a resolved --profile host must win over a stale DATABRICKS_HOST that
// happens to be set on the parent env.
func TestBuildEnvResolvedAuthOverridesPassthrough(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_HOST", "https://stale.cloud.databricks.com")
	authCfg := &config.Config{Host: "https://profile.cloud.databricks.com"}
	got := buildEnv(ctx, authCfg)
	assert.Equal(t, "https://profile.cloud.databricks.com", got["DATABRICKS_HOST"])
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
