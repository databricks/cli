package storage

import (
	"errors"

	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"golang.org/x/oauth2"
)

// ToU2MTokenCache adapts a CLI Store to the SDK's u2m_cache.TokenCache so it
// can be passed to u2m.WithTokenCache for the U2M PersistentAuth flow, the one
// place the SDK requires that interface. The SDK's Store(key, nil) "delete"
// convention maps to Store.Delete.
func ToU2MTokenCache(s Store) cache.TokenCache {
	return &sdkTokenCache{store: s}
}

// sdkTokenCache is the ToU2MTokenCache adapter.
type sdkTokenCache struct {
	store Store
}

// Store implements u2m_cache.TokenCache. A nil token is the SDK's delete
// signal; everything else is a plain put with no metadata.
func (sdktc *sdkTokenCache) Store(key string, t *oauth2.Token) error {
	if t == nil {
		return sdktc.store.Delete(key)
	}
	return sdktc.store.Put(key, Entry{Token: t})
}

// Lookup implements cache.TokenCache, translating the CLI miss sentinel to
// the SDK's so the SDK's errors.Is(err, cache.ErrNotFound) branches fire.
func (sdktc *sdkTokenCache) Lookup(key string) (*oauth2.Token, error) {
	e, err := sdktc.store.Lookup(key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, cache.ErrNotFound
		}
		return nil, err
	}
	return e.Token, nil
}

var _ cache.TokenCache = (*sdkTokenCache)(nil)
