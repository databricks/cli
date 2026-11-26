package skills

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/log"
)

// catalogFreshFor is how long a cached catalog is served without asking the live source.
const catalogFreshFor = 10 * time.Minute

// chain serves skills from the live source, then the cache, then the bundled copy.
type chain struct {
	live    Source
	cache   *Cache
	bundled Source
	now     func() time.Time
}

// NewChain returns a Source that prefers the live source and falls back to
// the cache and then to the bundled copy. A nil cache caches nothing.
//
// Catalog serves a cached catalog younger than catalogFreshFor without a live
// call. Otherwise it asks the live source and caches the result. When the live
// source fails it serves the newest cached catalog marked Stale, or the bundled
// catalog marked Stale when nothing is cached.
//
// ReadFile serves a file of the newest cached release with no live call. Otherwise
// it reads the file live, caching it only under a catalog read live in the same
// call. The bundled copy stands in only for an unreachable live source, and only
// when it is the catalog's release.
//
//deadcode:allow the skills commands call it once they are wired to the chain
func NewChain(live Source, cache *Cache, bundled Source) Source {
	return newChain(live, cache, bundled, time.Now)
}

func newChain(live Source, cache *Cache, bundled Source, now func() time.Time) *chain {
	return &chain{live: live, cache: cache, bundled: bundled, now: now}
}

func (c *chain) Catalog(ctx context.Context) (*Catalog, error) {
	cat, _, err := c.catalog(ctx)
	return cat, err
}

// catalog is Catalog. It also reports whether the catalog was read from the
// live source in this call.
func (c *chain) catalog(ctx context.Context) (*Catalog, bool, error) {
	cached, hasCached := c.cache.Latest()
	if hasCached && c.now().Sub(cached.FetchedAt) < catalogFreshFor {
		return cached, false, nil
	}
	live, liveErr := c.live.Catalog(ctx)
	if liveErr == nil {
		live.FetchedAt = c.now()
		if err := c.cache.putCatalog(live); err != nil {
			log.Debugf(ctx, "[skills] cache catalog: %v", err)
		}
		return live, true, nil
	}
	if hasCached {
		cached.Stale = true
		return cached, false, nil
	}
	bundled, err := c.bundled.Catalog(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("no skills catalog available: live source: %w; bundled source: %w", liveErr, err)
	}
	bundled.Stale = true
	return bundled, false, nil
}

func (c *chain) ReadFile(ctx context.Context, skill, file string) ([]byte, error) {
	// A cached file of the newest cached release is served before any live call,
	// whatever the catalog's age, so a cached read makes no live call.
	if cached, ok := c.cache.Latest(); ok {
		if data, ok := c.cache.readFile(cached.Version, skill, file); ok {
			return data, nil
		}
	}
	cat, fromLive, err := c.catalog(ctx)
	if err != nil {
		return nil, err
	}
	// A stale catalog means the live source already failed, so it is not asked again.
	if cat.Stale {
		return c.readBundled(ctx, cat.Version, skill, file)
	}
	data, liveErr := c.live.ReadFile(ctx, skill, file)
	if liveErr == nil {
		// A cached catalog may be older than the live release, so its files are not cached.
		if fromLive {
			if err := c.cache.writeFile(cat.Version, skill, file, data); err != nil {
				log.Debugf(ctx, "[skills] cache file %q of skill %q: %v", file, skill, err)
			}
		}
		return data, nil
	}
	if !errors.Is(liveErr, ErrSourceUnreachable) {
		return nil, liveErr
	}
	data, err = c.readBundled(ctx, cat.Version, skill, file)
	if err != nil {
		return nil, fmt.Errorf("live source: %w; %w", liveErr, err)
	}
	return data, nil
}

// readBundled reads a file from the bundled copy, only when it is release version.
func (c *chain) readBundled(ctx context.Context, version, skill, file string) ([]byte, error) {
	bundled, err := c.bundled.Catalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("read %q of skill %q: bundled source: %w", file, skill, err)
	}
	if bundled.Version != version {
		return nil, fmt.Errorf("read %q of skill %q: bundled release %s is not release %s", file, skill, bundled.Version, version)
	}
	data, err := c.bundled.ReadFile(ctx, skill, file)
	if err != nil {
		return nil, fmt.Errorf("read %q of skill %q: bundled source: %w", file, skill, err)
	}
	return data, nil
}
