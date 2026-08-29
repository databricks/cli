// Package storage selects and constructs the CLI's U2M token storage backend.
//
// Two modes are supported. Secure writes to the OS-native keyring under the
// profile cache key only; it is the resolver default. Plaintext writes to
// ~/.databricks/token-cache.json with host-key dual-write for older Go SDK
// versions (v0.61-v0.103); it is the opt-in fallback for environments where
// the OS keyring is not available.
package storage

import (
	"errors"

	"golang.org/x/oauth2"
)

// ErrNotFound is returned by Store.Lookup when no entry exists for the key, or
// when a stored entry cannot be decoded by this CLI version (an unknown format
// is treated as a miss so the caller re-mints rather than failing). It is the
// CLI-owned counterpart to the SDK's u2m_cache.ErrNotFound; the adapter in
// ToU2MTokenCache translates between the two.
var ErrNotFound = errors.New("token not found")

// Entry is the value held in the CLI token store. It wraps the credential so
// the schema can grow additive metadata (e.g. a config fingerprint, scopes)
// without changing the Store interface. Backends persist it verbatim and never
// interpret its contents.
type Entry struct {
	// Token is the cached OAuth token. Always set for stored entries.
	Token *oauth2.Token
}

// Store is the CLI's token-storage abstraction: a key/value store with no
// policy of its own. Implementations are the plaintext file cache and the OS
// keyring cache. The interface is owned by the CLI rather than the SDK so the
// entry schema can evolve (metadata, per-entry resilience) without being
// constrained by the SDK's U2M-internal u2m_cache.TokenCache, which only
// carries a bare *oauth2.Token.
type Store interface {
	// Put writes e under key, replacing any existing entry.
	Put(key string, e Entry) error

	// Lookup returns the entry stored under key, or ErrNotFound.
	Lookup(key string) (Entry, error)

	// Delete removes the entry under key. Deleting a missing entry is not an
	// error.
	Delete(key string) error
}
