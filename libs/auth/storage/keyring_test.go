package storage

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
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
		return "", keyring.ErrNotFound
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

func newTestStore(backend keyringBackend) *keyringStore {
	return &keyringStore{
		backend:        backend,
		timeout:        100 * time.Millisecond,
		keyringSvcName: "databricks-cli",
	}
}

func TestKeyringStore_Store_WritesJSON(t *testing.T) {
	backend := newFakeBackend()
	c := newTestStore(backend)

	tok := &oauth2.Token{AccessToken: "abc", TokenType: "Bearer"}

	require.NoError(t, c.Put("my-profile", Entry{Token: tok}))

	stored, ok := backend.items[itemKey("databricks-cli", "my-profile")]
	require.True(t, ok, "token should be stored under service=databricks-cli, account=my-profile")

	var got keyringEntry
	require.NoError(t, json.Unmarshal([]byte(stored), &got))
	require.NotNil(t, got.Token)
	assert.Equal(t, "abc", got.Token.AccessToken)
	assert.Equal(t, "Bearer", got.Token.TokenType)
}

func TestKeyringStore_Store_PropagatesBackendError(t *testing.T) {
	boom := errors.New("backend boom")
	backend := newFakeBackend()
	backend.setErr = boom
	c := newTestStore(backend)

	err := c.Put("my-profile", Entry{Token: &oauth2.Token{AccessToken: "x"}})
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestKeyringStore_Lookup_ReturnsStoredToken(t *testing.T) {
	backend := newFakeBackend()
	c := newTestStore(backend)

	want := &oauth2.Token{AccessToken: "abc", TokenType: "Bearer"}
	require.NoError(t, c.Put("my-profile", Entry{Token: want}))

	got, err := c.Lookup("my-profile")
	require.NoError(t, err)
	assert.Equal(t, "abc", got.Token.AccessToken)
	assert.Equal(t, "Bearer", got.Token.TokenType)
}

func TestKeyringStore_Lookup_MissingReturnsCacheErrNotFound(t *testing.T) {
	backend := newFakeBackend()
	c := newTestStore(backend)

	_, err := c.Lookup("nope")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestKeyringStore_Lookup_PropagatesOtherErrors(t *testing.T) {
	boom := errors.New("backend boom")
	backend := newFakeBackend()
	backend.getErr = boom
	c := newTestStore(backend)

	_, err := c.Lookup("my-profile")
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	// The non-ErrNotFound path wraps the backend error with actionable
	// guidance so users on systems without a usable keyring backend
	// (e.g. headless Linux) know what to do.
	assert.Contains(t, err.Error(), "OS keyring unreachable")
	assert.Contains(t, err.Error(), "DATABRICKS_AUTH_STORAGE=plaintext")
	assert.Contains(t, err.Error(), "databricks auth login")
}

// ErrNotFound has to pass through unwrapped because callers branch on it
// (cache.ErrNotFound is the "no token, please log in" signal). Wrapping it
// with the unreachability hint would mislead the user.
func TestKeyringStore_Lookup_NotFoundIsNotWrapped(t *testing.T) {
	backend := newFakeBackend()
	c := newTestStore(backend)

	_, err := c.Lookup("nope")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.NotContains(t, err.Error(), "OS keyring unreachable")
}

func TestKeyringStore_Lookup_CorruptedJSONReturnsError(t *testing.T) {
	backend := newFakeBackend()
	backend.items[itemKey("databricks-cli", "my-profile")] = "{not json"
	c := newTestStore(backend)

	_, err := c.Lookup("my-profile")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal token")
}

func TestKeyringStore_StoreNil_DeletesEntry(t *testing.T) {
	backend := newFakeBackend()
	c := newTestStore(backend)

	require.NoError(t, c.Put("my-profile", Entry{Token: &oauth2.Token{AccessToken: "abc"}}))
	require.NoError(t, c.Delete("my-profile"))

	_, ok := backend.items[itemKey("databricks-cli", "my-profile")]
	assert.False(t, ok, "entry should be gone after delete")
}

func TestKeyringStore_StoreNil_MissingIsIdempotent(t *testing.T) {
	backend := newFakeBackend()
	backend.deleteErr = keyring.ErrNotFound
	c := newTestStore(backend)

	err := c.Delete("never-stored")
	require.NoError(t, err, "deleting a missing entry must not error")
}

func TestKeyringStore_StoreNil_PropagatesOtherDeleteErrors(t *testing.T) {
	boom := errors.New("backend boom")
	backend := newFakeBackend()
	backend.deleteErr = boom
	c := newTestStore(backend)

	err := c.Delete("my-profile")
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestKeyringStore_Store_TimesOut(t *testing.T) {
	backend := newFakeBackend()
	backend.setBlock = true
	c := newTestStore(backend) // 100ms timeout from newTestStore

	start := time.Now()
	err := c.Put("my-profile", Entry{Token: &oauth2.Token{AccessToken: "x"}})
	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr, "expected TimeoutError, got %T: %v", err, err)
	assert.Less(t, time.Since(start), 2*time.Second, "should time out quickly")
}

func TestKeyringStore_Lookup_TimesOut(t *testing.T) {
	backend := newFakeBackend()
	backend.getBlock = true
	c := newTestStore(backend)

	_, err := c.Lookup("my-profile")
	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr, "expected TimeoutError, got %T: %v", err, err)
}

func TestKeyringStore_StoreNil_TimesOut(t *testing.T) {
	backend := newFakeBackend()
	backend.deleteBlock = true
	c := newTestStore(backend)

	err := c.Delete("my-profile")
	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr, "expected TimeoutError, got %T: %v", err, err)
}

