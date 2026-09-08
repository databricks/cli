package storage

import (
	"errors"

	"github.com/databricks/cli/libs/auth/u2m/cache"
	"golang.org/x/oauth2"
)

// ToU2MTokenCache adapts a CLI Store to the U2M cache.TokenCache interface.
// The Store(key, nil) delete convention maps to Store.Delete.
func ToU2MTokenCache(s Store) cache.TokenCache {
	return &u2mTokenCache{store: s}
}

// u2mTokenCache is the ToU2MTokenCache adapter.
type u2mTokenCache struct {
	store Store
}

// Store implements cache.TokenCache. A nil token is the U2M delete
// signal; everything else is a plain put with no metadata.
func (tc *u2mTokenCache) Store(key string, t *oauth2.Token) error {
	if t == nil {
		return tc.store.Delete(key)
	}
	return tc.store.Put(key, Entry{Token: t})
}

// Lookup implements cache.TokenCache, translating the storage miss sentinel
// to the U2M package's sentinel.
func (tc *u2mTokenCache) Lookup(key string) (*oauth2.Token, error) {
	e, err := tc.store.Lookup(key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, cache.ErrNotFound
		}
		return nil, err
	}
	return e.Token, nil
}

var _ cache.TokenCache = (*u2mTokenCache)(nil)
