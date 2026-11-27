package skills

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errLiveDown = fmt.Errorf("connection refused: %w", ErrSourceUnreachable)

// fakeSource serves a fixed catalog and files, or fails, and counts its calls.
type fakeSource struct {
	catalog *Catalog
	files   map[string]string
	err     error
	readErr error
	calls   int
}

func (f *fakeSource) Catalog(context.Context) (*Catalog, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	cat := *f.catalog
	return &cat, nil
}

func (f *fakeSource) ReadFile(_ context.Context, skill, file string) ([]byte, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if f.readErr != nil {
		return nil, f.readErr
	}
	data, ok := f.files[skill+"/"+file]
	if !ok {
		return nil, ErrUnknownFile
	}
	return []byte(data), nil
}

// newTestChain returns a chain with a temporary cache and a clock the test moves
// by changing *now.
func newTestChain(t *testing.T, live, bundled Source) (*chain, *time.Time) {
	t.Helper()
	now := t0
	cache := &Cache{dir: t.TempDir()}
	return newChain(live, cache, bundled, func() time.Time { return now }), &now
}

func TestChainLiveReachableIsCachedAndNotStale(t *testing.T) {
	c, now := newTestChain(t, &fakeSource{catalog: release("1.0.0")}, &fakeSource{catalog: release("0.9.0")})

	cat, err := c.Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", cat.Version)
	assert.False(t, cat.Stale)
	assert.Equal(t, *now, cat.FetchedAt)

	cached, ok := c.cache.Latest()
	require.True(t, ok)
	assert.Equal(t, "1.0.0", cached.Version)
}

func TestChainFreshCacheMakesNoLiveCall(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0")}
	c, now := newTestChain(t, live, &fakeSource{})

	_, err := c.Catalog(t.Context())
	require.NoError(t, err)
	*now = now.Add(catalogFreshFor - time.Second)
	_, err = c.Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, live.calls)

	*now = now.Add(2 * time.Second)
	_, err = c.Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 2, live.calls, "after the window the live source is asked again")
}

func TestChainFreshCacheIsServedAsCacheAndNotStale(t *testing.T) {
	c, _ := newTestChain(t, &fakeSource{catalog: release("1.0.0")}, &fakeSource{})

	_, err := c.Catalog(t.Context())
	require.NoError(t, err)
	cat, err := c.Catalog(t.Context())
	require.NoError(t, err)
	assert.False(t, cat.Stale, "a fresh cached copy is not a fallback")
	assert.Equal(t, cacheLabel, cat.Label)
}

func TestChainLiveUnreachableWithWarmCacheIsStale(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0")}
	c, now := newTestChain(t, live, &fakeSource{catalog: release("0.9.0")})

	_, err := c.Catalog(t.Context())
	require.NoError(t, err)
	firstFetch := *now

	live.catalog, live.err = nil, errLiveDown
	*now = now.Add(30 * time.Minute)
	cat, err := c.Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", cat.Version)
	assert.True(t, cat.Stale)
	assert.Equal(t, cacheLabel, cat.Label)
	assert.Equal(t, firstFetch, cat.FetchedAt, "the age is the time the cached copy was fetched")
}

func TestChainLiveUnreachableWithNoCacheUsesBundledAsStale(t *testing.T) {
	c, _ := newTestChain(t, &fakeSource{err: errLiveDown}, &fakeSource{catalog: release("0.9.0")})

	cat, err := c.Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "0.9.0", cat.Version)
	assert.True(t, cat.Stale)
	_, ok := c.cache.Latest()
	assert.False(t, ok, "the bundled catalog is not cached")
}

func TestChainNothingAvailableNamesLiveFailure(t *testing.T) {
	c, _ := newTestChain(t, &fakeSource{err: errLiveDown}, &fakeSource{err: errors.New("no bundle")})

	_, err := c.Catalog(t.Context())
	require.ErrorIs(t, err, errLiveDown)
}

func TestChainCachedFileReadMakesNoLiveCall(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0"), files: map[string]string{"alpha/SKILL.md": "live body"}}
	c, now := newTestChain(t, live, &fakeSource{})

	data, err := c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, "live body", string(data))
	calls := live.calls

	// Past the catalog's freshness window, the cached file is still read without a live call.
	*now = now.Add(catalogFreshFor + time.Minute)
	live.err = errLiveDown
	data, err = c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, "live body", string(data))
	assert.Equal(t, calls, live.calls, "a cached file read asks the live source for nothing")
}

func TestChainReadFileUnknownFileIsNotBundled(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0"), files: map[string]string{}}
	bundled := &fakeSource{catalog: release("1.0.0"), files: map[string]string{"alpha/SKILL.md": "bundled body"}}
	c, _ := newTestChain(t, live, bundled)

	_, err := c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.ErrorIs(t, err, ErrUnknownFile, "the live release does not list the file, so the bundled copy is not used")
}

