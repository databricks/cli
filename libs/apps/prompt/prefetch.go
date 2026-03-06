package prompt

import (
	"context"

	"github.com/databricks/cli/libs/apps/manifest"
)

// pagedConstructor creates a PagedFetcher with its first page loaded.
type pagedConstructor func(ctx context.Context) (*PagedFetcher, error)

// pagedConstructors maps resource types to their paged lister constructor.
var pagedConstructors = map[string]pagedConstructor{
	ResourceTypeSQLWarehouse:      NewPagedSQLWarehouses,
	ResourceTypeJob:               NewPagedJobs,
	ResourceTypeServingEndpoint:   NewPagedServingEndpoints,
	ResourceTypeGenieSpace:        NewPagedGenieSpaces,
	ResourceTypeExperiment:        NewPagedExperiments,
	ResourceTypeUCConnection:      NewPagedConnections,
	ResourceTypeVectorSearchIndex: NewPagedVectorSearchIndexes,
}

// PrefetchResources kicks off a background goroutine for every resource type
// found in resources. Each goroutine fetches the first page (pageSize items)
// and stores a PagedFetcher with the iterator alive for subsequent LoadMore
// calls. The function returns immediately with a context carrying the cache.
// Resource types without a registered constructor (e.g., secrets or UC volumes
// that require multi-step prompts) are silently skipped.
func PrefetchResources(ctx context.Context, resources []manifest.Resource) context.Context {
	cache := CacheFromContext(ctx)
	if cache == nil {
		cache = NewResourceCache()
	}

	for _, r := range resources {
		ctor, ok := pagedConstructors[r.Type]
		if !ok {
			continue
		}
		if cache.GetFetcher(r.Type) != nil {
			continue
		}
		f := &PagedFetcher{done: make(chan struct{})}
		cache.SetFetcher(r.Type, f)
		go func(create pagedConstructor) {
			defer close(f.done)
			fetcher, err := create(ctx)
			if err != nil {
				f.Err = err
				return
			}
			f.Items = fetcher.Items
			f.HasMore = fetcher.HasMore
			f.loadMore = fetcher.loadMore
		}(ctor)
	}

	return ContextWithCache(ctx, cache)
}
