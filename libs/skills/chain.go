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
// Both fall back only for an error that wraps ErrSourceUnreachable, and not
// once the caller's context is canceled.
//
//deadcode:allow the skills commands call it once they are wired to the chain
func NewChain(live Source, cache *Cache, bundled Source) Source {
	return newChain(live, cache, bundled, time.Now)
}

func newChain(live Source, cache *Cache, bundled Source, now func() time.Time) *chain {
	return &chain{live: live, cache: cache, bundled: bundled, now: now}
}

// catalogRead is the catalog a read serves, with what catalog() learned on the way.
type catalogRead struct {
	cat *Catalog
	// bundled is the bundled catalog when catalog() read it, else nil.
	bundled *Catalog
	// liveErr is the live failure behind a stale catalog, else nil.
	liveErr error
}

func (c *chain) Catalog(ctx context.Context) (*Catalog, error) {
	cached, _ := c.cache.Latest()
	read, err := c.catalog(ctx, cached)
	if err != nil {
		return nil, err
	}
	return read.cat, nil
}

// catalog returns the catalog to serve, given the newest cached catalog, which
// is nil when nothing is cached.
func (c *chain) catalog(ctx context.Context, cached *Catalog) (*catalogRead, error) {
	// A negative age, from a clock that moved back, is not fresh.
	if cached != nil {
		if age := c.now().Sub(cached.FetchedAt); age >= 0 && age < catalogFreshFor {
			return &catalogRead{cat: fromCache(cached, false)}, nil
		}
	}
	live, liveErr := c.live.Catalog(ctx)
	if liveErr == nil {
		live.FetchedAt = c.now()
		if err := c.cache.putCatalog(storedCopy(cached, live)); err != nil {
			log.Debugf(ctx, "[skills] cache catalog: %v", err)
		}
		return &catalogRead{cat: live}, nil
	}
	if !fallsBack(ctx, liveErr) {
		return nil, liveErr
	}
	if cached != nil {
		return &catalogRead{cat: fromCache(cached, true), liveErr: liveErr}, nil
	}
	bundled, err := c.bundled.Catalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("no skills catalog available: live source: %w; bundled source: %w", liveErr, err)
	}
	bundled.Stale = true
	return &catalogRead{cat: bundled, bundled: bundled, liveErr: liveErr}, nil
}

// storedCopy returns the copy of live to cache. When a clock moved back has
// stamped the newest cached catalog at or after live, the copy is stamped one
// nanosecond after it, so Latest keeps the newest write whatever the release
// versions are (the cache breaks ties by version as a string).
func storedCopy(newest, live *Catalog) *Catalog {
	cp := *live
	if newest != nil && !cp.FetchedAt.After(newest.FetchedAt) {
		cp.FetchedAt = newest.FetchedAt.Add(time.Nanosecond)
	}
	return &cp
}

// fallsBack reports whether err lets the chain fall back. Only an unreachable
// source does, and only while the caller still wants the answer: once the
// caller has canceled, the live error is returned instead.
func fallsBack(ctx context.Context, err error) bool {
	return errors.Is(err, ErrSourceUnreachable) && ctx.Err() == nil
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
	read, err := c.catalog(ctx, cached)
	if err != nil {
		return nil, err
	}
	// A stale catalog means the live source already failed, so it is not asked again.
	if read.cat.Stale {
		return c.fallBack(ctx, read, skill, file)
	}
	data, liveErr := c.live.ReadFile(ctx, skill, file)
	if liveErr == nil {
		c.cacheFile(ctx, read.cat.Version, skill, file, data)
		return data, nil
	}
	if !fallsBack(ctx, liveErr) {
		return nil, liveErr
	}
	read.liveErr = liveErr
	return c.fallBack(ctx, read, skill, file)
}

// fallBack serves a file from the bundled copy, when its release is the
// catalog's release. The live failure stays in the error.
func (c *chain) fallBack(ctx context.Context, read *catalogRead, skill, file string) ([]byte, error) {
	data, err := c.readBundled(ctx, read.cat, read.bundled, skill, file)
	if err != nil {
		return nil, fmt.Errorf("live source: %w; %w", read.liveErr, err)
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
