package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"golang.org/x/oauth2"
)

// notFoundHintCache wraps a TokenCache so Lookup returns ErrNotFound with
// a hint pointing the user at `databricks auth login`. When mode is secure
// and the legacy file-backed cache has entries, the hint uses the upgrade-
// specific copy so users who logged in with an older CLI version know why
// their cached credentials are no longer being read.
//
// errors.Is(err, cache.ErrNotFound) continues to return true because the
// wrap uses %w; the SDK's branches on ErrNotFound still fire.
//
// Store is delegated unchanged; only Lookup needs the message polish.
type notFoundHintCache struct {
	inner           cache.TokenCache
	mode            StorageMode
	legacyCachePath string
}

func (c *notFoundHintCache) Store(key string, t *oauth2.Token) error {
	return c.inner.Store(key, t)
}

func (c *notFoundHintCache) Lookup(key string) (*oauth2.Token, error) {
	tok, err := c.inner.Lookup(key)
	if err == nil || !errors.Is(err, cache.ErrNotFound) {
		return tok, err
	}
	if c.mode == StorageModeSecure && legacyCacheHasTokens(c.legacyCachePath) {
		return nil, &notFoundHint{msg: "stored credentials from older CLI versions are no longer used; run `databricks auth login` to sign in again, or set DATABRICKS_AUTH_STORAGE=plaintext to keep using the file cache"}
	}
	return nil, &notFoundHint{msg: "no cached credentials; run `databricks auth login` to sign in"}
}

// notFoundHint replaces cache.ErrNotFound's terse "token not found" string
// with an actionable message while still satisfying errors.Is(err,
// cache.ErrNotFound). The SDK's loadToken wraps every cache error with
// "cache: %w", and fmt.Errorf("...: %w", ErrNotFound) would tack the
// original "token not found" onto the end of our hint, producing
// "cache: <hint>: token not found". A custom type lets us own the
// rendered message while still unwrapping to ErrNotFound for callers
// that branch on it.
type notFoundHint struct {
	msg string
}

func (e *notFoundHint) Error() string { return e.msg }
func (e *notFoundHint) Unwrap() error { return cache.ErrNotFound }

// withNotFoundHint wraps inner so ErrNotFound from Lookup carries an
// actionable hint. The legacy file path is resolved up front (where ctx
// is available) so Lookup can do its check without needing a context.
//
// Resolution failures for the home directory are not fatal: an empty
// legacyCachePath simply disables the upgrade-specific message, which
// falls back to the generic "run auth login" hint.
func withNotFoundHint(ctx context.Context, inner cache.TokenCache, mode StorageMode) cache.TokenCache {
	var legacyCachePath string
	if home, err := env.UserHomeDir(ctx); err == nil {
		legacyCachePath = filepath.Join(home, tokenCacheFilePath)
	}
	return &notFoundHintCache{inner: inner, mode: mode, legacyCachePath: legacyCachePath}
}

// legacyCacheHasTokens reports whether the file at path is a valid token
// cache with at least one entry. Best-effort and read-only: any I/O or
// parse error returns false so we never claim "you have legacy tokens"
// when we cannot actually tell.
func legacyCacheHasTokens(path string) bool {
	if path == "" {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var f tokenCacheFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return false
	}
	return len(f.Tokens) > 0
}
