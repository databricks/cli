package storage

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

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

func newTestCache(backend keyringBackend) *KeyringCache {
	return &KeyringCache{
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
