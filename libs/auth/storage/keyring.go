package storage

import (
	"cmp"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// keyringServiceName is the service name used for every entry the CLI writes
// to the OS-native secure store. The account field carries the per-entry
// cache key the U2M manager passes through Store.Put / Lookup.
const keyringServiceName = "databricks-cli"

// AccessTokenPartMarker separates a cache key from the index of one of its
// access token parts, the items that hold the access token of an entry too
// large for one item. A cache key must not contain it, or the entry would
// share its account with a part of another entry.
const AccessTokenPartMarker = "#access-token-"

// accessTokenPartSize is how many bytes of an access token go into one part.
// It stays under the smallest backend limit. Windows Credential Manager caps
// a credential at 2560 bytes. go-keyring's macOS backend base64-encodes the
// secret into one command line for `security -i`, which reads at most 4096
// bytes per line, so about 3000 bytes fit.
// https://learn.microsoft.com/en-us/windows/win32/api/wincred/ns-wincred-credentialw
// https://github.com/zalando/go-keyring/blob/v0.2.8/keyring_darwin.go#L86-L89
const accessTokenPartSize = 2048

// ErrAccessTokenNotStored is wrapped in the error of a Put that stored the
// entry of a split token, with its refresh token, but not all of its access
// token parts. Lookup then returns the token without an access token, so the
// next command refreshes it.
var ErrAccessTokenNotStored = errors.New("access token not stored")

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

// splitLookupAttempts and splitLookupDelay make Lookup read an incomplete
// split token again after 25, 50, 100 and 200 milliseconds, like the store
// update recovery in u2m.
const (
	splitLookupAttempts = 5
	splitLookupDelay    = 25 * time.Millisecond
)

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
	entryFields

	// Split is set for a token that is too large for one keyring item. CLI
	// versions that predate it still read the entry, so Token is then empty
	// rather than nil, because releases before v1.24 dereference it. It must
	// not carry the refresh token either: they would rotate it and then fail
	// to store the oversized result.
	Split *splitToken `json:"split,omitempty"`
}

// entryFields holds the fields of Entry. Entry converts to and from it, so a
// field added to Entry but not here fails to compile instead of being dropped
// by the keyring store.
type entryFields struct {
	Token        *oauth2.Token `json:"token"`
	PendingLogin *PendingLogin `json:"pending_login,omitempty"`
}

// splitToken is the part of an oversized token that stays in its entry.
type splitToken struct {
	// Token is the token without its access token.
	Token oauth2.Token `json:"token"`

	// AccessTokenParts is the number of parts that hold the access token and
	// AccessTokenSHA256 is its hex digest. The parts are written one at a
	// time, so only the digest tells a complete access token from one that
	// another process is still writing or has since replaced.
	AccessTokenParts  int    `json:"access_token_parts"`
	AccessTokenSHA256 string `json:"access_token_sha256"`
}

// zalandoBackend delegates to the process-wide zalando/go-keyring provider.
type zalandoBackend struct{}

func (zalandoBackend) Set(service, account, secret string) error {
	if runtime.GOOS == "darwin" && darwinCommandTooBig(service, account, secret) {
		return keyring.ErrSetDataTooBig
	}
	return keyring.Set(service, account, secret)
}

