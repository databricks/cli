// Package hostmetadata provides a cached implementation of the SDK's
// HostMetadataResolver, backed by the CLI's shared file cache.
package hostmetadata

import (
	"context"
	"errors"
	"time"

	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
)

const (
	positiveCacheComponent = "host-metadata"
	negativeCacheComponent = "host-metadata-negative"
	positiveCacheTTL       = 1 * time.Hour
	negativeCacheTTL       = 60 * time.Second
)

// errNotCached forces a cache miss in the negative-cache probe without storing
// anything, since GetOrCompute only writes on success.
var errNotCached = errors.New("not cached")

// hostFingerprint is the cache key for a given host.
type hostFingerprint struct {
	Host string `json:"host"`
}

// negativeSentinel records a failed host-metadata fetch in the negative cache.
type negativeSentinel struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// Attach installs a caching HostMetadataResolver on cfg.
func Attach(cfg *config.Config) {
	// cache.NewCache uses ctx only for env lookups and cleanup-walk debug
	// logs; there is no cancellation signal to propagate. Using a background
	// context keeps Attach callable from sites without a caller ctx in scope
	// (e.g. bundle.Workspace.Client).
	ctx := context.Background() //nolint:gocritic // Attach has no caller ctx and cache.NewCache does not use ctx for cancellation.
	positive := cache.NewCache(ctx, positiveCacheComponent, positiveCacheTTL, nil)
	negative := cache.NewCache(ctx, negativeCacheComponent, negativeCacheTTL, nil)
	cfg.HostMetadataResolver = newResolver(cfg, positive, negative)
}

// newResolver returns a HostMetadataResolver that consults the negative cache
// before hitting the positive cache, and records failed fetches so subsequent
// calls within negativeCacheTTL skip the network entirely.
func newResolver(cfg *config.Config, positive, negative *cache.Cache) config.HostMetadataResolver {
	return func(ctx context.Context, host string) (*config.HostMetadata, error) {
		fp := hostFingerprint{Host: host}

		// Check negative cache first. errNotCached makes GetOrCompute skip the
		// write, so this is a read-only probe.
		sentinel, err := cache.GetOrCompute[*negativeSentinel](ctx, negative, fp, func(ctx context.Context) (*negativeSentinel, error) {
			return nil, errNotCached
		})
		if err == nil && sentinel != nil && sentinel.Error {
			log.Debugf(ctx, "[hostmetadata] negative cache hit for %s: %s", host, sentinel.Message)
			return nil, nil
		}

		// Positive cache: on miss, delegate to the SDK's default HTTP resolver.
		meta, err := cache.GetOrCompute[*config.HostMetadata](ctx, positive, fp, func(ctx context.Context) (*config.HostMetadata, error) {
			return cfg.DefaultHostMetadataResolver()(ctx, host)
		})
		if err != nil {
			log.Debugf(ctx, "[hostmetadata] fetch failed for %s, recording negative: %v", host, err)
			// Best-effort write to negative cache; ignore errors.
			_, _ = cache.GetOrCompute[*negativeSentinel](ctx, negative, fp, func(ctx context.Context) (*negativeSentinel, error) {
				return &negativeSentinel{Error: true, Message: err.Error()}, nil
			})
			return nil, nil
		}

		return meta, nil
	}
}
