package storage

import (
	"cmp"
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
//
// This is needed because keyring backends can block indefinitely with no
// client-side cancel. For example, on Linux the Secret Service waits for
// a GUI unlock prompt that no one answers in a headless session.
const defaultKeyringTimeout = 3 * time.Second

// keyringBackend is the subset of zalando/go-keyring the cache depends on.
// Extracted as an interface so tests can inject a fake.
type keyringBackend interface {
	Set(service, account, secret string) error
	Get(service, account string) (string, error)
	Delete(service, account string) error
}

// keyringEntry is the on-disk envelope stored under each keyring account.
// Wrapping the token in a struct lets us add fields later (scopes, profile
// checksum, store time, ...) without breaking older CLI versions that read
// the same entry.
type keyringEntry struct {
	Token *oauth2.Token `json:"token"`
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

// keyringCache stores OAuth tokens in the OS-native secure store.
// It implements the SDK's cache.TokenCache interface.
//
// The type is unexported so that the only way to construct a working instance
// is NewKeyringCache. A bare &keyringCache{} has a nil backend, which would
// panic on first use.
type keyringCache struct {
	backend        keyringBackend
	timeout        time.Duration
	keyringSvcName string
}

// NewKeyringCache returns a cache.TokenCache backed by the OS-native secure
// store (via zalando/go-keyring) with a 3-second per-operation timeout.
func NewKeyringCache() cache.TokenCache {
	return &keyringCache{
		backend:        zalandoBackend{},
		timeout:        defaultKeyringTimeout,
		keyringSvcName: keyringServiceName,
	}
}

// Store stores t under key. Nil t deletes the entry; deleting a missing
// entry is not an error.
func (k *keyringCache) Store(key string, t *oauth2.Token) error {
	if t == nil {
		return k.withTimeout("delete", func() error {
			err := k.backend.Delete(k.keyringSvcName, key)
			if errors.Is(err, keyring.ErrNotFound) {
				return nil
			}
			return err
		})
	}
	raw, err := json.Marshal(keyringEntry{Token: t})
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	return k.withTimeout("set", func() error {
		return k.backend.Set(k.keyringSvcName, key, string(raw))
	})
}

// Lookup returns the token under key or cache.ErrNotFound.
func (k *keyringCache) Lookup(key string) (*oauth2.Token, error) {
	var raw string
	err := k.withTimeout("get", func() error {
		got, gerr := k.backend.Get(k.keyringSvcName, key)
		if gerr != nil {
			return gerr
		}
		raw = got
		return nil
	})
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, cache.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var entry keyringEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}
	return entry.Token, nil
}

// Compile-time confirmation that keyringCache satisfies the SDK interface.
var _ cache.TokenCache = (*keyringCache)(nil)

// TimeoutError is returned when a keyring operation exceeds the configured
// timeout. Callers can use errors.As to detect and present a clear message.
type TimeoutError struct {
	Op string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("keyring %s timed out", cmp.Or(e.Op, "operation"))
}

// withTimeout runs op in a goroutine and returns its error, or a
// *TimeoutError if op does not complete before k.timeout elapses. The
// goroutine is not cancelled; it will complete (or outlive the process)
// in the background. This mirrors the pattern used by GitHub CLI; see
// https://github.com/cli/cli/blob/trunk/internal/keyring/keyring.go.
func (k *keyringCache) withTimeout(op string, fn func() error) error {
	ch := make(chan error, 1)
	go func() {
		ch <- fn()
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(k.timeout):
		return &TimeoutError{Op: op}
	}
}
