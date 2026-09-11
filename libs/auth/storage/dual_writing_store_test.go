package storage

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// testStore is a minimal in-memory Store used only by wrapper tests.
type testStore struct {
	mu      sync.Mutex
	entries map[string]Entry
}

func newDualWritingTestStore() *testStore {
	return &testStore{entries: map[string]Entry{}}
}

func (s *testStore) Put(key string, e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = e
	return nil
}

func (s *testStore) Lookup(key string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[key]
	if !ok {
		return Entry{}, ErrNotFound
	}
	return e, nil
}

func (s *testStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
	return nil
}

// plainArg implements OAuthArgument only, exercising the "no host key" branch.
type plainArg struct {
	key string
}

func (a plainArg) GetCacheKey() string { return a.key }

// hostArg implements HostCacheKeyProvider so the wrapper mirrors the token
// to the configured host key.
type hostArg struct {
	key     string
	hostKey string
}

func (a hostArg) GetCacheKey() string     { return a.key }
func (a hostArg) GetHostCacheKey() string { return a.hostKey }

type discoveryArg struct {
	key  string
	host string
}

func (a discoveryArg) GetCacheKey() string       { return a.key }
func (a discoveryArg) GetDiscoveredHost() string { return a.host }

func TestDualWritingStorePutPrimaryMirrorsHost(t *testing.T) {
	inner := newDualWritingTestStore()
	arg := hostArg{key: "profile-a", hostKey: "https://example.databricks.com"}
	c := NewDualWritingStore(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc", RefreshToken: "r"}

	require.NoError(t, c.Put("profile-a", Entry{Token: tok}))

	primary, err := inner.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, primary.Token)

	host, err := inner.Lookup("https://example.databricks.com")
	require.NoError(t, err)
	assert.Equal(t, tok, host.Token)
}

func TestDualWritingStoreDeletePrimaryMirrorsHost(t *testing.T) {
	inner := newDualWritingTestStore()
	arg := hostArg{key: "profile-a", hostKey: "https://example.databricks.com"}
	s := NewDualWritingStore(inner, arg)
	require.NoError(t, s.Put("profile-a", Entry{Token: &oauth2.Token{AccessToken: "abc"}}))

	require.NoError(t, s.Delete("profile-a"))

	_, err := inner.Lookup("profile-a")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = inner.Lookup("https://example.databricks.com")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDualWritingStorePutNonPrimaryDoesNotMirror(t *testing.T) {
	// An explicit host-key write must pass through without re-expanding into
	// another pair.
	inner := newDualWritingTestStore()
	arg := hostArg{key: "profile-a", hostKey: "https://example.databricks.com"}
	c := NewDualWritingStore(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Put("https://example.databricks.com", Entry{Token: tok}))

	host, err := inner.Lookup("https://example.databricks.com")
	require.NoError(t, err)
	assert.Equal(t, tok, host.Token)
	_, err = inner.Lookup("profile-a")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDualWritingStorePutNoHostKey(t *testing.T) {
	inner := newDualWritingTestStore()
	arg := plainArg{key: "profile-a"}
	c := NewDualWritingStore(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Put("profile-a", Entry{Token: tok}))

	got, err := inner.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, got.Token)
	assert.Len(t, inner.entries, 1)
}

func TestDualWritingStorePutHostKeyEqualsPrimary(t *testing.T) {
	inner := newDualWritingTestStore()
	arg := hostArg{key: "https://example.databricks.com", hostKey: "https://example.databricks.com"}
	c := NewDualWritingStore(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Put("https://example.databricks.com", Entry{Token: tok}))

	assert.Len(t, inner.entries, 1)
}

func TestDualWritingStoreDiscoveryArgWithDiscoveredHost(t *testing.T) {
	inner := newDualWritingTestStore()
	arg := discoveryArg{key: "profile-a", host: "https://example.databricks.com"}
	c := NewDualWritingStore(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Put("profile-a", Entry{Token: tok}))

	primary, err := inner.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, primary.Token)

	host, err := inner.Lookup("https://example.databricks.com")
	require.NoError(t, err)
	assert.Equal(t, tok, host.Token)
}

func TestDualWritingStoreDiscoveryArgWithEmptyDiscoveredHost(t *testing.T) {
	inner := newDualWritingTestStore()
	arg := discoveryArg{key: "profile-a"}
	c := NewDualWritingStore(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Put("profile-a", Entry{Token: tok}))

	assert.Len(t, inner.entries, 1)
	primary, err := inner.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, primary.Token)
}

func TestDualWritingStoreLookupDelegates(t *testing.T) {
	inner := newDualWritingTestStore()
	arg := hostArg{key: "profile-a", hostKey: "https://example.databricks.com"}
	c := NewDualWritingStore(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}
	require.NoError(t, inner.Put("profile-a", Entry{Token: tok}))

	got, err := c.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, got.Token)

	_, err = c.Lookup("missing")
	require.ErrorIs(t, err, ErrNotFound)
}

// failOnHostKeyCache returns an error when asked to write under hostKey;
// primary writes succeed. Used to verify the wrapper treats host-key
// mirrors as best-effort.
type failOnHostKeyStore struct {
	testStore
	hostKey string
}

func (s *failOnHostKeyStore) Put(key string, e Entry) error {
	if key == s.hostKey {
		return errors.New("simulated host-key write failure")
	}
	return s.testStore.Put(key, e)
}

func TestDualWritingStorePutHostKeyFailureIsBestEffort(t *testing.T) {
	const (
		profileKey = "profile-a"
		hostKey    = "https://example.databricks.com"
	)
	inner := &failOnHostKeyStore{testStore: *newDualWritingTestStore(), hostKey: hostKey}
	arg := hostArg{key: profileKey, hostKey: hostKey}
	c := NewDualWritingStore(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Put(profileKey, Entry{Token: tok}), "host-key mirror failure must not propagate to primary Put")

	primary, err := inner.Lookup(profileKey)
	require.NoError(t, err)
	assert.Equal(t, tok, primary.Token, "primary write must persist even when host-key mirror fails")
}
