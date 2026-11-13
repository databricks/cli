package sandbox_test

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalIdentityConcurrentCreationAndIsolation(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	const count = 12
	identities := make([]*sandbox.Identity, count)
	failures := make([]error, count)
	var group sync.WaitGroup
	for i := range count {
		group.Go(func() {
			identities[i], failures[i] = sandbox.LocalIdentity(ctx, "https://workspace.test/123", "caller", "sandboxes/test")
		})
	}
	group.Wait()
	for i := range count {
		require.NoError(t, failures[i])
		require.NotNil(t, identities[i])
		assert.Equal(t, identities[0], identities[i])
	}
	first := identities[0]
	for _, scope := range []struct{ workspace, caller, name string }{
		{"https://other.test/123", "caller", "sandboxes/test"},
		{"https://workspace.test/123", "other", "sandboxes/test"},
		{"https://workspace.test/123", "caller", "sandboxes/other"},
	} {
		other, err := sandbox.LocalIdentity(ctx, scope.workspace, scope.caller, scope.name)
		require.NoError(t, err)
		assert.NotEqual(t, first.KeyPath, other.KeyPath)
		assert.NotEqual(t, first.PublicKey, other.PublicKey)
	}
	if runtime.GOOS != "windows" {
		stat, err := os.Stat(first.KeyPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), stat.Mode().Perm())
		stat, err = os.Stat(filepath.Dir(first.KeyPath))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), stat.Mode().Perm())
	}
	require.NoError(t, first.PinHostKey(first.PublicKey))
	content, err := os.ReadFile(first.KnownHostsPath)
	require.NoError(t, err)
	assert.Equal(t, first.HostKeyAlias+" "+string(first.PublicKey), string(content))
	require.NoError(t, os.WriteFile(first.KeyPath, []byte("invalid key"), 0o600))
	_, err = sandbox.LocalIdentity(ctx, "https://workspace.test/123", "caller", "sandboxes/test")
	require.ErrorContains(t, err, "parse local SSH key")
}