func TestProbeKeyring(t *testing.T) {
	boom := errors.New("backend boom")
	cases := []struct {
		name        string
		setErr      error
		deleteErr   error
		setBlock    bool
		timeout     time.Duration
		wantErr     error
		wantTimeout bool
	}{
		{
			name:    "success leaves no entry",
			timeout: 100 * time.Millisecond,
		},
		{
			name:    "set error propagates",
			setErr:  boom,
			timeout: 100 * time.Millisecond,
			wantErr: boom,
		},
		{
			name:        "set times out",
			setBlock:    true,
			timeout:     50 * time.Millisecond,
			wantTimeout: true,
		},
		{
			name:      "delete error propagates",
			deleteErr: boom,
			timeout:   100 * time.Millisecond,
			wantErr:   boom,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backend := newFakeBackend()
			backend.setErr = tc.setErr
			backend.deleteErr = tc.deleteErr
			backend.setBlock = tc.setBlock

			err := probeWithBackend(backend, tc.timeout)

			switch {
			case tc.wantErr != nil:
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			case tc.wantTimeout:
				require.Error(t, err)
				var timeoutErr *TimeoutError
				assert.ErrorAs(t, err, &timeoutErr)
			default:
				require.NoError(t, err)
				assert.Empty(t, backend.items, "probe must clean up after itself")
			}
		})
	}
}

func TestProbeKeyringRead(t *testing.T) {
	boom := errors.New("backend boom")
	cases := []struct {
		name        string
		getErr      error
		getBlock    bool
		timeout     time.Duration
		wantErr     error
		wantTimeout bool
	}{
		{
			// keyring.ErrNotFound is the success signal: the backend
			// responded that no entry exists for our probe account,
			// which means it is reachable.
			name:    "ErrNotFound counts as reachable",
			getErr:  keyring.ErrNotFound,
			timeout: 100 * time.Millisecond,
		},
		{
			name:    "other backend error propagates",
			getErr:  boom,
			timeout: 100 * time.Millisecond,
			wantErr: boom,
		},
		{
			name:        "get times out",
			getBlock:    true,
			timeout:     50 * time.Millisecond,
			wantTimeout: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backend := newFakeBackend()
			backend.getErr = tc.getErr
			backend.getBlock = tc.getBlock

			err := probeReadWithBackend(backend, tc.timeout)

			switch {
			case tc.wantErr != nil:
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			case tc.wantTimeout:
				require.Error(t, err)
				var timeoutErr *TimeoutError
				assert.ErrorAs(t, err, &timeoutErr)
			default:
				require.NoError(t, err)
				assert.Empty(t, backend.items, "read probe must not write to the keyring")
			}
		})
	}
}
