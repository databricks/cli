package storage

import (
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	u2m_cache "github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"golang.org/x/oauth2"
)

// DualWrite stores t under arg.GetCacheKey() (the primary key) and, if a
// legacy host-based cache key can be derived from arg, also stores t under
// that host key. Mirrors the convention used historically by
// PersistentAuth.dualWrite and hostCacheKey in the SDK.
func DualWrite(cache u2m_cache.TokenCache, arg u2m.OAuthArgument, t *oauth2.Token) error {
	primaryKey := arg.GetCacheKey()
	if err := cache.Store(primaryKey, t); err != nil {
		return err
	}
	hostKey := hostCacheKey(arg)
	if hostKey != "" && hostKey != primaryKey {
		if err := cache.Store(hostKey, t); err != nil {
			return err
		}
	}
	return nil
}

// hostCacheKey mirrors PersistentAuth.hostCacheKey in the SDK: discovery
// arguments learn their host from the OAuth callback, so their host key
// must be read via GetDiscoveredHost *after* Challenge(). Static arguments
// use HostCacheKeyProvider.
func hostCacheKey(arg u2m.OAuthArgument) string {
	if discoveryArg, ok := arg.(u2m.DiscoveryOAuthArgument); ok {
		return discoveryArg.GetDiscoveredHost()
	}
	if hcp, ok := arg.(u2m.HostCacheKeyProvider); ok {
		return hcp.GetHostCacheKey()
	}
	return ""
}
