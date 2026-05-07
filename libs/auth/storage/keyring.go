package storage

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/google/uuid"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// keyringServiceName is the service name used for every entry the CLI writes
// to the OS-native secure store. The account field carries the per-entry
// cache key the SDK passes through TokenCache.Store / Lookup.
const keyringServiceName = "databricks-cli"

// keyringProbeAccountPrefix is prefixed onto a per-call random suffix to form
// the account name ProbeKeyring writes and deletes. A fixed name like
// "__probe__" could collide with a user profile of the same name (which is
// what keyringCache uses as the account field), so the probe would clobber
// and delete that user's stored token. Per-call randomness also means
// concurrent probes don't step on each other.
const keyringProbeAccountPrefix = "__probe_"

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

// ProbeKeyring returns nil if a write+delete cycle completed within the
// standard timeout. Callers distinguish two non-nil shapes via errors.As:
//
//   - *TimeoutError: indeterminate. The keyring did not respond within
//     the timeout. The common cause on Linux is a locked collection
//     waiting on a GUI unlock prompt with the user mid-typing, but a
//     hung or slow daemon produces the same shape. The login path
//     optimistically stays on keyring: if the user is unlocking, the
//     prompt continues in parallel with OAuth and the final Store
//     succeeds against an unlocked keyring; if the keyring is genuinely
//     stuck, the final Store also times out and login fails late
//     instead of early. The cost of guessing wrong is one wasted OAuth
//     ceremony, not a silently-plaintext token.
//   - any other error: the keyring returned a definitive failure (no
//     daemon, headless session with no secret service, dismissed prompt,
//     ...). Login falls back to plaintext now rather than failing after
//     OAuth.
//
// Probing also has a useful side effect: triggering the unlock prompt up
// front, before the browser step. The user can answer it while OAuth is in
// flight instead of after.
func ProbeKeyring() error {
	return probeWithBackend(zalandoBackend{}, defaultKeyringTimeout)
}

func probeWithBackend(backend keyringBackend, timeout time.Duration) error {
	c := &keyringCache{
		backend:        backend,
		timeout:        timeout,
		keyringSvcName: keyringServiceName,
	}
	account := keyringProbeAccountPrefix + uuid.NewString()
	tok := &oauth2.Token{AccessToken: "probe"}
	if err := c.Store(account, tok); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := c.Store(account, nil); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
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
