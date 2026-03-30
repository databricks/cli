package auth

import (
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"golang.org/x/oauth2"
)

type inMemoryTokenCache struct {
	Tokens map[string]*oauth2.Token
}

// Lookup implements TokenCache.
// Returns a copy to match real (file-backed) cache behavior, where each
// Lookup deserializes a fresh struct. Without the copy, callers that
// mutate the returned token (e.g. clearing RefreshToken) would corrupt
// entries shared across test cases.
func (i *inMemoryTokenCache) Lookup(key string) (*oauth2.Token, error) {
	token, ok := i.Tokens[key]
	if !ok {
		return nil, cache.ErrNotFound
	}
	cp := *token
	return &cp, nil
}

// Store implements TokenCache.
// Stores a copy to prevent callers from mutating cached entries after Store
// returns (mirrors file-backed cache semantics).
func (i *inMemoryTokenCache) Store(key string, t *oauth2.Token) error {
	if t == nil {
		delete(i.Tokens, key)
	} else {
		cp := *t
		i.Tokens[key] = &cp
	}
	return nil
}

var _ cache.TokenCache = (*inMemoryTokenCache)(nil)
