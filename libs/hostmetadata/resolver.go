// Package hostmetadata provides a cached implementation of the SDK's
// HostMetadataResolver, backed by the CLI's shared file cache.
//
// Importing this package (typically via a blank import from main) installs
// [config.DefaultHostMetadataResolverFactory] so every *config.Config the
// CLI constructs automatically gets the cached resolver on first EnsureResolved.
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

// errNegativeHit is returned from the positive-cache compute callback when the
// negative cache already has a sentinel for the host. It signals the outer
// resolver to return (nil, nil) without running fetch or writing to positive.
var errNegativeHit = errors.New("negative cache hit")

// hostFingerprint is the cache key for a given host.
type hostFingerprint struct {
	Host string `json:"host"`
}

// negativeSentinel records a failed host-metadata fetch in the negative cache.
type negativeSentinel struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

func init() {
	config.DefaultHostMetadataResolverFactory = func(cfg *config.Config) config.HostMetadataResolver {
		return NewResolver(cfg.DefaultHostMetadataResolver())
	}
}

// NewResolver returns a HostMetadataResolver backed by a positive and negative
// file cache. On positive hit it returns the cached metadata; on miss it
// probes the negative cache, then falls through to fetch and records failures
// so subsequent calls within negativeCacheTTL skip the network. The fetch
// function is invoked on miss, typically cfg.DefaultHostMetadataResolver().
func NewResolver(fetch config.HostMetadataResolver) config.HostMetadataResolver {
	// cache.NewCache uses ctx only for env lookups and cleanup-walk debug
	// logs; there is no cancellation signal to propagate. Using a background
	// context keeps NewResolver callable from sites without a caller ctx
	// in scope (e.g. the factory invoked from Config.EnsureResolved).
	ctx := context.Background() //nolint:gocritic // no caller ctx and cache.NewCache does not use ctx for cancellation.
	positive := cache.NewCache(ctx, positiveCacheComponent, positiveCacheTTL, nil)
	negative := cache.NewCache(ctx, negativeCacheComponent, negativeCacheTTL, nil)

	return func(ctx context.Context, host string) (*config.HostMetadata, error) {
		fp := hostFingerprint{Host: host}

		// Positive cache wraps the whole miss path so that the happy path (hit)
		// is a single disk read — no synthetic probe, no negative-cache traffic.
		meta, err := cache.GetOrCompute[*config.HostMetadata](ctx, positive, fp, func(ctx context.Context) (*config.HostMetadata, error) {
			sentinel, sErr := cache.GetOrCompute[*negativeSentinel](ctx, negative, fp, func(ctx context.Context) (*negativeSentinel, error) {
				return nil, errNotCached
			})
			if sErr == nil && sentinel != nil && sentinel.Error {
				log.Debugf(ctx, "[hostmetadata] negative cache hit for %s: %s", host, sentinel.Message)
				return nil, errNegativeHit
			}
			return fetch(ctx, host)
		})
		if err == nil {
			return meta, nil
		}
		if errors.Is(err, errNegativeHit) {
			return nil, nil
		}
		// Transient errors (cancellation, deadline) say nothing about the
		// host's long-term availability — don't cache them.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, nil
		}
		log.Debugf(ctx, "[hostmetadata] fetch failed for %s, recording negative: %v", host, err)
		// Best-effort write; ignore failures.
		_, _ = cache.GetOrCompute[*negativeSentinel](ctx, negative, fp, func(ctx context.Context) (*negativeSentinel, error) {
			return &negativeSentinel{Error: true, Message: err.Error()}, nil
		})
		return nil, nil
	}
}
