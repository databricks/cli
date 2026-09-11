package aircmd

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeDockerConfig writes config.json into a temp dir and returns a context with
// DOCKER_CONFIG pointing at it.
func writeDockerConfig(t *testing.T, body string) context.Context {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600))
	return env.Set(t.Context(), "DOCKER_CONFIG", dir)
}

func b64(t *testing.T, s string) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func TestRegistryKey(t *testing.T) {
	assert.Equal(t, dockerHubAuthKey, registryKey("docker.io/library/ubuntu:latest"))
	assert.Equal(t, dockerHubAuthKey, registryKey("index.docker.io/x"))
	assert.Equal(t, "nvcr.io", registryKey("nvcr.io/nvidia/pytorch:24.01"))
	assert.Equal(t, "ghcr.io", registryKey("ghcr.io/org/img"))
}

func TestDecodeDockerAuth(t *testing.T) {
	u, s, ok := decodeDockerAuth(base64.StdEncoding.EncodeToString([]byte("alice:pat123")))
	require.True(t, ok)
	assert.Equal(t, "alice", u)
	assert.Equal(t, "pat123", s)

	_, _, ok = decodeDockerAuth("not-base64!!")
	assert.False(t, ok)
	_, _, ok = decodeDockerAuth(base64.StdEncoding.EncodeToString([]byte("noseparator")))
	assert.False(t, ok)
}

func TestReadDockerCredentialsInlineAuth(t *testing.T) {
	ctx := writeDockerConfig(t, `{"auths":{"nvcr.io":{"auth":"`+b64(t, "bob:secret")+`"}}}`)
	u, s, ok := readDockerCredentials(ctx, "nvcr.io/nvidia/pytorch:24.01")
	require.True(t, ok)
	assert.Equal(t, "bob", u)
	assert.Equal(t, "secret", s)
}

func TestReadDockerCredentialsDockerHubLegacyKey(t *testing.T) {
	ctx := writeDockerConfig(t, `{"auths":{"https://index.docker.io/v1/":{"auth":"`+b64(t, "carol:tok")+`"}}}`)
	u, _, ok := readDockerCredentials(ctx, "docker.io/library/ubuntu:latest")
	require.True(t, ok)
	assert.Equal(t, "carol", u)
}

func TestReadDockerCredentialsNeedsNormalizedURL(t *testing.T) {
	// A bare "ubuntu" has no registry host, so it must be normalized before
	// lookup; discoverCredentials normalizes so Docker Hub creds are found.
	ctx := writeDockerConfig(t, `{"auths":{"https://index.docker.io/v1/":{"auth":"`+b64(t, "dave:tok")+`"}}}`)
	_, _, ok := readDockerCredentials(ctx, "ubuntu")
	assert.False(t, ok)
	u, _, ok := readDockerCredentials(ctx, normalizeDockerImageURL("ubuntu"))
	require.True(t, ok)
	assert.Equal(t, "dave", u)
}

func TestReadDockerCredentialsMissingRegistry(t *testing.T) {
	ctx := writeDockerConfig(t, `{"auths":{"nvcr.io":{"auth":"`+b64(t, "bob:secret")+`"}}}`)
	_, _, ok := readDockerCredentials(ctx, "ghcr.io/org/img:latest")
	assert.False(t, ok)
}

func TestReadDockerCredentialsNoConfigFile(t *testing.T) {
	// DOCKER_CONFIG points at an empty dir with no config.json.
	ctx := env.Set(t.Context(), "DOCKER_CONFIG", t.TempDir())
	_, _, ok := readDockerCredentials(ctx, "nvcr.io/img:latest")
	assert.False(t, ok)
}

func TestReadDockerCredentialsCredHelper(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("credential helper stub is a POSIX shell script")
	}
	// A credHelper for the registry takes precedence over inline auth. The stub
	// echoes a fixed Username/Secret payload on the `get` protocol.
	binDir := t.TempDir()
	stub := filepath.Join(binDir, "docker-credential-airtest")
	require.NoError(t, os.WriteFile(stub, []byte("#!/bin/sh\necho '{\"Username\":\"helperuser\",\"Secret\":\"helpersecret\"}'\n"), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	ctx := writeDockerConfig(t, `{"credHelpers":{"nvcr.io":"airtest"},"auths":{"nvcr.io":{"auth":"`+b64(t, "inline:ignored")+`"}}}`)
	u, s, ok := readDockerCredentials(ctx, "nvcr.io/nvidia/pytorch:24.01")
	require.True(t, ok)
	assert.Equal(t, "helperuser", u)
	assert.Equal(t, "helpersecret", s)
}
