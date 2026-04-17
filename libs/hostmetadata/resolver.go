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
	Host string
}

// negativeSentinel records a failed host-metadata fetch in the negative cache.
type negativeSentinel struct {
	Error   bool
	Message string
}

// Attach creates caching wrappers for positive and negative host-metadata
// results and installs them on cfg.
func Attach(ctx context.Context, cfg *config.Config) error {
	positive := cache.NewCache(ctx, positiveCacheComponent, positiveCacheTTL, nil)
	negative := cache.NewCache(ctx, negativeCacheComponent, negativeCacheTTL, nil)
	cfg.HostMetadataResolver = newResolver(cfg, positive, negative)
	return nil
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
