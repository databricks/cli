package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// keyringServiceName is the service name used for every entry the CLI writes
// to the OS-native secure store. The account field carries the per-entry
// cache key the SDK passes through TokenCache.Store / Lookup.
const keyringServiceName = "databricks-cli"

// defaultKeyringTimeout is how long a single keyring operation is allowed
// to run before the wrapper returns a TimeoutError. Matches the value used
// by GitHub CLI.
const defaultKeyringTimeout = 3 * time.Second

// keyringBackend is the subset of zalando/go-keyring the cache depends on.
// Extracted as an interface so tests can inject a fake.
type keyringBackend interface {
	Set(service, account, secret string) error
	Get(service, account string) (string, error)
	Delete(service, account string) error
}

// zalandoBackend delegates to the process-wide zalando/go-keyring provider.
type zalandoBackend struct{}

func (zalandoBackend) Set(service, account, secret string) error {
	return keyring.Set(service, account, secret)
}

func (zalandoBackend) Get(service, account string) (string, error) {
	return keyring.Get(service, account)
}

func (zalandoBackend) Delete(service, account string) error {
	return keyring.Delete(service, account)
}

// KeyringCache stores OAuth tokens in the OS-native secure store.
// It implements the SDK's cache.TokenCache interface.
type KeyringCache struct {
	backend        keyringBackend
	timeout        time.Duration
	errNotFound    error
	keyringSvcName string
}

// NewKeyringCache returns a KeyringCache backed by the process-wide
// zalando/go-keyring provider with a 3-second per-operation timeout.
func NewKeyringCache() *KeyringCache {
	return &KeyringCache{
		backend:        zalandoBackend{},
		timeout:        defaultKeyringTimeout,
		errNotFound:    keyring.ErrNotFound,
		keyringSvcName: keyringServiceName,
	}
}

// Store stores t under key. Nil t deletes the entry.
func (k *KeyringCache) Store(key string, t *oauth2.Token) error {
	if t == nil {
		// Implemented in Task 5.
		return errors.New("delete not implemented yet")
	}
	raw, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	return k.backend.Set(k.keyringSvcName, key, string(raw))
}

// Lookup returns the token under key or cache.ErrNotFound.
func (k *KeyringCache) Lookup(key string) (*oauth2.Token, error) {
	// Implemented in Task 4.
	return nil, errors.New("not implemented yet")
}

// Compile-time confirmation that KeyringCache satisfies the SDK interface.
var _ cache.TokenCache = (*KeyringCache)(nil)
