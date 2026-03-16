package prompt

import (
	"context"

	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/databricks/cli/libs/log"
)

// pagedConstructor creates a PagedFetcher with its first page loaded.
type pagedConstructor func(ctx context.Context) (*PagedFetcher, error)

// pagedConstructors maps resource types to their paged lister constructor.
var pagedConstructors = map[string]pagedConstructor{
	ResourceTypeSQLWarehouse:      ListSQLWarehouses,
	ResourceTypeJob:               ListJobs,
	ResourceTypeServingEndpoint:   ListServingEndpoints,
	ResourceTypeGenieSpace:        ListGenieSpaces,
	ResourceTypeExperiment:        ListExperiments,
	ResourceTypeUCConnection:      ListConnections,
	ResourceTypeVectorSearchIndex: ListVectorSearchIndexes,
}

// Internal cache keys for first-step fetchers used by multi-step prompts.
const (
	cacheKeyCatalogs          = "_catalogs"
	cacheKeyDatabaseInstances = "_database_instances"
	cacheKeyPostgresProjects  = "_postgres_projects"
)

// firstStepPrefetch maps multi-step resource types to the cache key and
// constructor for their first picker step (e.g., volume → catalogs).
var firstStepPrefetch = map[string]struct {
	cacheKey string
	ctor     pagedConstructor
}{
	ResourceTypeVolume:     {cacheKeyCatalogs, ListCatalogs},
	ResourceTypeUCFunction: {cacheKeyCatalogs, ListCatalogs},
	ResourceTypeDatabase:   {cacheKeyDatabaseInstances, ListDatabaseInstances},
	ResourceTypePostgres:   {cacheKeyPostgresProjects, ListPostgresProjects},
}

// PrefetchResources kicks off a background goroutine for every resource type
// found in resources. Each goroutine fetches the first page (pageSize items)
// and stores a PagedFetcher with the iterator alive for subsequent LoadMore
// calls. The function returns immediately with a context carrying the cache.
//
// For single-step resources (warehouses, jobs, etc.) the fetcher is stored
// under the resource type key. For multi-step resources (volumes, functions,
// databases, postgres) the first-step fetcher (catalogs, instances, projects)
// is prefetched under an internal cache key so the first picker renders
// instantly.
//
// Goroutine lifecycle: each goroutine makes one SDK API call to fetch the
// first page. The goroutines respect the provided context — when the context
// is cancelled (e.g. Ctrl+C), the SDK HTTP client aborts the request and the
// goroutine terminates shortly after. Callers do not need to explicitly join
// the goroutines; they are short-lived and self-draining.
func PrefetchResources(ctx context.Context, resources []manifest.Resource) context.Context {
	cache := CacheFromContext(ctx)
	if cache == nil {
		cache = NewResourceCache()
	}

	for _, r := range resources {
		// Single-step resources — full paged fetcher.
		if ctor, ok := pagedConstructors[r.Type]; ok {
			if cache.GetFetcher(r.Type) == nil {
				log.Debugf(ctx, "Prefetching resource type %q", r.Type)
				startPrefetch(ctx, cache, r.Type, ctor)
			}
		}

		// Multi-step resources — prefetch the first step.
		if fs, ok := firstStepPrefetch[r.Type]; ok {
			if cache.GetFetcher(fs.cacheKey) == nil {
				log.Debugf(ctx, "Prefetching first step %q for resource type %q", fs.cacheKey, r.Type)
				startPrefetch(ctx, cache, fs.cacheKey, fs.ctor)
			}
		}
	}

	return ContextWithCache(ctx, cache)
}

// startPrefetch launches a background goroutine that creates a PagedFetcher
// and stores it in the cache under the given key. The fetcher's done channel
// is closed after all fields are written, establishing a happens-before
// relationship — callers must call WaitForFirstPage before reading fields.
func startPrefetch(ctx context.Context, cache *ResourceCache, key string, ctor pagedConstructor) {
	f := &PagedFetcher{done: make(chan struct{})}
	cache.SetFetcher(key, f)
	go func() {
		defer close(f.done)
		fetcher, err := ctor(ctx)
		if err != nil {
			f.Err = err
			return
		}
		f.Items = fetcher.Items
		f.HasMore = fetcher.HasMore
		f.Capped = fetcher.Capped
		f.loadMore = fetcher.loadMore
	}()
}
