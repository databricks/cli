package cache

import (
	"context"

	"golang.org/x/oauth2"
)

type TokenCache interface {
	Store(key string, t *oauth2.Token) error
	Lookup(key string) (*oauth2.Token, error)
}

var tokenCache int

func WithTokenCache(ctx context.Context, c TokenCache) context.Context {
	return context.WithValue(ctx, &tokenCache, c)
}

func GetTokenCache(ctx context.Context) TokenCache {
	c, ok := ctx.Value(&tokenCache).(TokenCache)
	if !ok {
		return &FileTokenCache{}
	}
	return c
}
