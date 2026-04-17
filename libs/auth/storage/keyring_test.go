package storage

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// fakeBackend is a test double for keyringBackend. It records Set/Get/Delete
// calls and lets tests program responses.
type fakeBackend struct {
	items map[string]string // key = service+":"+account

	setErr    error
	getErr    error
	deleteErr error

	setBlock    bool // if true, Set blocks forever (for timeout tests)
	getBlock    bool
	deleteBlock bool
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{items: map[string]string{}}
}

func itemKey(service, account string) string { return service + ":" + account }

func (f *fakeBackend) Set(service, account, secret string) error {
	if f.setBlock {
		select {}
	}
	if f.setErr != nil {
		return f.setErr
	}
	f.items[itemKey(service, account)] = secret
	return nil
}

func (f *fakeBackend) Get(service, account string) (string, error) {
	if f.getBlock {
		select {}
	}
	if f.getErr != nil {
		return "", f.getErr
	}
	v, ok := f.items[itemKey(service, account)]
	if !ok {
		return "", errNotFoundSentinel
	}
	return v, nil
}

func (f *fakeBackend) Delete(service, account string) error {
	if f.deleteBlock {
		select {}
	}
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.items, itemKey(service, account))
	return nil
}

// errNotFoundSentinel lets tests assert what the cache does with a missing
// entry. Production code uses the real zalando/go-keyring ErrNotFound; the
// fake uses its own sentinel that the test wires into the cache.
var errNotFoundSentinel = errors.New("fake keyring: not found")

func newTestCache(backend keyringBackend) *keyringCache {
	return &keyringCache{
		backend:        backend,
		timeout:        100 * time.Millisecond,
		errNotFound:    errNotFoundSentinel,
		keyringSvcName: "databricks-cli",
	}
}

func TestKeyringCache_Store_WritesJSON(t *testing.T) {
	backend := newFakeBackend()
	c := newTestCache(backend)

	tok := &oauth2.Token{AccessToken: "abc", TokenType: "Bearer"}

	require.NoError(t, c.Store("my-profile", tok))

	stored, ok := backend.items[itemKey("databricks-cli", "my-profile")]
	require.True(t, ok, "token should be stored under service=databricks-cli, account=my-profile")

	var got oauth2.Token
	require.NoError(t, json.Unmarshal([]byte(stored), &got))
	assert.Equal(t, "abc", got.AccessToken)
	assert.Equal(t, "Bearer", got.TokenType)
}

func TestKeyringCache_Store_PropagatesBackendError(t *testing.T) {
	boom := errors.New("backend boom")
	backend := newFakeBackend()
	backend.setErr = boom
	c := newTestCache(backend)

	err := c.Store("my-profile", &oauth2.Token{AccessToken: "x"})
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestKeyringCache_Lookup_ReturnsStoredToken(t *testing.T) {
	backend := newFakeBackend()
	c := newTestCache(backend)

	want := &oauth2.Token{AccessToken: "abc", TokenType: "Bearer"}
	require.NoError(t, c.Store("my-profile", want))

	got, err := c.Lookup("my-profile")
	require.NoError(t, err)
	assert.Equal(t, "abc", got.AccessToken)
	assert.Equal(t, "Bearer", got.TokenType)
}

func TestKeyringCache_Lookup_MissingReturnsCacheErrNotFound(t *testing.T) {
	backend := newFakeBackend()
	c := newTestCache(backend)

	_, err := c.Lookup("nope")
	require.Error(t, err)
	assert.ErrorIs(t, err, cache.ErrNotFound)
}

func TestKeyringCache_Lookup_PropagatesOtherErrors(t *testing.T) {
	boom := errors.New("backend boom")
	backend := newFakeBackend()
	backend.getErr = boom
	c := newTestCache(backend)

	_, err := c.Lookup("my-profile")
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestKeyringCache_Lookup_CorruptedJSONReturnsError(t *testing.T) {
	backend := newFakeBackend()
	backend.items[itemKey("databricks-cli", "my-profile")] = "{not json"
	c := newTestCache(backend)

	_, err := c.Lookup("my-profile")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal token")
}

func TestKeyringCache_StoreNil_DeletesEntry(t *testing.T) {
	backend := newFakeBackend()
	c := newTestCache(backend)

	require.NoError(t, c.Store("my-profile", &oauth2.Token{AccessToken: "abc"}))
	require.NoError(t, c.Store("my-profile", nil))

	_, ok := backend.items[itemKey("databricks-cli", "my-profile")]
	assert.False(t, ok, "entry should be gone after delete")
}

func TestKeyringCache_StoreNil_MissingIsIdempotent(t *testing.T) {
	backend := newFakeBackend()
	backend.deleteErr = errNotFoundSentinel
	c := newTestCache(backend)

	err := c.Store("never-stored", nil)
	require.NoError(t, err, "deleting a missing entry must not error")
}

func TestKeyringCache_StoreNil_PropagatesOtherDeleteErrors(t *testing.T) {
	boom := errors.New("backend boom")
	backend := newFakeBackend()
	backend.deleteErr = boom
	c := newTestCache(backend)

	err := c.Store("my-profile", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestKeyringCache_Store_TimesOut(t *testing.T) {
	backend := newFakeBackend()
	backend.setBlock = true
	c := newTestCache(backend) // 100ms timeout from newTestCache

	start := time.Now()
	err := c.Store("my-profile", &oauth2.Token{AccessToken: "x"})
	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr, "expected TimeoutError, got %T: %v", err, err)
	assert.Less(t, time.Since(start), 2*time.Second, "should time out quickly")
}

func TestKeyringCache_Lookup_TimesOut(t *testing.T) {
	backend := newFakeBackend()
	backend.getBlock = true
	c := newTestCache(backend)

	_, err := c.Lookup("my-profile")
	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr, "expected TimeoutError, got %T: %v", err, err)
}

func TestKeyringCache_StoreNil_TimesOut(t *testing.T) {
	backend := newFakeBackend()
	backend.deleteBlock = true
	c := newTestCache(backend)

	err := c.Store("my-profile", nil)
	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr, "expected TimeoutError, got %T: %v", err, err)
}
