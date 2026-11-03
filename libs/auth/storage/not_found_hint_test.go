package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// missingStore always returns ErrNotFound on Lookup. Lets us drive the
// wrapper without going through the real file or keyring cache.
type missingStore struct{}

func (missingStore) Put(string, Entry) error      { return nil }
func (missingStore) Lookup(string) (Entry, error) { return Entry{}, ErrNotFound }
func (missingStore) Delete(string) error          { return nil }

// foundStore always returns a token. Used to confirm the wrapper passes
// successful lookups through unchanged.
type foundStore struct{ tok *oauth2.Token }

func (s foundStore) Put(string, Entry) error      { return nil }
func (s foundStore) Lookup(string) (Entry, error) { return Entry{Token: s.tok}, nil }
func (s foundStore) Delete(string) error          { return nil }

// boomStore returns a non-ErrNotFound error. The wrapper must not add a
// "run auth login" hint here; the error is about something else.
type boomStore struct{ err error }

func (s boomStore) Put(string, Entry) error      { return nil }
func (s boomStore) Lookup(string) (Entry, error) { return Entry{}, s.err }
func (s boomStore) Delete(string) error          { return nil }

func writeLegacyStore(t *testing.T, path string, hasEntries bool) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	tokens := map[string]*fileEntry{}
	if hasEntries {
		tokens["my-profile"] = &fileEntry{Token: &oauth2.Token{AccessToken: "abc"}}
	}
	body, err := json.Marshal(tokenStoreFile{Version: tokenStoreVersion, Tokens: tokens})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, body, 0o600))
}

func TestNotFoundHintStore_SecureWithLegacyEntries_UsesUpgradeMessage(t *testing.T) {
	tmp := t.TempDir()
	legacyPath := filepath.Join(tmp, tokenStoreFilePath)
	writeLegacyStore(t, legacyPath, true)

	c := &notFoundHintStore{inner: missingStore{}, mode: StorageModeSecure, legacyStorePath: legacyPath}
	_, err := c.Lookup("anything")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Contains(t, err.Error(), "stored credentials from older CLI versions")
	assert.Contains(t, err.Error(), "databricks auth login")
	assert.Contains(t, err.Error(), "DATABRICKS_AUTH_STORAGE=plaintext")
}

func TestNotFoundHintStore_SecureWithEmptyLegacyFile_UsesGenericMessage(t *testing.T) {
	tmp := t.TempDir()
	legacyPath := filepath.Join(tmp, tokenStoreFilePath)
	writeLegacyStore(t, legacyPath, false)

	c := &notFoundHintStore{inner: missingStore{}, mode: StorageModeSecure, legacyStorePath: legacyPath}
	_, err := c.Lookup("anything")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Contains(t, err.Error(), "no cached credentials")
	assert.NotContains(t, err.Error(), "stored credentials from older CLI versions")
}

func TestNotFoundHintStore_SecureNoLegacyFile_UsesGenericMessage(t *testing.T) {
	c := &notFoundHintStore{inner: missingStore{}, mode: StorageModeSecure, legacyStorePath: filepath.Join(t.TempDir(), "missing.json")}
	_, err := c.Lookup("anything")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Contains(t, err.Error(), "no cached credentials")
}

func TestNotFoundHintStore_Plaintext_AlwaysGenericMessage(t *testing.T) {
	tmp := t.TempDir()
	legacyPath := filepath.Join(tmp, tokenStoreFilePath)
	writeLegacyStore(t, legacyPath, true)

	// Even with a populated legacy file present, plaintext mode reads from
	// that same file, so the upgrade copy would be misleading.
	c := &notFoundHintStore{inner: missingStore{}, mode: StorageModePlaintext, legacyStorePath: legacyPath}
	_, err := c.Lookup("anything")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Contains(t, err.Error(), "no cached credentials")
	assert.NotContains(t, err.Error(), "stored credentials from older CLI versions")
}

func TestNotFoundHintStore_NonErrNotFound_PassesThrough(t *testing.T) {
	boom := errors.New("backend blew up")
	c := &notFoundHintStore{inner: boomStore{err: boom}, mode: StorageModeSecure, legacyStorePath: ""}
	_, err := c.Lookup("anything")
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.NotContains(t, err.Error(), "no cached credentials")
}

func TestNotFoundHintStore_SuccessfulLookupUnchanged(t *testing.T) {
	tok := &oauth2.Token{AccessToken: "abc"}
	c := &notFoundHintStore{inner: foundStore{tok: tok}, mode: StorageModeSecure, legacyStorePath: ""}
	got, err := c.Lookup("anything")
	require.NoError(t, err)
	assert.Equal(t, tok, got.Token)
}

func TestNotFoundHintStore_PutIsDelegated(t *testing.T) {
	s := &notFoundHintStore{inner: missingStore{}, mode: StorageModeSecure, legacyStorePath: ""}
	require.NoError(t, s.Put("k", Entry{Token: &oauth2.Token{AccessToken: "abc"}}))
}

func TestHintForNotFound(t *testing.T) {
	t.Run("nil returns empty", func(t *testing.T) {
		assert.Empty(t, HintForNotFound(nil))
	})

	t.Run("plain ErrNotFound returns empty", func(t *testing.T) {
		// An unwrapped ErrNotFound carries no hint, so the caller
		// (e.g. `auth token`) falls back to its default error path.
		assert.Empty(t, HintForNotFound(ErrNotFound))
	})

	t.Run("unrelated error returns empty", func(t *testing.T) {
		assert.Empty(t, HintForNotFound(errors.New("something else")))
	})

	t.Run("notFoundHint returns the hint message", func(t *testing.T) {
		h := &notFoundHint{msg: "do the thing"}
		assert.Equal(t, "do the thing", HintForNotFound(h))
	})

	t.Run("notFoundHint behind an fmt.Errorf wrap returns the hint", func(t *testing.T) {
		// PersistentAuth wraps every store error with `cache: %w`, so the hint
		// is one Unwrap away when it surfaces in callers. errors.As must
		// still find it.
		h := &notFoundHint{msg: "do the thing"}
		wrapped := fmt.Errorf("cache: %w", h)
		assert.Equal(t, "do the thing", HintForNotFound(wrapped))
	})
}

func TestLegacyStoreHasTokens(t *testing.T) {
	tmp := t.TempDir()

	t.Run("empty path returns false", func(t *testing.T) {
		assert.False(t, legacyStoreHasTokens(""))
	})

	t.Run("missing file returns false", func(t *testing.T) {
		assert.False(t, legacyStoreHasTokens(filepath.Join(tmp, "missing.json")))
	})

	t.Run("garbage file returns false", func(t *testing.T) {
		p := filepath.Join(tmp, "garbage.json")
		require.NoError(t, os.WriteFile(p, []byte("not json"), 0o600))
		assert.False(t, legacyStoreHasTokens(p))
	})

	t.Run("empty token map returns false", func(t *testing.T) {
		p := filepath.Join(tmp, "empty.json")
		writeLegacyStore(t, p, false)
		assert.False(t, legacyStoreHasTokens(p))
	})

	t.Run("populated token map returns true", func(t *testing.T) {
		p := filepath.Join(tmp, "populated.json")
		writeLegacyStore(t, p, true)
		assert.True(t, legacyStoreHasTokens(p))
	})
}
