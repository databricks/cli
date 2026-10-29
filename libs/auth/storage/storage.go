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
// signal to callers that they need to mint or otherwise obtain a token.
var ErrNotFound = errors.New("token not found")

// Entry is the value held in the CLI token store. It wraps the credential so
// the schema can grow additive metadata (e.g. a config fingerprint, scopes)
// without changing the Store interface. Backends persist it verbatim and never
// interpret its contents.
type Entry struct {
	// Token is the cached OAuth token. Set for every stored entry except a
	// pending login.
	Token *oauth2.Token

	// PendingLogin is set instead of Token for a login that one CLI
	// invocation started and a later one completes.
	PendingLogin *PendingLogin
}

// PendingLogin is a device code login that is waiting for the user's
// confirmation code. Keeping the device code in the store means it never
// leaves the machine that started the login.
type PendingLogin struct {
	// DeviceCode is exchanged together with the confirmation code for a token.
	DeviceCode string `json:"device_code"`

	// UserCode is the code the user enters in the browser. It identifies the
	// login when a newer one replaced it under the same key.
	UserCode string `json:"user_code"`
}

// Store is the CLI's token-storage abstraction: a key/value store with no
// policy of its own. Implementations are the plaintext file cache and the OS
// keyring and in-memory stores. The entry schema can evolve with additive
// metadata without changing the Store interface.
type Store interface {
	// Put writes e under key, replacing any existing entry.
	Put(key string, e Entry) error

	// Lookup returns the entry stored under key, or ErrNotFound.
	Lookup(key string) (Entry, error)

	// Delete removes the entry under key. Deleting a missing entry is not an
	// error.
	Delete(key string) error
}
