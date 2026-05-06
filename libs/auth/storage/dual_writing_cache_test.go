package storage

import (
	"errors"
	"sync"
	"testing"

	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	u2m_cache "github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// memoryCache is a minimal in-memory TokenCache used only by wrapper tests.
type memoryCache struct {
	mu     sync.Mutex
	tokens map[string]*oauth2.Token
}

func newMemoryCache() *memoryCache {
	return &memoryCache{tokens: map[string]*oauth2.Token{}}
}

func (c *memoryCache) Store(key string, t *oauth2.Token) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t == nil {
		delete(c.tokens, key)
		return nil
	}
	c.tokens[key] = t
	return nil
}

func (c *memoryCache) Lookup(key string) (*oauth2.Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.tokens[key]
	if !ok {
		return nil, u2m_cache.ErrNotFound
	}
	return t, nil
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

func TestDualWritingCacheStorePrimaryMirrorsHost(t *testing.T) {
	inner := newMemoryCache()
	arg := hostArg{key: "profile-a", hostKey: "https://example.databricks.com"}
	c := NewDualWritingTokenCache(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc", RefreshToken: "r"}

	require.NoError(t, c.Store("profile-a", tok))

	primary, err := inner.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, primary)

	host, err := inner.Lookup("https://example.databricks.com")
	require.NoError(t, err)
	assert.Equal(t, tok, host)
}

func TestDualWritingCacheStoreNonPrimaryDoesNotMirror(t *testing.T) {
	// An older SDK still running its internal dualWrite will follow up the
	// primary Store with a Store(hostKey, t). The wrapper must pass that
	// second write through without re-expanding into another pair.
	inner := newMemoryCache()
	arg := hostArg{key: "profile-a", hostKey: "https://example.databricks.com"}
	c := NewDualWritingTokenCache(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Store("https://example.databricks.com", tok))

	host, err := inner.Lookup("https://example.databricks.com")
	require.NoError(t, err)
	assert.Equal(t, tok, host)
	_, err = inner.Lookup("profile-a")
	require.ErrorIs(t, err, u2m_cache.ErrNotFound)
}

func TestDualWritingCacheStoreNoHostKey(t *testing.T) {
	inner := newMemoryCache()
	arg := plainArg{key: "profile-a"}
	c := NewDualWritingTokenCache(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Store("profile-a", tok))

	got, err := inner.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, got)
	assert.Len(t, inner.tokens, 1)
}

func TestDualWritingCacheStoreHostKeyEqualsPrimary(t *testing.T) {
	inner := newMemoryCache()
	arg := hostArg{key: "https://example.databricks.com", hostKey: "https://example.databricks.com"}
	c := NewDualWritingTokenCache(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Store("https://example.databricks.com", tok))

	assert.Len(t, inner.tokens, 1)
}

func TestDualWritingCacheDiscoveryArgWithDiscoveredHost(t *testing.T) {
	inner := newMemoryCache()
	arg, err := u2m.NewBasicDiscoveryOAuthArgument("profile-a")
	require.NoError(t, err)
	arg.SetDiscoveredHost("https://example.databricks.com")
	c := NewDualWritingTokenCache(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Store("profile-a", tok))

	primary, err := inner.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, primary)

	host, err := inner.Lookup("https://example.databricks.com")
	require.NoError(t, err)
	assert.Equal(t, tok, host)
}

func TestDualWritingCacheDiscoveryArgWithEmptyDiscoveredHost(t *testing.T) {
	inner := newMemoryCache()
	arg, err := u2m.NewBasicDiscoveryOAuthArgument("profile-a")
	require.NoError(t, err)
	c := NewDualWritingTokenCache(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Store("profile-a", tok))

	assert.Len(t, inner.tokens, 1)
	primary, err := inner.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, primary)
}

func TestDualWritingCacheLookupDelegates(t *testing.T) {
	inner := newMemoryCache()
	arg := hostArg{key: "profile-a", hostKey: "https://example.databricks.com"}
	c := NewDualWritingTokenCache(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}
	require.NoError(t, inner.Store("profile-a", tok))

	got, err := c.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, got)

	_, err = c.Lookup("missing")
	require.ErrorIs(t, err, u2m_cache.ErrNotFound)
}

// failOnHostKeyCache returns an error when asked to write under hostKey;
// primary writes succeed. Used to verify the wrapper treats host-key
// mirrors as best-effort.
type failOnHostKeyCache struct {
	memoryCache
	hostKey string
}

func (c *failOnHostKeyCache) Store(key string, t *oauth2.Token) error {
	if key == c.hostKey {
		return errors.New("simulated host-key write failure")
	}
	return c.memoryCache.Store(key, t)
}

func TestDualWritingCacheStoreHostKeyFailureIsBestEffort(t *testing.T) {
	const (
		profileKey = "profile-a"
		hostKey    = "https://example.databricks.com"
	)
	inner := &failOnHostKeyCache{memoryCache: *newMemoryCache(), hostKey: hostKey}
	arg := hostArg{key: profileKey, hostKey: hostKey}
	c := NewDualWritingTokenCache(inner, arg)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, c.Store(profileKey, tok), "host-key mirror failure must not propagate to primary Store")

	primary, err := inner.Lookup(profileKey)
	require.NoError(t, err)
	assert.Equal(t, tok, primary, "primary write must persist even when host-key mirror fails")
}