func TestChainReadFileFallsBackToBundledWhenUnreachable(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0"), readErr: ErrSourceUnreachable}
	bundled := &fakeSource{catalog: release("1.0.0"), files: map[string]string{"alpha/SKILL.md": "bundled body"}}
	c, _ := newTestChain(t, live, bundled)

	data, err := c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, "bundled body", string(data))
}

func TestChainReadFileStaleCacheOtherBundledReleaseIsNotServed(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0")}
	bundled := &fakeSource{catalog: release("0.9.0"), files: map[string]string{"alpha/SKILL.md": "bundled body"}}
	c, now := newTestChain(t, live, bundled)

	_, err := c.Catalog(t.Context())
	require.NoError(t, err)
	live.catalog, live.err = nil, errLiveDown
	*now = now.Add(30 * time.Minute)

	_, err = c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.ErrorContains(t, err, "is not release 1.0.0", "0.9.0 bytes under a 1.0.0 catalog are not served")
}

func TestChainCatalogThenReadFileIsCachedForOfflineReads(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0"), files: map[string]string{"alpha/SKILL.md": "live body"}}
	c, now := newTestChain(t, live, &fakeSource{catalog: release("0.9.0")})

	_, err := c.Catalog(t.Context())
	require.NoError(t, err)
	data, err := c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, "live body", string(data))
	_, ok := c.cache.readFile("1.0.0", "alpha", "SKILL.md")
	require.True(t, ok, "the file is cached under the release the catalog came from")

	live.catalog, live.err = nil, errLiveDown
	*now = now.Add(30 * time.Minute)
	data, err = c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, "live body", string(data), "offline, the cached file is served")
}

func TestChainFreshCacheUnreachableFileReadFallsBackToBundled(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0")}
	bundled := &fakeSource{catalog: release("1.0.0"), files: map[string]string{"alpha/SKILL.md": "bundled body"}}
	c, _ := newTestChain(t, live, bundled)

	_, err := c.Catalog(t.Context())
	require.NoError(t, err)
	live.readErr = errLiveDown
	data, err := c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, "bundled body", string(data), "a fresh cached catalog does not change the fallback")
}

func TestChainNonUnreachableLiveErrorIsNotFallenBack(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0")}
	c, now := newTestChain(t, live, &fakeSource{catalog: release("0.9.0")})

	_, err := c.Catalog(t.Context())
	require.NoError(t, err)
	live.catalog, live.err = nil, errors.New("bad manifest")
	*now = now.Add(time.Hour)
	_, err = c.Catalog(t.Context())
	require.ErrorContains(t, err, "bad manifest", "only an unreachable live source falls back")
}

func TestChainFutureFetchedAtIsNotFresh(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0")}
	c, now := newTestChain(t, live, &fakeSource{})

	_, err := c.Catalog(t.Context())
	require.NoError(t, err)
	// The clock moved back, so the cached catalog is in the future.
	*now = now.Add(-time.Hour)
	_, err = c.Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 2, live.calls, "a catalog fetched in the future is not fresh")
}

func TestChainReadFileLiveReleaseChangeIsNotCachedUnderCachedCatalog(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0")}
	c, now := newTestChain(t, live, &fakeSource{})

	_, err := c.Catalog(t.Context())
	require.NoError(t, err)
	// Live moves on to 2.0.0 while the 1.0.0 catalog is still fresh.
	*now = now.Add(time.Minute)
	live.catalog = release("2.0.0")
	live.files = map[string]string{"alpha/SKILL.md": "2.0.0 body"}

	data, err := c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0 body", string(data))
	_, ok := c.cache.readFile("1.0.0", "alpha", "SKILL.md")
	assert.False(t, ok, "2.0.0 bytes are not stored under the 1.0.0 catalog")
	_, ok = c.cache.readFile("2.0.0", "alpha", "SKILL.md")
	assert.False(t, ok, "the 2.0.0 bytes are not cached, since the catalog is 1.0.0")
}

func TestChainCacheWriteFailureIsIgnored(t *testing.T) {
	// A cache folder under a regular file cannot be created, so every write fails.
	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))
	live := &fakeSource{catalog: release("1.0.0"), files: map[string]string{"alpha/SKILL.md": "live body"}}
	c := newChain(live, &Cache{dir: filepath.Join(blocker, "skills")}, &fakeSource{}, func() time.Time { return t0 })

	cat, err := c.Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", cat.Version)
	data, err := c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, "live body", string(data))
}

func TestChainNilCacheReadsLive(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0"), files: map[string]string{"alpha/SKILL.md": "live body"}}
	c := newChain(live, nil, &fakeSource{}, func() time.Time { return t0 })

	cat, err := c.Catalog(t.Context())
	require.NoError(t, err)
	assert.False(t, cat.Stale)
	data, err := c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, "live body", string(data))
}
