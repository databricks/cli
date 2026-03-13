package prompt

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceCacheSetAndGet(t *testing.T) {
	cache := NewResourceCache()
	f := &PagedFetcher{Items: []ListItem{{ID: "1", Label: "one"}}}

	cache.SetFetcher("sql_warehouse", f)
	got := cache.GetFetcher("sql_warehouse")
	require.NotNil(t, got)
	assert.Equal(t, f, got)
}

func TestResourceCacheGetMissReturnsNil(t *testing.T) {
	cache := NewResourceCache()
	assert.Nil(t, cache.GetFetcher("nonexistent"))
}

func TestResourceCacheOverwrite(t *testing.T) {
	cache := NewResourceCache()
	f1 := &PagedFetcher{Items: []ListItem{{ID: "1"}}}
	f2 := &PagedFetcher{Items: []ListItem{{ID: "2"}}}

	cache.SetFetcher("key", f1)
	cache.SetFetcher("key", f2)

	got := cache.GetFetcher("key")
	assert.Equal(t, f2, got)
}

func TestResourceCacheConcurrentAccess(t *testing.T) {
	cache := NewResourceCache()
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := "key"
			f := &PagedFetcher{Items: []ListItem{{ID: string(rune('A' + i%26))}}}
			cache.SetFetcher(key, f)
			_ = cache.GetFetcher(key)
		}()
	}

	wg.Wait()
	assert.NotNil(t, cache.GetFetcher("key"))
}

func TestContextWithCacheRoundTrip(t *testing.T) {
	ctx := t.Context()
	cache := NewResourceCache()

	assert.Nil(t, CacheFromContext(ctx), "no cache in bare context")

	ctx = ContextWithCache(ctx, cache)
	got := CacheFromContext(ctx)
	require.NotNil(t, got)
	assert.Equal(t, cache, got)
}

func TestCacheFromContextNilOnMissingValue(t *testing.T) {
	assert.Nil(t, CacheFromContext(t.Context()))
}
