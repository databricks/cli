package auth

import (
	"errors"
	"testing"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// stubCache is a test double for cache.TokenCache that records the source
// it was constructed from. It lets the tests confirm which factory ran.
type stubCache struct{ source string }

func (stubCache) Store(string, *oauth2.Token) error    { return nil }
func (stubCache) Lookup(string) (*oauth2.Token, error) { return nil, cache.ErrNotFound }

func fakeFactories(t *testing.T) cacheFactories {
	t.Helper()
	return cacheFactories{
		newFile:    func() (cache.TokenCache, error) { return stubCache{source: "file"}, nil },
		newKeyring: func() cache.TokenCache { return stubCache{source: "keyring"} },
	}
}

func TestNewAuthCache_DefaultsToLegacyFile(t *testing.T) {
	ctx := t.Context()

	got, mode, err := newAuthCacheWith(ctx, "", fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, storage.StorageModeLegacy, mode)
	assert.Equal(t, "file", got.(stubCache).source)
}

func TestNewAuthCache_OverrideSecureUsesKeyring(t *testing.T) {
	ctx := t.Context()

	got, mode, err := newAuthCacheWith(ctx, storage.StorageModeSecure, fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, storage.StorageModeSecure, mode)
	assert.Equal(t, "keyring", got.(stubCache).source)
}

func TestNewAuthCache_EnvVarSelectsSecure(t *testing.T) {
	ctx := env.Set(t.Context(), storage.EnvVar, "secure")

	got, mode, err := newAuthCacheWith(ctx, "", fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, storage.StorageModeSecure, mode)
	assert.Equal(t, "keyring", got.(stubCache).source)
}

func TestNewAuthCache_PlaintextFallsBackToFile(t *testing.T) {
	ctx := t.Context()

	got, mode, err := newAuthCacheWith(ctx, storage.StorageModePlaintext, fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, storage.StorageModePlaintext, mode)
	assert.Equal(t, "file", got.(stubCache).source)
}

func TestNewAuthCache_InvalidOverrideReturnsError(t *testing.T) {
	ctx := t.Context()

	_, _, err := newAuthCacheWith(ctx, storage.StorageMode("bogus"), fakeFactories(t))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown storage mode "bogus"`)
}

func TestNewAuthCache_InvalidEnvReturnsError(t *testing.T) {
	ctx := env.Set(t.Context(), storage.EnvVar, "bogus")

	_, _, err := newAuthCacheWith(ctx, "", fakeFactories(t))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABRICKS_AUTH_STORAGE")
}

func TestNewAuthCache_FileFactoryErrorPropagates(t *testing.T) {
	ctx := t.Context()
	boom := errors.New("disk full")
	factories := cacheFactories{
		newFile:    func() (cache.TokenCache, error) { return nil, boom },
		newKeyring: func() cache.TokenCache { return stubCache{source: "keyring"} },
	}

	_, _, err := newAuthCacheWith(ctx, storage.StorageModeLegacy, factories)

	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}
