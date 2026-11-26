package skills

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var t0 = time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)

func fetchedAt(cat *Catalog, at time.Time) *Catalog {
	cat.FetchedAt = at
	return cat
}

func TestCacheLatestIsNewestAndSkipsCorruptEntries(t *testing.T) {
	dir := t.TempDir()
	cache := &Cache{dir: dir}
	_, ok := cache.Latest()
	assert.False(t, ok)

	require.NoError(t, cache.putCatalog(fetchedAt(release("1.0.0"), t0)))
	require.NoError(t, cache.putCatalog(fetchedAt(release("2.0.0"), t0.Add(time.Hour))))
	// A truncated entry, as a crash in a non-atomic writer would leave.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "3.0.0"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "3.0.0", catalogFile), []byte(`{"Version":"3`), 0o600))

	latest, ok := cache.Latest()
	require.True(t, ok)
	assert.Equal(t, "2.0.0", latest.Version)
}

func TestCacheInterruptedWriteLeavesNoEntry(t *testing.T) {
	dir := t.TempDir()
	cache := &Cache{dir: dir}
	require.NoError(t, cache.putCatalog(fetchedAt(release("1.0.0"), t0)))

	// The rename that publishes the new entry fails when its path is a non-empty folder.
	blocked := filepath.Join(dir, "2.0.0", catalogFile)
	require.NoError(t, os.MkdirAll(filepath.Join(blocked, "x"), 0o700))
	require.Error(t, cache.putCatalog(fetchedAt(release("2.0.0"), t0.Add(time.Hour))))

	entries, err := os.ReadDir(filepath.Dir(blocked))
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".tmp", "no temp file is left behind")
	}
	latest, ok := cache.Latest()
	require.True(t, ok)
	assert.Equal(t, "1.0.0", latest.Version)
}

func TestCacheRemovesOlderVersions(t *testing.T) {
	dir := t.TempDir()
	cache := &Cache{dir: dir}
	require.NoError(t, cache.putCatalog(fetchedAt(release("1.0.0"), t0)))
	require.NoError(t, cache.writeFile("1.0.0", "alpha", "SKILL.md", []byte("old")))
	require.NoError(t, cache.putCatalog(fetchedAt(release("2.0.0"), t0.Add(time.Hour))))
	require.NoError(t, cache.putCatalog(fetchedAt(release("3.0.0"), t0.Add(2*time.Hour))))

	_, err := os.Stat(filepath.Join(dir, "1.0.0"))
	assert.ErrorIs(t, err, os.ErrNotExist, "the release before the previous one is removed with its files")
	_, err = os.Stat(filepath.Join(dir, "2.0.0", catalogFile))
	assert.NoError(t, err, "the previous release is kept")
}

func TestCacheFilesAreKeyedByVersionAndStayInside(t *testing.T) {
	cache := &Cache{dir: t.TempDir()}

	require.NoError(t, cache.writeFile("1.0.0", "alpha", "references/guide.md", []byte("guide")))
	data, ok := cache.readFile("1.0.0", "alpha", "references/guide.md")
	require.True(t, ok)
	assert.Equal(t, "guide", string(data))
	_, ok = cache.readFile("2.0.0", "alpha", "references/guide.md")
	assert.False(t, ok)

	for _, c := range []struct{ version, skill, file string }{
		{"1.0.0", "..", "SKILL.md"},
		{"1.0.0", "a/b", "SKILL.md"},
		{"1.0.0", "alpha", "../../secret"},
		{"../1.0.0", "alpha", "SKILL.md"},
	} {
		assert.Error(t, cache.writeFile(c.version, c.skill, c.file, []byte("x")), "%+v", c)
		_, ok := cache.readFile(c.version, c.skill, c.file)
		assert.False(t, ok, "%+v", c)
	}
}

func TestNewCacheHonoursCacheEnabled(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_CACHE_DIR", t.TempDir())

	assert.Nil(t, NewCache(env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "false")), "a disabled cache stores nothing")

	enabled := NewCache(env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true"))
	require.NotNil(t, enabled)
	assert.Equal(t, filepath.Join(env.Get(ctx, "DATABRICKS_CACHE_DIR"), build.GetInfo().GetSanitizedVersion(), "skills"), enabled.dir)
}

func TestNewCacheWithoutCacheFolderStoresNothing(t *testing.T) {
	// With no cache folder configured and no user cache dir, the cache is off
	// rather than an error, as in libs/cache.
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("LOCALAPPDATA", "")
	ctx := env.Set(t.Context(), "DATABRICKS_CACHE_DIR", "")

	assert.Nil(t, NewCache(ctx))
}
