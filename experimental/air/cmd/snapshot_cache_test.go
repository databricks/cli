package aircmd

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extractTarball returns entry name -> content for a .tar.gz.
func extractTarball(t *testing.T, path string) map[string]string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gz.Close()

	out := map[string]string{}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		buf := make([]byte, hdr.Size)
		_, _ = tr.Read(buf)
		out[hdr.Name] = string(buf)
	}
	return out
}

func statFiles(t *testing.T, repo string, rels ...string) []snapshotFile {
	t.Helper()
	var files []snapshotFile
	for _, rel := range rels {
		info, err := os.Lstat(filepath.Join(repo, filepath.FromSlash(rel)))
		require.NoError(t, err)
		files = append(files, snapshotFile{rel: filepath.FromSlash(rel), size: info.Size(), modTime: info.ModTime().UnixNano()})
	}
	return files
}

func TestSnapshotCacheKey(t *testing.T) {
	base := snapshotCacheKey("/repo", "/repo/air.yaml", nil)
	assert.Equal(t, base, snapshotCacheKey("/repo", "/repo/air.yaml", nil), "stable for identical inputs")
	assert.NotEqual(t, base, snapshotCacheKey("/other", "/repo/air.yaml", nil), "repo path matters")
	assert.NotEqual(t, base, snapshotCacheKey("/repo", "/repo/other.yaml", nil), "config path matters")
	assert.NotEqual(t, base, snapshotCacheKey("/repo", "/repo/air.yaml", []string{"src"}), "include_paths matter")
	// include_paths order must not matter.
	assert.Equal(t,
		snapshotCacheKey("/repo", "/repo/air.yaml", []string{"a", "b"}),
		snapshotCacheKey("/repo", "/repo/air.yaml", []string{"b", "a"}))
}

func TestSnapshotChanged(t *testing.T) {
	old := &snapshotManifest{Entries: map[string]cacheEntry{
		"a.txt":    {Size: 1, ModTime: 100},
		"src/b.py": {Size: 2, ModTime: 200},
	}}
	unchanged := []snapshotFile{{rel: "a.txt", size: 1, modTime: 100}, {rel: filepath.FromSlash("src/b.py"), size: 2, modTime: 200}}
	assert.False(t, snapshotChanged(unchanged, old))

	modified := []snapshotFile{{rel: "a.txt", size: 1, modTime: 100}, {rel: filepath.FromSlash("src/b.py"), size: 2, modTime: 999}}
	assert.True(t, snapshotChanged(modified, old), "mtime change detected")

	resized := []snapshotFile{{rel: "a.txt", size: 5, modTime: 100}, {rel: filepath.FromSlash("src/b.py"), size: 2, modTime: 200}}
	assert.True(t, snapshotChanged(resized, old), "size change detected")

	removed := []snapshotFile{{rel: "a.txt", size: 1, modTime: 100}}
	assert.True(t, snapshotChanged(removed, old), "deletion detected")

	added := append(append([]snapshotFile(nil), unchanged...), snapshotFile{rel: "c.txt", size: 3, modTime: 300})
	assert.True(t, snapshotChanged(added, old), "addition detected")
}

func TestWarmSnapshotColdBuild(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "a.txt", "alpha")
	writeRepoFile(t, repo, "src/model.py", "print()")
	dirName := filepath.Base(repo)

	cacheDir := t.TempDir()
	tarPath := filepath.Join(cacheDir, snapshotCacheTarName)
	out := filepath.Join(t.TempDir(), "snap.tar.gz")

	files := statFiles(t, repo, "a.txt", "src/model.py")
	require.NoError(t, rebuildWarmSnapshot(repo, dirName, files, nil, tarPath, out))

	contents := extractTarball(t, out)
	assert.Equal(t, "alpha", contents[dirName+"/a.txt"])
	assert.Equal(t, "print()", contents[dirName+"/src/model.py"])

	// The warm tar and manifest are persisted for the next run.
	assert.FileExists(t, tarPath)
	m := loadSnapshotManifest(filepath.Join(cacheDir, snapshotCacheManifestName))
	require.NotNil(t, m)
	assert.Equal(t, dirName, m.DirName)
	assert.Len(t, m.Entries, 2)
}

