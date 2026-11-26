package skills

import (
	"context"
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
// ReadFile serves a file from the newest cached release when that release has it,
// with no live call at any catalog age. Otherwise it reads the file live and caches
// it, and falls back to the bundled copy.
//
//deadcode:allow the skills commands call it once they are wired to the chain
func NewChain(live Source, cache *Cache, bundled Source) Source {
	return newChain(live, cache, bundled, time.Now)
}

func newChain(live Source, cache *Cache, bundled Source, now func() time.Time) *chain {
	return &chain{live: live, cache: cache, bundled: bundled, now: now}
}

func (c *chain) Catalog(ctx context.Context) (*Catalog, error) {
	cached, hasCached := c.cache.Latest()
	if hasCached && c.now().Sub(cached.FetchedAt) < catalogFreshFor {
		return cached, nil
	}
	live, liveErr := c.live.Catalog(ctx)
	if liveErr == nil {
		live.FetchedAt = c.now()
		if err := c.cache.putCatalog(live); err != nil {
			log.Debugf(ctx, "[skills] cache catalog: %v", err)
		}
		return live, nil
	}
	if hasCached {
		cached.Stale = true
		return cached, nil
	}
	bundled, err := c.bundled.Catalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("no skills catalog available: live source: %w; bundled source: %w", liveErr, err)
	}
	bundled.Stale = true
	return bundled, nil
}

func (c *chain) ReadFile(ctx context.Context, skill, file string) ([]byte, error) {
	// A cached file of the newest cached release is served before any live call,
	// whatever the catalog's age, so a cached read makes no live call.
	if cached, ok := c.cache.Latest(); ok {
		if data, ok := c.cache.readFile(cached.Version, skill, file); ok {
			return data, nil
		}
	}
	cat, err := c.Catalog(ctx)
	if err != nil {
		return nil, err
	}
	if data, ok := c.cache.readFile(cat.Version, skill, file); ok {
		return data, nil
	}
	// A stale catalog means the live source already failed, so it is not asked again.
	if cat.Stale {
		return c.bundled.ReadFile(ctx, skill, file)
	}
	data, liveErr := c.live.ReadFile(ctx, skill, file)
	if liveErr == nil {
		if err := c.cache.writeFile(cat.Version, skill, file, data); err != nil {
			log.Debugf(ctx, "[skills] cache file %q of skill %q: %v", file, skill, err)
		}
		return data, nil
	}
	data, err = c.bundled.ReadFile(ctx, skill, file)
	if err != nil {
		return nil, fmt.Errorf("read %q of skill %q: live source: %w; bundled source: %w", file, skill, liveErr, err)
	}
	return data, nil
}
