package storage

import (
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	u2m_cache "github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"golang.org/x/oauth2"
)

// DualWritingTokenCache wraps a TokenCache so that every write under the
// primary OAuth cache key is also mirrored under the legacy host-based key.
// This preserves the cross-SDK compatibility convention historically
// implemented inside PersistentAuth.dualWrite in the SDK, now moved
// caller-side per the cache-ownership split between SDK and CLI.
//
// Mirroring happens inside Store, so every SDK-internal write (Challenge,
// refresh, discovery) dual-writes without requiring each call site to invoke
// a helper explicitly.
type DualWritingTokenCache struct {
	inner u2m_cache.TokenCache
	arg   u2m.OAuthArgument
}

// NewDualWritingTokenCache returns a TokenCache wrapping inner that mirrors
// writes made under arg.GetCacheKey() to the argument's host key when one
// can be derived (via DiscoveryOAuthArgument.GetDiscoveredHost or
// HostCacheKeyProvider.GetHostCacheKey).
func NewDualWritingTokenCache(inner u2m_cache.TokenCache, arg u2m.OAuthArgument) *DualWritingTokenCache {
	return &DualWritingTokenCache{inner: inner, arg: arg}
}

// Store implements [u2m_cache.TokenCache]. Writes under the primary key are
// also mirrored under the host key (when distinct); writes under any other
// key pass through unchanged so that a Store(hostKey, t) from an older SDK
// that still dual-writes internally does not recursively re-expand.
//
// The host-key mirror is best-effort: if the second Store fails, the error
// is silently dropped. The host-key entry is a backward-compat shim for old
// Go SDK versions (v0.61-v0.103) that still look up by host. Failing the
// whole Store call would break primary login over a non-essential mirror,
// so a stale host-key entry is the lesser harm.
func (c *DualWritingTokenCache) Store(key string, t *oauth2.Token) error {
	if err := c.inner.Store(key, t); err != nil {
		return err
	}
	primaryKey := c.arg.GetCacheKey()
	if key != primaryKey {
		return nil
	}
	hostKey := hostCacheKey(c.arg)
	if hostKey == "" || hostKey == primaryKey {
		return nil
	}
	_ = c.inner.Store(hostKey, t)
	return nil
}

// Lookup implements [u2m_cache.TokenCache]; delegates to the inner cache.
func (c *DualWritingTokenCache) Lookup(key string) (*oauth2.Token, error) {
	return c.inner.Lookup(key)
}

// hostCacheKey mirrors the SDK's former PersistentAuth.hostCacheKey:
// discovery arguments expose the host via GetDiscoveredHost (populated by
// Challenge); static arguments expose it via HostCacheKeyProvider.
func hostCacheKey(arg u2m.OAuthArgument) string {
	if discoveryArg, ok := arg.(u2m.DiscoveryOAuthArgument); ok {
		return discoveryArg.GetDiscoveredHost()
	}
	if hcp, ok := arg.(u2m.HostCacheKeyProvider); ok {
		return hcp.GetHostCacheKey()
	}
	return ""
}
