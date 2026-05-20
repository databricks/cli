package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// missingCache always returns ErrNotFound on Lookup. Lets us drive the
// wrapper without going through the real file or keyring cache.
type missingCache struct{}

func (missingCache) Store(string, *oauth2.Token) error    { return nil }
func (missingCache) Lookup(string) (*oauth2.Token, error) { return nil, cache.ErrNotFound }

// foundCache always returns a token. Used to confirm the wrapper passes
// successful lookups through unchanged.
type foundCache struct{ tok *oauth2.Token }

func (c foundCache) Store(string, *oauth2.Token) error    { return nil }
func (c foundCache) Lookup(string) (*oauth2.Token, error) { return c.tok, nil }

// boomCache returns a non-ErrNotFound error. The wrapper must not add a
// "run auth login" hint here; the error is about something else.
type boomCache struct{ err error }

func (c boomCache) Store(string, *oauth2.Token) error    { return nil }
func (c boomCache) Lookup(string) (*oauth2.Token, error) { return nil, c.err }

func writeLegacyCache(t *testing.T, path string, hasEntries bool) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	tokens := map[string]*oauth2.Token{}
	if hasEntries {
		tokens["my-profile"] = &oauth2.Token{AccessToken: "abc"}
	}
	body, err := json.Marshal(tokenCacheFile{Version: tokenCacheVersion, Tokens: tokens})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, body, 0o600))
}

func TestNotFoundHintCache_SecureWithLegacyEntries_UsesUpgradeMessage(t *testing.T) {
	tmp := t.TempDir()
	legacyPath := filepath.Join(tmp, tokenCacheFilePath)
	writeLegacyCache(t, legacyPath, true)

	c := &notFoundHintCache{inner: missingCache{}, mode: StorageModeSecure, legacyCachePath: legacyPath}
	_, err := c.Lookup("anything")
	require.Error(t, err)
	assert.ErrorIs(t, err, cache.ErrNotFound)
	assert.Contains(t, err.Error(), "stored credentials from older CLI versions")
	assert.Contains(t, err.Error(), "databricks auth login")
	assert.Contains(t, err.Error(), "DATABRICKS_AUTH_STORAGE=plaintext")
}

func TestNotFoundHintCache_SecureWithEmptyLegacyFile_UsesGenericMessage(t *testing.T) {
	tmp := t.TempDir()
	legacyPath := filepath.Join(tmp, tokenCacheFilePath)
	writeLegacyCache(t, legacyPath, false)

	c := &notFoundHintCache{inner: missingCache{}, mode: StorageModeSecure, legacyCachePath: legacyPath}
	_, err := c.Lookup("anything")
	require.Error(t, err)
	assert.ErrorIs(t, err, cache.ErrNotFound)
	assert.Contains(t, err.Error(), "no cached credentials")
	assert.NotContains(t, err.Error(), "stored credentials from older CLI versions")
}

func TestNotFoundHintCache_SecureNoLegacyFile_UsesGenericMessage(t *testing.T) {
	c := &notFoundHintCache{inner: missingCache{}, mode: StorageModeSecure, legacyCachePath: filepath.Join(t.TempDir(), "missing.json")}
	_, err := c.Lookup("anything")
	require.Error(t, err)
	assert.ErrorIs(t, err, cache.ErrNotFound)
	assert.Contains(t, err.Error(), "no cached credentials")
}

func TestNotFoundHintCache_Plaintext_AlwaysGenericMessage(t *testing.T) {
	tmp := t.TempDir()
	legacyPath := filepath.Join(tmp, tokenCacheFilePath)
	writeLegacyCache(t, legacyPath, true)

	// Even with a populated legacy file present, plaintext mode reads from
	// that same file, so the upgrade copy would be misleading.
	c := &notFoundHintCache{inner: missingCache{}, mode: StorageModePlaintext, legacyCachePath: legacyPath}
	_, err := c.Lookup("anything")
	require.Error(t, err)
	assert.ErrorIs(t, err, cache.ErrNotFound)
	assert.Contains(t, err.Error(), "no cached credentials")
	assert.NotContains(t, err.Error(), "stored credentials from older CLI versions")
}

func TestNotFoundHintCache_NonErrNotFound_PassesThrough(t *testing.T) {
	boom := errors.New("backend blew up")
	c := &notFoundHintCache{inner: boomCache{err: boom}, mode: StorageModeSecure, legacyCachePath: ""}
	_, err := c.Lookup("anything")
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.NotContains(t, err.Error(), "no cached credentials")
}

func TestNotFoundHintCache_SuccessfulLookupUnchanged(t *testing.T) {
	tok := &oauth2.Token{AccessToken: "abc"}
	c := &notFoundHintCache{inner: foundCache{tok: tok}, mode: StorageModeSecure, legacyCachePath: ""}
	got, err := c.Lookup("anything")
	require.NoError(t, err)
	assert.Equal(t, tok, got)
}

func TestNotFoundHintCache_StoreIsDelegated(t *testing.T) {
	c := &notFoundHintCache{inner: missingCache{}, mode: StorageModeSecure, legacyCachePath: ""}
	require.NoError(t, c.Store("k", &oauth2.Token{AccessToken: "abc"}))
}

func TestLegacyCacheHasTokens(t *testing.T) {
	tmp := t.TempDir()

	t.Run("empty path returns false", func(t *testing.T) {
		assert.False(t, legacyCacheHasTokens(""))
	})

	t.Run("missing file returns false", func(t *testing.T) {
		assert.False(t, legacyCacheHasTokens(filepath.Join(tmp, "missing.json")))
	})

	t.Run("garbage file returns false", func(t *testing.T) {
		p := filepath.Join(tmp, "garbage.json")
		require.NoError(t, os.WriteFile(p, []byte("not json"), 0o600))
		assert.False(t, legacyCacheHasTokens(p))
	})

	t.Run("empty token map returns false", func(t *testing.T) {
		p := filepath.Join(tmp, "empty.json")
		writeLegacyCache(t, p, false)
		assert.False(t, legacyCacheHasTokens(p))
	})

	t.Run("populated token map returns true", func(t *testing.T) {
		p := filepath.Join(tmp, "populated.json")
		writeLegacyCache(t, p, true)
		assert.True(t, legacyCacheHasTokens(p))
	})
}
