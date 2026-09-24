// Package storage selects and constructs the CLI's U2M token storage backend.
//
// Two modes are supported. Secure writes to the OS-native keyring under the
// profile cache key only; it is the resolver default. Plaintext writes to
// ~/.databricks/token-cache.json with host-key dual-write for older Go SDK
// versions (v0.61-v0.103); it is the opt-in fallback for environments where
// the OS keyring is not available.
package storage

import (
	"context"
	"errors"

	"golang.org/x/oauth2"
)

// ErrNotFound is returned by Store.Lookup when no entry exists for the key, or
// when a stored entry cannot be decoded by this CLI version (an unknown format
// is treated as a miss so the caller re-mints rather than failing). It is the
// signal to callers that they need to mint or otherwise obtain a token.
var ErrNotFound = errors.New("token not found")

// Entry is the value held in the CLI token store. It wraps the credential so
// the schema can grow additive metadata (e.g. a config fingerprint, scopes)
// without changing the Store interface. Backends persist it verbatim and never
// interpret its contents.
type Entry struct {
	// Token is the cached OAuth token. Always set for stored entries.
	Token *oauth2.Token
}

// Store provides read-only access to the token store and coordinates
// mutations through WithLock. Implementations must not expose mutable
// operations directly.
type Store interface {
	// Lookup returns the entry stored under key, or ErrNotFound.
	Lookup(key string) (Entry, error)

	// WithLock runs fn while holding the store's transaction lock. The
	// callback owns no lock-release operation; WithLock releases the lock
	// after fn returns, including when fn returns an error.
	WithLock(ctx context.Context, fn func(LockedStore) error) error
}

// LockedStore exposes token mutations only while a Store transaction is held.
type LockedStore interface {
	// Lookup returns the entry stored under key, or ErrNotFound.
	Lookup(key string) (Entry, error)

	// Put writes e under key, replacing any existing entry.
	Put(key string, e Entry) error

	// Delete removes the entry under key. Deleting a missing entry is not an
	// error.
	Delete(key string) error
}
