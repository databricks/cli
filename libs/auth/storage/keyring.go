package storage

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/google/uuid"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// keyringServiceName is the service name used for every entry the CLI writes
// to the OS-native secure store. The account field carries the per-entry
// cache key passed through LockedStore.Put and Store.Lookup.
const keyringServiceName = "databricks-cli"

// keyringProbeAccountPrefix is prefixed onto a per-call random suffix to form
// the account name ProbeKeyring writes and deletes. A fixed name like
// "__probe__" could collide with a user profile of the same name (which is
// what keyringStore uses as the account field), so the probe would clobber
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

// keyringStore stores OAuth tokens in the OS-native secure store.
// It implements the Store interface.
//
// The type is unexported so that the only way to construct a working instance
// is NewKeyringStore. A bare &keyringStore{} has a nil backend, which would
// panic on first use.
type keyringStore struct {
	backend        keyringBackend
	timeout        time.Duration
	keyringSvcName string

	// operationMu excludes new backend calls while a transaction waits for
	// timed-out calls to finish before releasing the cross-process sidecar.
	operationMu sync.RWMutex
	pendingMu   sync.Mutex
	pending     int
	pendingDone chan struct{}
}

// NewKeyringStore returns a Store backed by the OS-native secure store (via
// zalando/go-keyring) with a 3-second per-operation timeout.
func NewKeyringStore() Store {
	return &keyringStore{
		backend:        zalandoBackend{},
		timeout:        defaultKeyringTimeout,
		keyringSvcName: keyringServiceName,
	}
}

// ProbeKeyring returns nil if the OS keyring accepted a write+delete
// cycle within the standard timeout. *TimeoutError means the keyring
// was unresponsive (locked or hung, indistinguishable here); other
// errors are definitive failures.
//
// Used by the login path, where we want to validate both read and write
// capability before committing to the keyring backend.
func ProbeKeyring(ctx context.Context) error {
	return probeWithBackend(ctx, zalandoBackend{}, defaultKeyringTimeout)
}

func probeWithBackend(ctx context.Context, backend keyringBackend, timeout time.Duration) error {
	c := &keyringStore{
		backend:        backend,
		timeout:        timeout,
		keyringSvcName: keyringServiceName,
	}
	account := keyringProbeAccountPrefix + uuid.NewString()
	tok := &oauth2.Token{AccessToken: "probe"}
	return c.WithLock(ctx, func(locked LockedStore) error {
		if err := locked.Put(account, Entry{Token: tok}); err != nil {
			return fmt.Errorf("write: %w", err)
		}
		if err := locked.Delete(account); err != nil {
			return fmt.Errorf("delete: %w", err)
		}
		return nil
	})
}

// ProbeKeyringRead returns nil if the OS keyring accepted a Get for a
// non-existent account within the standard timeout (i.e. the backend is
// reachable and responded with keyring.ErrNotFound). *TimeoutError means
// the keyring was unresponsive; other errors are definitive failures.
//
// Used by the read path so probing does not write to the keyring. A
// successful probe is indistinguishable from the user not having an
// entry for this probe account; we treat both as "reachable".
func ProbeKeyringRead(ctx context.Context) error {
	return probeReadWithBackend(ctx, zalandoBackend{}, defaultKeyringTimeout)
}

func probeReadWithBackend(_ context.Context, backend keyringBackend, timeout time.Duration) error {
	c := &keyringStore{
		backend:        backend,
		timeout:        timeout,
		keyringSvcName: keyringServiceName,
	}
	account := keyringProbeAccountPrefix + uuid.NewString()
	_, err := c.readEntry(account)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// WithLock implements Store using the same stable sidecar as the plaintext
// store. Timed-out backend calls are allowed to finish before the sidecar is
// released so a late write cannot overlap the next transaction.
func (k *keyringStore) WithLock(ctx context.Context, fn func(LockedStore) error) error {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return fmt.Errorf("failed loading home directory: %w", err)
	}
	lockLocation := filepath.Join(home, tokenStoreLockFilePath)
	return withTokenStoreFileLock(ctx, lockLocation, func() error {
		k.operationMu.Lock()
		defer k.operationMu.Unlock()
		err := fn(&keyringLockedStore{store: k})
		k.waitForPending()
		return err
	})
}

