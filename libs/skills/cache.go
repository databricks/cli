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
)

const (
	catalogFile = "catalog.json"
	filesDir    = "files"

	// keptVersions is how many release versions are kept: the current and the previous one.
	keptVersions = 2
)

// Cache keeps skills releases on disk, one folder per release version, so a
// file of a release is written once and never replaced. A nil Cache stores nothing.
type Cache struct {
	dir string
}

// pinnedCacheEnv turns the cache on for a pinned source, so a test that pins the
// source to a fake server has a cache to cover. No acceptance test sets it yet.
// Unset, a pinned source stores nothing.
const pinnedCacheEnv = "DATABRICKS_SKILLS_TEST_PINNED_CACHE"

// pinnedDir is the folder, beside the release folder, that holds the catalogs of
// pinned sources. Each source has its own folder in it.
const pinnedDir = "skills-pinned"

// NewCache returns the skills cache under the CLI's cache folder, beside the
// other caches of this CLI version (see libs/cache). It is off unless
// DATABRICKS_SKILLS_PREVIEW is set. DATABRICKS_CACHE_ENABLED=false turns it off,
// as it does for the other caches, and so does a cache folder that cannot be found.
// A pinned source, DATABRICKS_SKILLS_REF or DATABRICKS_SKILLS_BASE_URL, stores
// nothing unless pinnedCacheEnv is set. Then it stores in a folder keyed by the
// source, never in the release folder, which is keyed by release version only.
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

// putCatalog stores cat under its release version and removes the releases
// older than the newest keptVersions.
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
	// Keep the newest versions with a readable catalog, and remove every other
	// folder, including one whose catalog does not parse.
	keep := map[string]bool{cat.Version: true}
	cats := c.catalogs()
	for _, old := range cats[:min(keptVersions, len(cats))] {
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
		return cmp.Or(b.FetchedAt.Compare(a.FetchedAt), strings.Compare(b.Version, a.Version))
	})
	return cats
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
// the cache folder of their release. The file must be clean, so that "./SKILL.md"
// is not cached under the name "SKILL.md" and reads do not depend on the cache.
func cacheable(version, skill, file string) bool {
	return isName(version) && isName(skill) && !escapesDir(file) && path.Clean(file) == file
}
