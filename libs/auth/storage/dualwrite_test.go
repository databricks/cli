package storage

import (
	"sync"
	"testing"

	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	u2m_cache "github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// memoryCache is a minimal in-memory TokenCache used only by DualWrite tests.
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

// hostArg implements HostCacheKeyProvider so DualWrite mirrors the token to
// the configured host key.
type hostArg struct {
	key     string
	hostKey string
}

func (a hostArg) GetCacheKey() string     { return a.key }
func (a hostArg) GetHostCacheKey() string { return a.hostKey }

func TestDualWriteNoHostKey(t *testing.T) {
	cache := newMemoryCache()
	arg := plainArg{key: "profile-a"}
	tok := &oauth2.Token{AccessToken: "abc", RefreshToken: "r"}

	require.NoError(t, DualWrite(cache, arg, tok))

	got, err := cache.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, got)
	assert.Len(t, cache.tokens, 1)
}

func TestDualWriteHostKeyDistinct(t *testing.T) {
	cache := newMemoryCache()
	arg := hostArg{key: "profile-a", hostKey: "https://example.databricks.com"}
	tok := &oauth2.Token{AccessToken: "abc", RefreshToken: "r"}

	require.NoError(t, DualWrite(cache, arg, tok))

	primary, err := cache.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, primary)

	host, err := cache.Lookup("https://example.databricks.com")
	require.NoError(t, err)
	assert.Equal(t, tok, host)

	assert.Len(t, cache.tokens, 2)
}

func TestDualWriteHostKeyEqualsPrimary(t *testing.T) {
	cache := newMemoryCache()
	arg := hostArg{key: "https://example.databricks.com", hostKey: "https://example.databricks.com"}
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, DualWrite(cache, arg, tok))

	assert.Len(t, cache.tokens, 1)
}

func TestDualWriteDiscoveryArgWithDiscoveredHost(t *testing.T) {
	cache := newMemoryCache()
	arg, err := u2m.NewBasicDiscoveryOAuthArgument("profile-a")
	require.NoError(t, err)
	arg.SetDiscoveredHost("https://example.databricks.com")
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, DualWrite(cache, arg, tok))

	primary, err := cache.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, primary)

	host, err := cache.Lookup("https://example.databricks.com")
	require.NoError(t, err)
	assert.Equal(t, tok, host)
}

func TestDualWriteDiscoveryArgWithEmptyDiscoveredHost(t *testing.T) {
	cache := newMemoryCache()
	arg, err := u2m.NewBasicDiscoveryOAuthArgument("profile-a")
	require.NoError(t, err)
	tok := &oauth2.Token{AccessToken: "abc"}

	require.NoError(t, DualWrite(cache, arg, tok))

	assert.Len(t, cache.tokens, 1)
	primary, err := cache.Lookup("profile-a")
	require.NoError(t, err)
	assert.Equal(t, tok, primary)
}
