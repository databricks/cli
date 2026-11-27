package skills

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/snapshot"
	"golang.org/x/mod/semver"
)

const (
	catalogFile = "catalog.json"
	filesDir    = "files"

	// keptVersions is how many release versions are kept: the current and the previous one.
	keptVersions = 2
)

// Cache keeps skills releases on disk, one folder per release version. The key
// is Catalog.Version, so a source whose catalogs are cached must set it to the
// release tag. The manifest's own version is a schema version ("2" for every
// release), and two releases would then share a folder. A read of a file never
// asks the source again; a write replaces its file atomically, so a reader sees
// the old bytes or the new ones. A nil Cache stores nothing.
type Cache struct {
	dir string
}

// pinnedCacheEnv turns the cache on for a pinned source, so a test that pins the
// source to a fake server has a cache to cover. Only tests set it.
// Unset, a pinned source stores nothing.
const pinnedCacheEnv = "DATABRICKS_SKILLS_TEST_PINNED_CACHE"

// pinnedDir is the folder, beside the release folder, that holds the catalogs of
// pinned sources. Each source has its own folder in it.
const pinnedDir = "skills-pinned"

// NewCache returns the skills cache under the CLI's cache folder, beside the
// other caches of this CLI version (see libs/cache). It is off unless
// DATABRICKS_SKILLS_PREVIEW is set. DATABRICKS_CACHE_ENABLED=false turns it off,
// as it does for the other caches, and so does a cache folder that cannot be found.
// A local folder, DATABRICKS_SKILLS_DIR, always turns it off, since its files are
// not a release. A pinned source, DATABRICKS_SKILLS_REF or DATABRICKS_SKILLS_BASE_URL,
// stores nothing unless pinnedCacheEnv is set. Then it stores in a folder keyed by
// the source, never in the release folder, which is keyed by release version only.
// A pinned branch ref or a mirror can change its files without changing its
// version. ReadFile serves a cached file before it asks the source, and cacheFile
// checks only the version, so such a source can keep stale files until its
// version changes. The release folder is not affected, since a release is fixed.
// The returned nil Cache stores nothing.
//
//deadcode:allow the skills commands call it once they are wired to the chain
func NewCache(ctx context.Context) *Cache {
	if !Enabled(ctx) {
		return nil
	}
	if enabled, ok := env.GetBool(ctx, "DATABRICKS_CACHE_ENABLED"); ok && !enabled {
		return nil
	}
	if env.Get(ctx, "DATABRICKS_SKILLS_DIR") != "" {
		return nil
	}
	ref := env.Get(ctx, "DATABRICKS_SKILLS_REF")
	baseURL := env.Get(ctx, "DATABRICKS_SKILLS_BASE_URL")
	pinned := ref != "" || baseURL != ""
	if pinned && !envTrue(ctx, pinnedCacheEnv) {
		return nil
	}
	base, err := cache.BaseDir(ctx)
	if err != nil {
		log.Debugf(ctx, "[skills] cache disabled: %v", err)
		return nil
	}
	version := build.GetInfo().GetSanitizedVersion()
	if pinned {
		return &Cache{dir: filepath.Join(base, version, pinnedDir, sourceKey(ref, baseURL))}
	}
	return &Cache{dir: filepath.Join(base, version, "skills")}
}

// sourceKey names the folder of a pinned source: a hash of its ref and base URL,
// so two pinned sources never share a folder.
func sourceKey(ref, baseURL string) string {
	return snapshot.HashFromContent([]byte("ref=" + ref + "\nbase_url=" + baseURL))
}

// Latest returns the cached catalog fetched most recently. It reads the disk only.
// It never sets Stale: the chain does, because only the chain knows whether a
// live read failed.
func (c *Cache) Latest() (*Catalog, bool) {
	if c == nil {
		return nil, false
	}
	cats := c.catalogs()
	if len(cats) == 0 {
		return nil, false
	}
	return cats[0], true
}

