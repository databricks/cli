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

// errNegativeHit is returned from the positive-cache compute callback when the
// negative cache already has a sentinel for the host. It signals the outer
// resolver to return (nil, nil) without running fetch or writing to positive.
var errNegativeHit = errors.New("negative cache hit")

type hostFingerprint struct {
	Host string `json:"host"`
}

// negativeSentinel marks a host whose last fetch failed. Only presence matters;
// the original error text is deliberately not persisted to disk.
type negativeSentinel struct {
	Error bool `json:"error"`
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
	// The SDK factory signature (func(cfg *config.Config) HostMetadataResolver)
	// gives us no caller ctx at construction, so Background is the only option
	// here. cache.NewCache uses ctx only for a one-time env lookup and
	// cleanup-walk logging; per-call ctx still flows through the returned
	// resolver below.
	ctx := context.Background() //nolint:gocritic // forced by SDK factory signature; see comment above.
	positive := cache.NewCache(ctx, positiveCacheComponent, positiveCacheTTL, nil)
	negative := cache.NewCache(ctx, negativeCacheComponent, negativeCacheTTL, nil)

	return func(ctx context.Context, host string) (*config.HostMetadata, error) {
		fp := hostFingerprint{Host: host}

		// Positive cache wraps the whole miss path so that the happy path (hit)
		// is a single disk read — no synthetic probe, no negative-cache traffic.
		meta, err := cache.GetOrCompute[*config.HostMetadata](ctx, positive, fp, func(ctx context.Context) (*config.HostMetadata, error) {
			if sentinel, ok := cache.Get[*negativeSentinel](ctx, negative, fp); ok && sentinel != nil && sentinel.Error {
				log.Debugf(ctx, "[hostmetadata] negative cache hit for %s", host)
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
		// host's long-term availability — don't write a negative sentinel.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, nil
		}
		log.Warnf(ctx, "[hostmetadata] fetch failed for %s, recording negative: %v", host, err)
		cache.Put(ctx, negative, fp, &negativeSentinel{Error: true})
		return nil, nil
	}
}
