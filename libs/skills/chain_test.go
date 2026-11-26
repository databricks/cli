package skills

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errLiveDown = errors.New("connection refused")

// fakeSource serves a fixed catalog and files, or fails, and counts its calls.
type fakeSource struct {
	catalog *Catalog
	files   map[string]string
	err     error
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
	data, ok := f.files[skill+"/"+file]
	if !ok {
		return nil, ErrUnknownFile
	}
	return []byte(data), nil
}

func release(version string) *Catalog {
	return &Catalog{Version: version, Label: "test", Skills: []Skill{{Name: "alpha", Files: []string{"SKILL.md"}}}}
}

// newTestChain returns a chain with a temporary cache and a clock the test moves
// by changing *now.
func newTestChain(t *testing.T, live, bundled Source) (*chain, *time.Time) {
	t.Helper()
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
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

func TestChainLiveUnreachableWithWarmCacheIsStale(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0")}
	c, now := newTestChain(t, live, &fakeSource{catalog: release("0.9.0")})

	_, err := c.Catalog(t.Context())
	require.NoError(t, err)
	fetchedAt := *now

	live.catalog, live.err = nil, errLiveDown
	*now = now.Add(30 * time.Minute)
	cat, err := c.Catalog(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", cat.Version)
	assert.True(t, cat.Stale)
	assert.Equal(t, fetchedAt, cat.FetchedAt, "the age is the time the cached copy was fetched")
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

func TestChainReadFileFallsBackToBundledUncached(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0"), files: map[string]string{}}
	bundled := &fakeSource{files: map[string]string{"alpha/SKILL.md": "bundled body"}}
	c, _ := newTestChain(t, live, bundled)

	data, err := c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, "bundled body", string(data))
	_, ok := c.cache.readFile("1.0.0", "alpha", "SKILL.md")
	assert.False(t, ok, "a file that was not read live is not cached")
}

func TestChainNilCacheReadsLive(t *testing.T) {
	live := &fakeSource{catalog: release("1.0.0"), files: map[string]string{"alpha/SKILL.md": "live body"}}
	c := newChain(live, nil, &fakeSource{}, time.Now)

	cat, err := c.Catalog(t.Context())
	require.NoError(t, err)
	assert.False(t, cat.Stale)
	data, err := c.ReadFile(t.Context(), "alpha", "SKILL.md")
	require.NoError(t, err)
	assert.Equal(t, "live body", string(data))
}