func TestWarmSnapshotReusesUnchangedFromCache(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "a.txt", "alpha")
	writeRepoFile(t, repo, "keep.py", "keep")
	dirName := filepath.Base(repo)

	cacheDir := t.TempDir()
	tarPath := filepath.Join(cacheDir, snapshotCacheTarName)
	files := statFiles(t, repo, "a.txt", "keep.py")
	require.NoError(t, rebuildWarmSnapshot(repo, dirName, files, nil, tarPath, filepath.Join(t.TempDir(), "cold.tar.gz")))

	old := loadSnapshotManifest(filepath.Join(cacheDir, snapshotCacheManifestName))
	require.NotNil(t, old)

	// Delete keep.py from disk but keep it in the file list with its original
	// size+mtime: a correct rebuild must copy its bytes from the warm tar, proving
	// unchanged members are not re-read from disk.
	require.NoError(t, os.Remove(filepath.Join(repo, "keep.py")))

	out := filepath.Join(t.TempDir(), "warm.tar.gz")
	require.NoError(t, rebuildWarmSnapshot(repo, dirName, files, old, tarPath, out))

	contents := extractTarball(t, out)
	assert.Equal(t, "keep", contents[dirName+"/keep.py"], "unchanged member copied from warm tar")
	assert.Equal(t, "alpha", contents[dirName+"/a.txt"])
}

func TestWarmSnapshotRebuildEditAddDelete(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "a.txt", "alpha")
	writeRepoFile(t, repo, "b.txt", "bravo")
	dirName := filepath.Base(repo)

	cacheDir := t.TempDir()
	tarPath := filepath.Join(cacheDir, snapshotCacheTarName)
	files := statFiles(t, repo, "a.txt", "b.txt")
	require.NoError(t, rebuildWarmSnapshot(repo, dirName, files, nil, tarPath, filepath.Join(t.TempDir(), "cold.tar.gz")))
	old := loadSnapshotManifest(filepath.Join(cacheDir, snapshotCacheManifestName))
	require.NotNil(t, old)

	// Edit a.txt (changed), delete b.txt, add c.txt.
	writeRepoFile(t, repo, "a.txt", "alpha-v2")
	require.NoError(t, os.Remove(filepath.Join(repo, "b.txt")))
	writeRepoFile(t, repo, "c.txt", "charlie")

	out := filepath.Join(t.TempDir(), "warm.tar.gz")
	newFiles := statFiles(t, repo, "a.txt", "c.txt")
	require.NoError(t, rebuildWarmSnapshot(repo, dirName, newFiles, old, tarPath, out))

	contents := extractTarball(t, out)
	assert.Equal(t, "alpha-v2", contents[dirName+"/a.txt"], "edited file updated")
	assert.Equal(t, "charlie", contents[dirName+"/c.txt"], "added file present")
	_, hasB := contents[dirName+"/b.txt"]
	assert.False(t, hasB, "deleted file dropped")

	// The refreshed manifest reflects the new set.
	updated := loadSnapshotManifest(filepath.Join(cacheDir, snapshotCacheManifestName))
	require.NotNil(t, updated)
	assert.Len(t, updated.Entries, 2)
}

func TestGzipFileRoundTrip(t *testing.T) {
	repo := t.TempDir()
	writeRepoFile(t, repo, "a.txt", "alpha")
	dirName := filepath.Base(repo)

	cacheDir := t.TempDir()
	tarPath := filepath.Join(cacheDir, snapshotCacheTarName)
	files := statFiles(t, repo, "a.txt")
	require.NoError(t, rebuildWarmSnapshot(repo, dirName, files, nil, tarPath, filepath.Join(t.TempDir(), "cold.tar.gz")))

	// gzipFile recompresses the warm tar directly (the no-change hit path).
	out := filepath.Join(t.TempDir(), "reuse.tar.gz")
	require.NoError(t, gzipFile(tarPath, out))
	contents := extractTarball(t, out)
	assert.Equal(t, "alpha", contents[dirName+"/a.txt"])
}
