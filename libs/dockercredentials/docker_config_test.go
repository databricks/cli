package dockercredentials

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRegistryHost = "123.container.us-west-2.cloud.databricks.test"

func readDockerConfigForTest(t *testing.T, path string) map[string]any {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	return got
}

func TestConfigureDockerCredentialHelperCreatesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docker", "config.json")

	require.NoError(t, SetCredentialHelper(path, testRegistryHost))

	got := readDockerConfigForTest(t, path)
	assert.Equal(t, map[string]any{
		testRegistryHost: HelperName,
	}, got["credHelpers"])

	info, err := os.Stat(path)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestConfigureDockerCredentialHelperPreservesExistingConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
  "auths": {
    "registry.example.test": {"auth": "abc"}
  },
  "credsStore": "desktop",
  "credHelpers": {
    "registry.example.test": "desktop"
  },
  "experimental": "enabled"
}`), 0o600))

	require.NoError(t, SetCredentialHelper(path, testRegistryHost))

	got := readDockerConfigForTest(t, path)
	assert.Equal(t, "desktop", got["credsStore"])
	assert.Equal(t, "enabled", got["experimental"])
	assert.Equal(t, map[string]any{
		"registry.example.test": "desktop",
		testRegistryHost:        HelperName,
	}, got["credHelpers"])
	assert.Contains(t, got, "auths")
}

func TestConfigureDockerCredentialHelperPreservesConfigSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(target, []byte(`{"credsStore":"desktop"}`), 0o600))
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	require.NoError(t, SetCredentialHelper(path, testRegistryHost))

	info, err := os.Lstat(path)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink)
	got := readDockerConfigForTest(t, target)
	assert.Equal(t, "desktop", got["credsStore"])
	assert.Equal(t, map[string]any{testRegistryHost: HelperName}, got["credHelpers"])
}

func TestConfigureDockerCredentialHelperIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
  "credHelpers": {
    "123.container.us-west-2.cloud.databricks.test": "databricks"
  }
}`), 0o600))

	before, err := os.ReadFile(path)
	require.NoError(t, err)

	require.NoError(t, SetCredentialHelper(path, testRegistryHost))

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestConfigureDockerCredentialHelperReplacesExistingHelper(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
  "credHelpers": {
    "123.container.us-west-2.cloud.databricks.test": "desktop"
  }
}`), 0o600))

	require.NoError(t, SetCredentialHelper(path, testRegistryHost))

	got := readDockerConfigForTest(t, path)
	assert.Equal(t, map[string]any{
		testRegistryHost: HelperName,
	}, got["credHelpers"])
}

func TestConfigureDockerCredentialHelperTreatsNullCredHelpersAsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"credHelpers": null}`), 0o600))

	require.NoError(t, SetCredentialHelper(path, testRegistryHost))

	got := readDockerConfigForTest(t, path)
	assert.Equal(t, map[string]any{
		testRegistryHost: HelperName,
	}, got["credHelpers"])
}

func TestConfigureDockerCredentialHelperRejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte("{not valid json"), 0o600))

	err := SetCredentialHelper(path, testRegistryHost)
	assert.ErrorContains(t, err, "read Docker config")
}
