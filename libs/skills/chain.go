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

// cacheLabel says where a catalog served from the cache came from.
const cacheLabel = "cached copy"

// chain serves skills from the live source, then the cache, then the bundled copy.
type chain struct {
	live    Source
	cache   *Cache
	bundled Source
	now     func() time.Time
}

// NewChain returns a Source that prefers the live source and falls back to
// the cache and then to the bundled copy. A nil cache stores nothing.
//
// Catalog serves a cached catalog younger than catalogFreshFor without a live
// call. Otherwise it asks the live source and caches the result. When the live
// source is unreachable, it serves the newest cached catalog, or the bundled
// catalog when nothing is cached, both marked Stale.
//
// ReadFile serves a file of the newest cached release with no live call.
// Otherwise it reads the file live, and caches it only when a live catalog read
// after the file read still reports the catalog's release. When the live source
// is unreachable, it serves the bundled copy, and only when the bundled release
// is the catalog's release.
//
// Both fall back only for an error that wraps ErrSourceUnreachable.
//
//deadcode:allow the skills commands call it once they are wired to the chain
func NewChain(live Source, cache *Cache, bundled Source) Source {
	return newChain(live, cache, bundled, time.Now)
}

func newChain(live Source, cache *Cache, bundled Source, now func() time.Time) *chain {
	return &chain{live: live, cache: cache, bundled: bundled, now: now}
}

func (c *chain) Catalog(ctx context.Context) (*Catalog, error) {
	cached, _ := c.cache.Latest()
	cat, _, err := c.catalog(ctx, cached)
	return cat, err
}

// catalog returns the catalog to serve, given the newest cached catalog, which
// is nil when nothing is cached. It also returns the bundled catalog when it
// had to read it, so ReadFile does not read it again.
func (c *chain) catalog(ctx context.Context, cached *Catalog) (*Catalog, *Catalog, error) {
	// A negative age, from a clock that moved back, is not fresh.
	if cached != nil {
		if age := c.now().Sub(cached.FetchedAt); age >= 0 && age < catalogFreshFor {
			return fromCache(cached, false), nil, nil
		}
	}
	live, liveErr := c.live.Catalog(ctx)
	if liveErr == nil {
		live.FetchedAt = c.now()
		if err := c.cache.putCatalog(live); err != nil {
			log.Debugf(ctx, "[skills] cache catalog: %v", err)
		}
		return live, nil, nil
	}
	if !errors.Is(liveErr, ErrSourceUnreachable) {
		return nil, nil, liveErr
	}
	if cached != nil {
		return fromCache(cached, true), nil, nil
	}
	bundled, err := c.bundled.Catalog(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("no skills catalog available: live source: %w; bundled source: %w", liveErr, err)
	}
	bundled.Stale = true
	return bundled, bundled, nil
}

// fromCache marks cat as served from the cache. It is stale when the live
// source could not refresh it.
func fromCache(cat *Catalog, stale bool) *Catalog {
	cat.Label = cacheLabel
	cat.Stale = stale
	return cat
}

func (c *chain) ReadFile(ctx context.Context, skill, file string) ([]byte, error) {
	cached, _ := c.cache.Latest()
	if cached != nil {
		if data, ok := c.cache.readFile(cached.Version, skill, file); ok {
			return data, nil
		}
	}
	cat, bundled, err := c.catalog(ctx, cached)
	if err != nil {
		return nil, err
	}
	// A stale catalog means the live source already failed, so it is not asked again.
	if cat.Stale {
		return c.readBundled(ctx, cat, bundled, skill, file)
	}
	data, liveErr := c.live.ReadFile(ctx, skill, file)
	if liveErr == nil {
		c.cacheFile(ctx, cat.Version, skill, file, data)
		return data, nil
	}
	if !errors.Is(liveErr, ErrSourceUnreachable) {
		return nil, liveErr
	}
	data, err = c.readBundled(ctx, cat, bundled, skill, file)
	if err != nil {
		return nil, fmt.Errorf("live source: %w; %w", liveErr, err)
	}
	return data, nil
}

// cacheFile stores data, read live, as a file of release version. A catalog
// cached under an older release may be behind the live one, so the file is
// stored only when a live catalog read after the file read still reports version.
// Releases move forward, so a newer release fails that check.
func (c *chain) cacheFile(ctx context.Context, version, skill, file string, data []byte) {
	if c.cache == nil {
		return
	}
	cur, err := c.live.Catalog(ctx)
	if err != nil || cur.Version != version {
		return
	}
	if err := c.cache.writeFile(version, skill, file, data); err != nil {
		log.Debugf(ctx, "[skills] cache file %q of skill %q: %v", file, skill, err)
	}
}

// readBundled reads a file from the bundled copy, and only when the bundled
// release is the release of cat. bundled is the bundled catalog when catalog()
// already read it, and nil when it has to be read here.
func (c *chain) readBundled(ctx context.Context, cat, bundled *Catalog, skill, file string) ([]byte, error) {
	if bundled == nil {
		var err error
		bundled, err = c.bundled.Catalog(ctx)
		if err != nil {
			return nil, fmt.Errorf("read %q of skill %q: bundled source: %w", file, skill, err)
		}
	}
	if bundled.Version != cat.Version {
		return nil, fmt.Errorf("read %q of skill %q: bundled release %s is not release %s", file, skill, bundled.Version, cat.Version)
	}
	data, err := c.bundled.ReadFile(ctx, skill, file)
	if err != nil {
		return nil, fmt.Errorf("read %q of skill %q: bundled source: %w", file, skill, err)
	}
	return data, nil
}
