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
