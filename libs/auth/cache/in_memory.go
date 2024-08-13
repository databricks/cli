package cache

import (
	"golang.org/x/oauth2"
)

type InMemoryTokenCache struct {
	Tokens map[string]*oauth2.Token
}

// Lookup implements TokenCache.
func (i *InMemoryTokenCache) Lookup(key string) (*oauth2.Token, error) {
	token, ok := i.Tokens[key]
	if !ok {
		return nil, ErrNotConfigured
	}
	return token, nil
}

// Store implements TokenCache.
func (i *InMemoryTokenCache) Store(key string, t *oauth2.Token) error {
	i.Tokens[key] = t
	return nil
}

var _ TokenCache = (*InMemoryTokenCache)(nil)
