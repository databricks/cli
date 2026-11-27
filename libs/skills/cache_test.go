package skills

import (
	"context"
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

func release(version string) *Catalog {
	return &Catalog{Version: version, Label: "test", Skills: []Skill{{Name: "alpha", Files: []string{"SKILL.md"}}}}
}

func TestCacheLatestIsNewestAndSkipsCorruptEntries(t *testing.T) {
	cache := &Cache{dir: t.TempDir()}
	_, ok := cache.Latest()
	assert.False(t, ok)

	// Each corrupt entry is newer than 2.0.0, so it wins unless its own skip
	// condition holds. The version-is-not-a-name case also fails the folder
	// check, so it does not isolate isName on its own.
	const newer = `"FetchedAt":"2026-11-26T12:00:00Z"`
	for _, c := range []struct{ name, folder, body string }{
		{"does not unmarshal", "3.0.0", `{"Version":"3.0.0",` + newer + `,"Skills":"not a list"}`},
		{"truncated", "4.0.0", `{"Version":"4.0.0",` + newer},
		{"version is not a name", "5.0.0", `{"Version":"../5.0.0",` + newer + `}`},
		{"version differs from folder", "6.0.0", `{"Version":"7.0.0",` + newer + `}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			cache := &Cache{dir: t.TempDir()}
			require.NoError(t, cache.putCatalog(fetchedAt(release("1.0.0"), t0)))
			require.NoError(t, cache.putCatalog(fetchedAt(release("2.0.0"), t0.Add(time.Hour))))
			require.NoError(t, os.MkdirAll(filepath.Join(cache.dir, c.folder), 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(cache.dir, c.folder, catalogFile), []byte(c.body), 0o600))

			latest, ok := cache.Latest()
			require.True(t, ok)
			assert.Equal(t, "2.0.0", latest.Version)
		})
	}
}

func TestCacheLatestBreaksTiesByVersion(t *testing.T) {
	cache := &Cache{dir: t.TempDir()}
	require.NoError(t, cache.putCatalog(fetchedAt(release("1.0.0"), t0)))
	require.NoError(t, cache.putCatalog(fetchedAt(release("2.0.0"), t0)))

	latest, ok := cache.Latest()
	require.True(t, ok)
	assert.Equal(t, "2.0.0", latest.Version)

	// Text order would put 9.0.0 after 10.0.0; semver puts 10.0.0 first.
	cache = &Cache{dir: t.TempDir()}
	require.NoError(t, cache.putCatalog(fetchedAt(release("9.0.0"), t0)))
	require.NoError(t, cache.putCatalog(fetchedAt(release("10.0.0"), t0)))
	latest, ok = cache.Latest()
	require.True(t, ok)
	assert.Equal(t, "10.0.0", latest.Version)
}

func TestCacheKeepsStoredReleaseAndNewestOther(t *testing.T) {
	dir := t.TempDir()
	cache := &Cache{dir: dir}
	require.NoError(t, cache.putCatalog(fetchedAt(release("2.0.0"), t0.Add(time.Hour))))
	require.NoError(t, cache.putCatalog(fetchedAt(release("3.0.0"), t0.Add(2*time.Hour))))

	// An older release is stored last. It is kept, and the newest other release
	// is kept with it, so the store holds two folders, not three.
	require.NoError(t, cache.putCatalog(fetchedAt(release("1.0.0"), t0)))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.ElementsMatch(t, []string{"1.0.0", "3.0.0"}, names)
}

func TestCacheSameSchemaVersionKeyedByRelease(t *testing.T) {
	// Two releases whose manifests both say version "2" are two releases: each
	// file is read from its own release, never from the other's folder.
	cache := &Cache{dir: t.TempDir()}
	require.NoError(t, cache.putCatalog(fetchedAt(release("v0.2.56"), t0)))
	require.NoError(t, cache.writeFile("v0.2.56", "alpha", "SKILL.md", []byte("old")))
	require.NoError(t, cache.putCatalog(fetchedAt(release("v0.2.57"), t0.Add(time.Hour))))

	_, ok := cache.readFile("v0.2.57", "alpha", "SKILL.md")
	assert.False(t, ok, "a release with no cached file is a miss, not the previous release's file")
	require.NoError(t, cache.writeFile("v0.2.57", "alpha", "SKILL.md", []byte("new")))

	data, ok := cache.readFile("v0.2.57", "alpha", "SKILL.md")
	require.True(t, ok)
	assert.Equal(t, "new", string(data))
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
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "0.5.0", "files"), 0o700))
	require.NoError(t, cache.putCatalog(fetchedAt(release("3.0.0"), t0.Add(2*time.Hour))))

	_, err := os.Stat(filepath.Join(dir, "1.0.0"))
	assert.ErrorIs(t, err, os.ErrNotExist, "the release before the previous one is removed with its files")
	_, err = os.Stat(filepath.Join(dir, "0.5.0"))
	assert.ErrorIs(t, err, os.ErrNotExist, "a folder with no readable catalog is removed")
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
		{"1.0.0", "alpha", "./SKILL.md"},
		{"1.0.0", "alpha", "references/../SKILL.md"},
		{"1.0.0", "alpha", "."},
		{"1.0.0", "alpha", `references\..\SKILL.md`},
		{"1.0.0", "alpha", `.\SKILL.md`},
		{"../1.0.0", "alpha", "SKILL.md"},
	} {
		assert.Error(t, cache.writeFile(c.version, c.skill, c.file, []byte("x")), "%+v", c)
		_, ok := cache.readFile(c.version, c.skill, c.file)
		assert.False(t, ok, "%+v", c)
	}
}

func TestNilCacheStoresNothing(t *testing.T) {
	var cache *Cache
	_, ok := cache.Latest()
	assert.False(t, ok)
	assert.NoError(t, cache.putCatalog(release("1.0.0")))
	assert.NoError(t, cache.writeFile("1.0.0", "alpha", "SKILL.md", []byte("x")))
	_, ok = cache.readFile("1.0.0", "alpha", "SKILL.md")
	assert.False(t, ok)
}

// newCacheTestCtx points the cache at a temporary folder and clears the pinned
// source variables, so a developer's shell cannot turn the cache off.
func newCacheTestCtx(t *testing.T) context.Context {
	ctx := env.Set(t.Context(), "DATABRICKS_CACHE_DIR", t.TempDir())
	ctx = env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_REF", "")
	return env.Set(ctx, "DATABRICKS_SKILLS_BASE_URL", "")
}

func TestNewCacheHonoursCacheEnabled(t *testing.T) {
	ctx := newCacheTestCtx(t)

	assert.Nil(t, NewCache(env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "false")), "a disabled cache stores nothing")

	enabled := NewCache(env.Set(ctx, "DATABRICKS_CACHE_ENABLED", "true"))
	require.NotNil(t, enabled)
	assert.Equal(t, filepath.Join(env.Get(ctx, "DATABRICKS_CACHE_DIR"), build.GetInfo().GetSanitizedVersion(), "skills"), enabled.dir)
}

func TestNewCacheOffForPinnedSource(t *testing.T) {
	ctx := newCacheTestCtx(t)
	assert.NotNil(t, NewCache(ctx))

	assert.Nil(t, NewCache(env.Set(ctx, "DATABRICKS_SKILLS_REF", "v1.0.0")), "a pinned ref stores nothing")
	assert.Nil(t, NewCache(env.Set(ctx, "DATABRICKS_SKILLS_BASE_URL", "http://localhost:8080")), "a custom base URL stores nothing")
}

func TestNewCacheWithoutCacheFolderStoresNothing(t *testing.T) {
	// With no cache folder configured and no user cache dir, the cache is off
	// rather than an error, as in libs/cache.
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("LOCALAPPDATA", "")
	ctx := env.Set(newCacheTestCtx(t), "DATABRICKS_CACHE_DIR", "")

	assert.Nil(t, NewCache(ctx))
}
