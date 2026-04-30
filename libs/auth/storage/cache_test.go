package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
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
		newFile:    func(context.Context) (cache.TokenCache, error) { return stubCache{source: "file"}, nil },
		newKeyring: func() cache.TokenCache { return stubCache{source: "keyring"} },
	}
}

// hermetic isolates the test from the caller's real env vars and
// .databrickscfg so ResolveStorageMode starts from a clean default.
func hermetic(t *testing.T) {
	t.Helper()
	t.Setenv(EnvVar, "")
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), "databrickscfg"))
}

func TestResolveCache_DefaultsToPlaintextFile(t *testing.T) {
	hermetic(t)
	ctx := t.Context()

	got, mode, err := resolveCacheWith(ctx, "", fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, StorageModePlaintext, mode)
	assert.Equal(t, "file", got.(stubCache).source)
}

func TestResolveCache_OverrideSecureUsesKeyring(t *testing.T) {
	hermetic(t)
	ctx := t.Context()

	got, mode, err := resolveCacheWith(ctx, StorageModeSecure, fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, StorageModeSecure, mode)
	assert.Equal(t, "keyring", got.(stubCache).source)
}

func TestResolveCache_EnvVarSelectsSecure(t *testing.T) {
	hermetic(t)
	ctx := env.Set(t.Context(), EnvVar, "secure")

	got, mode, err := resolveCacheWith(ctx, "", fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, StorageModeSecure, mode)
	assert.Equal(t, "keyring", got.(stubCache).source)
}

func TestResolveCache_PlaintextOverrideUsesFile(t *testing.T) {
	hermetic(t)
	ctx := t.Context()

	got, mode, err := resolveCacheWith(ctx, StorageModePlaintext, fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, StorageModePlaintext, mode)
	assert.Equal(t, "file", got.(stubCache).source)
}

func TestResolveCache_InvalidOverrideReturnsError(t *testing.T) {
	hermetic(t)
	ctx := t.Context()

	_, _, err := resolveCacheWith(ctx, StorageMode("bogus"), fakeFactories(t))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported storage mode "bogus"`)
}

func TestResolveCache_InvalidEnvReturnsError(t *testing.T) {
	hermetic(t)
	ctx := env.Set(t.Context(), EnvVar, "bogus")

	_, _, err := resolveCacheWith(ctx, "", fakeFactories(t))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABRICKS_AUTH_STORAGE")
}

func TestResolveCache_FileFactoryErrorPropagates(t *testing.T) {
	hermetic(t)
	ctx := t.Context()
	boom := errors.New("disk full")
	factories := cacheFactories{
		newFile:    func(context.Context) (cache.TokenCache, error) { return nil, boom },
		newKeyring: func() cache.TokenCache { return stubCache{source: "keyring"} },
	}

	_, _, err := resolveCacheWith(ctx, StorageModePlaintext, factories)

	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestWrapForOAuthArgument(t *testing.T) {
	const (
		host       = "https://example.com"
		profileKey = "myprofile"
	)
	arg, err := u2m.NewProfileWorkspaceOAuthArgument(host, profileKey)
	require.NoError(t, err)

	cases := []struct {
		name        string
		mode        StorageMode
		wantWrap    bool
		wantHostKey bool
	}{
		{"plaintext wraps and mirrors under host key", StorageModePlaintext, true, true},
		{"secure returns inner unchanged; no host-key mirror", StorageModeSecure, false, false},
		{"unknown returns inner unchanged; no host-key mirror", StorageModeUnknown, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inner := newMemoryCache()
			got := WrapForOAuthArgument(inner, tc.mode, arg)

			_, wrapped := got.(*DualWritingTokenCache)
			assert.Equal(t, tc.wantWrap, wrapped, "wrapper presence")

			tok := &oauth2.Token{AccessToken: "abc"}
			require.NoError(t, got.Store(profileKey, tok))

			primary, err := inner.Lookup(profileKey)
			require.NoError(t, err, "primary key must always be written")
			assert.Equal(t, tok, primary)

			_, err = inner.Lookup(host)
			if tc.wantHostKey {
				require.NoError(t, err, "host-key mirror expected in plaintext mode")
			} else {
				assert.ErrorIs(t, err, cache.ErrNotFound, "no host-key mirror expected outside plaintext mode")
			}
		})
	}
}