// darwinCommandTooBig reports whether go-keyring's macOS backend will reject
// secret as too big. It does so only after starting `security -i`, which it
// then never reaps, so Set checks first. The count leaves out shell quotes,
// which only make the command longer, so no secret that go-keyring would store
// is rejected.
// https://github.com/zalando/go-keyring/blob/v0.2.8/keyring_darwin.go#L82-L88
func darwinCommandTooBig(service, account, secret string) bool {
	password := "go-keyring-base64:" + base64.StdEncoding.EncodeToString([]byte(secret))
	command := fmt.Sprintf("add-generic-password -U -s %s -a %s -w %s\n", service, account, password)
	return len(command) > 4096
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

	// retryDelay is Lookup's first wait before it reads an incomplete split
	// token again, splitLookupDelay if zero. It doubles on each attempt.
	retryDelay time.Duration
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
func ProbeKeyring() error {
	return probeWithBackend(zalandoBackend{}, defaultKeyringTimeout)
}

func probeWithBackend(backend keyringBackend, timeout time.Duration) error {
	c := &keyringStore{
		backend:        backend,
		timeout:        timeout,
		keyringSvcName: keyringServiceName,
	}
	account := keyringProbeAccountPrefix + uuid.New().String()
	tok := &oauth2.Token{AccessToken: "probe"}
	if err := c.Put(account, Entry{Token: tok}); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := c.Delete(account); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// ProbeKeyringRead returns nil if the OS keyring accepted a Get for a
// non-existent account within the standard timeout (i.e. the backend is
// reachable and responded with keyring.ErrNotFound). *TimeoutError means
// the keyring was unresponsive; other errors are definitive failures.
//
// Used by the read path so probing does not write to the keyring. A
// successful probe is indistinguishable from the user not having an
// entry for that probe account; we treat both as "reachable".
func ProbeKeyringRead() error {
	return probeReadWithBackend(zalandoBackend{}, defaultKeyringTimeout)
}

func probeReadWithBackend(backend keyringBackend, timeout time.Duration) error {
	c := &keyringStore{
		backend:        backend,
		timeout:        timeout,
		keyringSvcName: keyringServiceName,
	}
	account := keyringProbeAccountPrefix + uuid.New().String()
	err := c.withTimeout("get", func() error {
		_, gerr := c.backend.Get(c.keyringSvcName, account)
		return gerr
	})
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// Put implements the Store interface.
func (k *keyringStore) Put(key string, e Entry) error {
	raw, err := json.Marshal(keyringEntry{entryFields: entryFields(e)})
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	err = k.set(key, string(raw))
	if e.Token != nil && errors.Is(err, keyring.ErrSetDataTooBig) {
		return k.putSplit(key, entryFields(e))
	}
	return err
}

// putSplit stores a token that is too large for one keyring item. The entry
// under key keeps everything but the access token, which goes into parts next
// to it.
//
// The entry is written first because it holds the refresh token. The server
// rotates the refresh token on every refresh, so losing the new one logs the
// user out, while losing the access token only costs another refresh.
func (k *keyringStore) putSplit(key string, fields entryFields) error {
	t := fields.Token
	sum := sha256.Sum256([]byte(t.AccessToken))
	n := (len(t.AccessToken) + accessTokenPartSize - 1) / accessTokenPartSize
	rest := *t
	rest.AccessToken = ""
	fields.Token = &oauth2.Token{}
	raw, err := json.Marshal(keyringEntry{
		entryFields: fields,
		Split: &splitToken{
			Token:             rest,
			AccessTokenParts:  n,
			AccessTokenSHA256: hex.EncodeToString(sum[:]),
		},
	})
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := k.set(key, string(raw)); err != nil {
		return err
	}
	for i := range n {
		start := i * accessTokenPartSize
		part := t.AccessToken[start:min(start+accessTokenPartSize, len(t.AccessToken))]
		if err := k.set(accessTokenPartKey(key, i), part); err != nil {
			return fmt.Errorf("%w: %w", ErrAccessTokenNotStored, err)
		}
	}
	return nil
}

// Delete implements the Store interface. Removing a missing entry is a no-op.
func (k *keyringStore) Delete(key string) error {
	err := k.remove(key)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	// A token that fits in one item can replace a split one without touching
	// its access token parts, so the entry does not say how many there are.
	// They are counted and then deleted from the last, so that a Delete that
	// fails part way leaves parts that the next one still finds.
	n := 0
	for ; ; n++ {
		_, err := k.get(accessTokenPartKey(key, n))
		if errors.Is(err, keyring.ErrNotFound) {
			break
		}
		if err != nil {
			return err
		}
	}
	for i := n - 1; i >= 0; i-- {
		err := k.remove(accessTokenPartKey(key, i))
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return err
		}
	}
	return nil
}

// Lookup implements the Store interface, returning ErrNotFound on a miss.
//
// A split token with missing or mismatched parts is read again a few times
// before it is returned without its access token, because another process
// may be writing or deleting it: refreshing at once could spend a refresh
// token that process replaced, or restore a login it deleted.
func (k *keyringStore) Lookup(key string) (Entry, error) {
	delay := cmp.Or(k.retryDelay, splitLookupDelay)
	for attempt := 1; ; attempt++ {
		e, complete, err := k.lookup(key)
		if err != nil || complete || attempt == splitLookupAttempts {
			return e, err
		}
		time.Sleep(delay)
		delay *= 2
	}
}

// lookup reads the entry under key once. It reports whether the entry is
// complete, which only a split token with missing or mismatched parts is not.
func (k *keyringStore) lookup(key string) (Entry, bool, error) {
	raw, err := k.get(key)
	if errors.Is(err, keyring.ErrNotFound) {
		return Entry{}, false, ErrNotFound
	}
	if err != nil {
		return Entry{}, false, wrapKeyringUnreachable(err)
	}

	var entry keyringEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return Entry{}, false, fmt.Errorf("unmarshal token: %w", err)
	}
	if entry.Split != nil {
		return k.lookupSplit(key, entry)
	}
	return Entry(entry.entryFields), true, nil
}

// lookupSplit assembles a token stored by putSplit. When the access token
// parts are missing or do not match the entry, because a write was interrupted
// or another process is writing, it returns the token without an access token
// and reports it as incomplete.
func (k *keyringStore) lookupSplit(key string, entry keyringEntry) (Entry, bool, error) {
	s := entry.Split
	t := s.Token
	e := Entry(entry.entryFields)
	e.Token = &t
	var accessToken strings.Builder
	for i := range s.AccessTokenParts {
		part, err := k.get(accessTokenPartKey(key, i))
		if errors.Is(err, keyring.ErrNotFound) {
			return e, false, nil
		}
		if err != nil {
			return Entry{}, false, wrapKeyringUnreachable(err)
		}
		accessToken.WriteString(part)
	}
	sum := sha256.Sum256([]byte(accessToken.String()))
	if hex.EncodeToString(sum[:]) != s.AccessTokenSHA256 {
		return e, false, nil
	}
	t.AccessToken = accessToken.String()
	return e, true, nil
}

// accessTokenPartKey returns the key of the i-th access token part of the
// entry stored under key.
func accessTokenPartKey(key string, i int) string {
	return key + AccessTokenPartMarker + strconv.Itoa(i)
}

// set writes secret under key within the store's timeout.
func (k *keyringStore) set(key, secret string) error {
	return k.withTimeout("set", func() error {
		return k.backend.Set(k.keyringSvcName, key, secret)
	})
}

// get reads the secret under key within the store's timeout.
func (k *keyringStore) get(key string) (string, error) {
	var secret string
	err := k.withTimeout("get", func() error {
		got, gerr := k.backend.Get(k.keyringSvcName, key)
		if gerr != nil {
			return gerr
		}
		secret = got
		return nil
	})
	// After a timeout the goroutine may still be writing secret.
	if err != nil {
		return "", err
	}
	return secret, nil
}

// remove deletes the item under key within the store's timeout.
func (k *keyringStore) remove(key string) error {
	return k.withTimeout("delete", func() error {
		return k.backend.Delete(k.keyringSvcName, key)
	})
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

// withTimeout runs op in a goroutine and returns its error, or a
// *TimeoutError if op does not complete before k.timeout elapses. The
// goroutine is not cancelled; it will complete (or outlive the process)
// in the background. This mirrors the pattern used by GitHub CLI; see
// https://github.com/cli/cli/blob/trunk/internal/keyring/keyring.go.
func (k *keyringStore) withTimeout(op string, fn func() error) error {
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
