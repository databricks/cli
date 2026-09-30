package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
)

// notFoundHintStore wraps a Store so Lookup returns ErrNotFound with
// a hint pointing the user at `databricks auth login`. When mode is secure
// and the legacy file-backed cache has entries, the hint uses the upgrade-
// specific copy so users who logged in with an older CLI version know why
// their cached credentials are no longer being read.
//
// errors.Is(err, ErrNotFound) continues to return true because the
// wrap uses %w; PersistentAuth's branches on ErrNotFound still fire.
//
// Put and Delete are delegated unchanged; only Lookup needs the message polish.
type notFoundHintStore struct {
	inner           Store
	mode            StorageMode
	legacyStorePath string
}

func (s *notFoundHintStore) Put(key string, e Entry) error {
	return s.inner.Put(key, e)
}

func (s *notFoundHintStore) Lookup(key string) (Entry, error) {
	e, err := s.inner.Lookup(key)
	if err == nil || !errors.Is(err, ErrNotFound) {
		return e, err
	}
	if s.mode == StorageModeSecure && legacyStoreHasTokens(s.legacyStorePath) {
		return Entry{}, &notFoundHint{msg: "stored credentials from older CLI versions are no longer used; run `databricks auth login` to sign in again, or set DATABRICKS_AUTH_STORAGE=plaintext to keep using the file cache"}
	}
	return Entry{}, &notFoundHint{msg: "no cached credentials; run `databricks auth login` to sign in"}
}

func (s *notFoundHintStore) Delete(key string) error { return s.inner.Delete(key) }

// notFoundHint replaces ErrNotFound's terse "token not found" string
// with an actionable message while still satisfying errors.Is(err,
// ErrNotFound). PersistentAuth.loadToken wraps every store error with
// "cache: %w", and fmt.Errorf("...: %w", ErrNotFound) would tack the
// original "token not found" onto the end of our hint, producing
// "cache: <hint>: token not found". A custom type lets us own the
// rendered message while still unwrapping to ErrNotFound for callers
// that branch on it.
type notFoundHint struct {
	msg string
}

func (e *notFoundHint) Error() string { return e.msg }
func (e *notFoundHint) Unwrap() error { return ErrNotFound }

// HintForNotFound extracts the actionable hint message from an error
// chain produced by notFoundHintStore. Returns the empty string if the
// chain does not contain a notFoundHint (e.g. an unwrapped
// ErrNotFound from a plain Store).
//
// Used by call sites like `auth token` that rewrite the SDK error for
// backwards-compatibility (the "databricks OAuth is not configured for
// this host" substring is load-bearing for older SDK fall-through
// logic) but want to surface the actionable hint to the user instead of
// dropping it.
func HintForNotFound(err error) string {
	if hint, ok := errors.AsType[*notFoundHint](err); ok {
		return hint.msg
	}
	return ""
}

// NewNotFoundHint returns an error that renders as msg but unwraps to
// ErrNotFound, mirroring what notFoundHintStore produces in
// production. Exported so tests in other packages (e.g. cmd/auth) can
// construct a hint-wrapped error without going through the full
// resolver setup.
func NewNotFoundHint(msg string) error {
	return &notFoundHint{msg: msg}
}

// withNotFoundHint wraps inner so ErrNotFound from Lookup carries an
// actionable hint. The legacy file path is resolved up front (where ctx
// is available) so Lookup can do its check without needing a context.
//
// Resolution failures for the home directory are not fatal: an empty
// legacyStorePath simply disables the upgrade-specific message, which
// falls back to the generic "run auth login" hint.
func withNotFoundHint(ctx context.Context, inner Store, mode StorageMode) Store {
	var legacyStorePath string
	if home, err := env.UserHomeDir(ctx); err == nil {
		legacyStorePath = filepath.Join(home, tokenStoreFilePath)
	}
	return &notFoundHintStore{inner: inner, mode: mode, legacyStorePath: legacyStorePath}
}

// legacyStoreHasTokens reports whether the file at path is a valid token
// cache with at least one entry. Best-effort and read-only: any I/O or
// parse error returns false so we never claim "you have legacy tokens"
// when we cannot actually tell.
func legacyStoreHasTokens(path string) bool {
	if path == "" {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var f tokenStoreFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return false
	}
	return len(f.Tokens) > 0
}
