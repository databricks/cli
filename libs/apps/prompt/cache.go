package prompt

import (
	"context"
	"sync"
)

type resourceCacheKey struct{}

// ResourceCache stores pre-fetched PagedFetcher instances so prompt functions
// can skip the initial API fetch when data has already been loaded in parallel.
type ResourceCache struct {
	mu       sync.RWMutex
	fetchers map[string]*PagedFetcher
}

// NewResourceCache creates an empty cache.
func NewResourceCache() *ResourceCache {
	return &ResourceCache{fetchers: make(map[string]*PagedFetcher)}
}

// SetFetcher stores a pre-created PagedFetcher for a resource type.
func (c *ResourceCache) SetFetcher(resourceType string, f *PagedFetcher) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fetchers[resourceType] = f
}

// GetFetcher returns the cached PagedFetcher for a resource type, or nil.
func (c *ResourceCache) GetFetcher(resourceType string) *PagedFetcher {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fetchers[resourceType]
}

// ContextWithCache returns a child context carrying the given resource cache.
func ContextWithCache(ctx context.Context, cache *ResourceCache) context.Context {
	return context.WithValue(ctx, resourceCacheKey{}, cache)
}

// CacheFromContext retrieves the resource cache from the context, or nil.
func CacheFromContext(ctx context.Context) *ResourceCache {
	if cache, ok := ctx.Value(resourceCacheKey{}).(*ResourceCache); ok {
		return cache
	}
	return nil
}