// putCatalog stores cat under its release version and keeps the release it
// stores plus the newest other release, which makes keptVersions in all. Every
// other folder is removed. Two processes may prune at once: a folder that one
// removes under a write by the other costs that write its entry, and the next
// read then misses. A miss is a live fetch, not a wrong answer.
func (c *Cache) putCatalog(cat *Catalog) error {
	if c == nil {
		return nil
	}
	if !isName(cat.Version) {
		return fmt.Errorf("cannot cache release version %q", cat.Version)
	}
	data, err := json.Marshal(cat)
	if err != nil {
		return err
	}
	path := filepath.Join(c.dir, cat.Version, catalogFile)
	if err := atomicfile.Write(path, data, 0o600, atomicfile.MkDir(0o700)); err != nil {
		return err
	}
	// Keep the release just stored and the newest others with a readable
	// catalog, until keptVersions are kept. The stored release may be older than
	// others, so it is kept first, not counted among the newest ones.
	keep := map[string]bool{cat.Version: true}
	for _, old := range c.catalogs() {
		if len(keep) == keptVersions {
			break
		}
		keep[old.Version] = true
	}
	entries, _ := os.ReadDir(c.dir)
	for _, e := range entries {
		if !keep[e.Name()] {
			_ = os.RemoveAll(filepath.Join(c.dir, e.Name()))
		}
	}
	return nil
}

// catalogs returns the readable cached catalogs, newest first. An entry that
// does not parse, or whose release version is not the name of its folder, is
// skipped, so a corrupt entry is never served. Catalogs fetched at the same
// time are ordered by version, so the order does not depend on the folder scan.
func (c *Cache) catalogs() []*Catalog {
	entries, _ := os.ReadDir(c.dir)
	var cats []*Catalog
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(c.dir, e.Name(), catalogFile))
		var cat Catalog
		if err != nil || json.Unmarshal(data, &cat) != nil || !isName(cat.Version) || cat.Version != e.Name() {
			continue
		}
		cats = append(cats, &cat)
	}
	slices.SortFunc(cats, func(a, b *Catalog) int {
		return cmp.Or(b.FetchedAt.Compare(a.FetchedAt), compareVersions(b.Version, a.Version))
	})
	return cats
}

// compareVersions orders release tags by semver, so 10.0.0 is newer than 9.0.0.
// semver ranks an invalid version below a valid one; only two invalid versions
// fall to text order.
func compareVersions(a, b string) int {
	va, vb := "v"+strings.TrimPrefix(a, "v"), "v"+strings.TrimPrefix(b, "v")
	return cmp.Or(semver.Compare(va, vb), strings.Compare(a, b))
}

// readFile returns the cached bytes of a file of a release version.
func (c *Cache) readFile(version, skill, file string) ([]byte, bool) {
	if c == nil || !cacheable(version, skill, file) {
		return nil, false
	}
	data, err := os.ReadFile(c.filePath(version, skill, file))
	return data, err == nil
}

// writeFile stores the bytes of a file of a release version.
func (c *Cache) writeFile(version, skill, file string, data []byte) error {
	if c == nil {
		return nil
	}
	if !cacheable(version, skill, file) {
		return fmt.Errorf("cannot cache file %q of skill %q", file, skill)
	}
	return atomicfile.Write(c.filePath(version, skill, file), data, 0o600, atomicfile.MkDir(0o700))
}

func (c *Cache) filePath(version, skill, file string) string {
	return filepath.Join(c.dir, version, filesDir, skill, filepath.FromSlash(file))
}

// cacheable reports whether the names, which come from a source, stay inside
// the cache folder of their release. The file must be clean and slash-separated,
// and not ".", so that "./SKILL.md" is not cached under the name "SKILL.md" and
// "references\..\SKILL.md" does not share an entry with "SKILL.md" on Windows.
func cacheable(version, skill, file string) bool {
	return isName(version) && isName(skill) && !escapesDir(file) && path.Clean(file) == file &&
		file != "." && !strings.Contains(file, `\`)
}