type keyringLockedStore struct {
	store *keyringStore
}

// Lookup implements LockedStore, returning ErrNotFound on a miss.
func (s *keyringLockedStore) Lookup(key string) (Entry, error) {
	return s.store.readEntry(key)
}

// Put implements LockedStore.
func (s *keyringLockedStore) Put(key string, e Entry) error {
	raw, err := json.Marshal(keyringEntry(e))
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	return s.store.withTimeout("set", func() error {
		return s.store.backend.Set(s.store.keyringSvcName, key, string(raw))
	})
}

// Delete implements LockedStore. Removing a missing entry is a no-op.
func (s *keyringLockedStore) Delete(key string) error {
	return s.store.withTimeout("delete", func() error {
		err := s.store.backend.Delete(s.store.keyringSvcName, key)
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	})
}

// Lookup implements Store.
func (k *keyringStore) Lookup(key string) (Entry, error) {
	k.operationMu.RLock()
	defer k.operationMu.RUnlock()
	return k.readEntry(key)
}

func (k *keyringStore) readEntry(key string) (Entry, error) {
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
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, wrapKeyringUnreachable(err)
	}

	var entry keyringEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return Entry{}, fmt.Errorf("unmarshal token: %w", err)
	}
	return Entry(entry), nil
}

// wrapKeyringUnreachable wraps a non-ErrNotFound keyring error with
// actionable guidance for users whose system has no usable keyring
// backend (Linux without a Secret Service / D-Bus session bus, headless
// containers, certain SSH sessions). Surfaces on the read path, where
// the resolver does not silently fall back to plaintext: a missing
// token might actually be reachable from the keyring on another machine,
// so we surface the unreachability instead of minting a fresh plaintext
// copy.
//
// ErrNotFound passes through unchanged because a clean miss is not an
// availability problem.
func wrapKeyringUnreachable(err error) error {
	if err == nil || errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return fmt.Errorf("OS keyring unreachable: %w (set DATABRICKS_AUTH_STORAGE=plaintext or run `databricks auth login` to use file-based token storage)", err)
}

// Compile-time confirmation that keyringStore satisfies the CLI interface.
var _ Store = (*keyringStore)(nil)

// TimeoutError is returned when a keyring operation exceeds the configured
// timeout. Callers can use errors.As to detect and present a clear message.
type TimeoutError struct {
	Op string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("keyring %s timed out", cmp.Or(e.Op, "operation"))
}

// withTimeout runs op in a goroutine and returns a *TimeoutError when the
// backend exceeds k.timeout. Timed-out goroutines remain tracked until they
// finish; WithLock waits for them before releasing the cross-process sidecar.
func (k *keyringStore) withTimeout(op string, fn func() error) error {
	k.startPending()
	ch := make(chan error, 1)
	go func() {
		defer k.finishPending()
		ch <- fn()
	}()
	timer := time.NewTimer(k.timeout)
	defer timer.Stop()
	select {
	case err := <-ch:
		return err
	case <-timer.C:
		return &TimeoutError{Op: op}
	}
}

func (k *keyringStore) startPending() {
	k.pendingMu.Lock()
	defer k.pendingMu.Unlock()
	if k.pending == 0 {
		k.pendingDone = make(chan struct{})
	}
	k.pending++
}

func (k *keyringStore) finishPending() {
	k.pendingMu.Lock()
	defer k.pendingMu.Unlock()
	k.pending--
	if k.pending == 0 {
		close(k.pendingDone)
	}
}

func (k *keyringStore) waitForPending() {
	for {
		k.pendingMu.Lock()
		if k.pending == 0 {
			k.pendingMu.Unlock()
			return
		}
		done := k.pendingDone
		k.pendingMu.Unlock()
		<-done
	}
}
