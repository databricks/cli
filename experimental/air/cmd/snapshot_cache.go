package aircmd

// Warm snapshot cache for the plain_tar path (working tree, no git ref). The Python
// CLI and the git_archive path re-pack the whole tree every run; for a large repo the
// file walk + read + gzip dominates submit latency. This keeps a warm, uncompressed
// tar of the tree on local disk plus a manifest of each member's identity (size+mtime)
// and byte range. On the next run we stat the file set, and rebuild the tarball by
// copying unchanged members verbatim from the warm tar — reading only changed files
// from disk — before gzipping the upload. Nothing changed is the degenerate case: we
// just recompress the warm tar. The cache is keyed by (repo path, config path,
// include_paths), so distinct repos or configs never share an entry, and it is gated
// to large trees (below the threshold a plain re-pack is cheap enough not to bother).

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/klauspost/pgzip"
)

const (
	// snapshotCacheVersion invalidates on-disk caches when the layout below changes.
	snapshotCacheVersion      = "v1"
	snapshotCacheManifestName = "manifest.json"
	snapshotCacheTarName      = "snapshot.tar"

	// snapshotCacheMinBytes gates the cache to trees large enough that the walk+read
	// cost dominates. Below it a plain re-pack is cheap and the cache bookkeeping
	// isn't worth it. Rough heuristic, not exact — tune from measurement.
	snapshotCacheMinBytes = 64 << 20 // 64 MiB
)

// tarTrailer is the two zero blocks that mark end-of-archive. writeSnapshot appends
// it because members are streamed without the tar.Writer's own Close (which would
// embed a trailer between members).
var tarTrailer = make([]byte, 2*512)

// cacheEntry records a member's identity for change detection (size+mtime) and its
// byte range within the warm snapshot.tar, so an unchanged member can be copied
// verbatim instead of re-read from disk.
type cacheEntry struct {
	Size    int64 `json:"size"`
	ModTime int64 `json:"mtime_ns"`
	Offset  int64 `json:"offset"`
	Length  int64 `json:"length"`
}

// snapshotManifest is the on-disk index of a warm snapshot.tar.
type snapshotManifest struct {
	Version string                `json:"version"`
	DirName string                `json:"dir_name"`
	Entries map[string]cacheEntry `json:"entries"` // keyed by slash-separated relative path
}

// snapshotCacheKey is a stable digest of the inputs that determine the tar's content
// set. Different repos, config files, or include_paths get different cache folders.
func snapshotCacheKey(absRepo, absConfig string, includePaths []string) string {
	paths := slices.Clone(includePaths)
	slices.Sort(paths)
	material := strings.Join(append([]string{absRepo, absConfig, snapshotCacheVersion}, paths...), "\x00")
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}

func snapshotCacheDir(absRepo, absConfig string, includePaths []string) string {
	return filepath.Join(os.TempDir(), "databricks", ".air", snapshotCacheKey(absRepo, absConfig, includePaths))
}

// packagePlainTarWithCache writes the working-tree tarball to outputTarball, using the
// warm cache when the tree is large enough. Small trees are packed fresh without
// touching the cache.
func packagePlainTarWithCache(ctx context.Context, repoPath, configPath string, includePaths []string, isGitRepo bool, outputTarball string) (err error) {
	start := time.Now()
	files, err := snapshotFiles(ctx, repoPath, includePaths, isGitRepo)
	if err != nil {
		return err
	}
	listDone := time.Now()
	var total int64
	for _, f := range files {
		total += f.size
	}
	dirName := filepath.Base(repoPath)

	// One timing line comparable to the shell path's "snapshot profile", so a
	// cache-on vs --no-cache run can be compared directly. Debug-only.
	mode := "rebuild-cold"
	defer func() {
		log.Debugf(ctx, "air snapshot cache: mode=%s files=%d uncompressed_bytes=%d list=%s pack=%s",
			mode, len(files), total, listDone.Sub(start), time.Since(listDone))
	}()

	if total < snapshotCacheMinBytes {
		mode = "skip-small"
		return writeGzOnly(repoPath, dirName, files, outputTarball)
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return err
	}
	cacheDir := snapshotCacheDir(absRepo, absConfig, includePaths)
	tarPath := filepath.Join(cacheDir, snapshotCacheTarName)
	manifestPath := filepath.Join(cacheDir, snapshotCacheManifestName)

	old := loadSnapshotManifest(manifestPath)
	if old != nil && old.DirName == dirName && fileExists(tarPath) && !snapshotChanged(files, old) {
		mode = "hit-nochange"
		return gzipFile(tarPath, outputTarball)
	}

	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return fmt.Errorf("failed to create snapshot cache dir: %w", err)
	}
	if old != nil {
		mode = "rebuild-warm"
	}
	return rebuildWarmSnapshot(repoPath, dirName, files, old, tarPath, outputTarball)
}

// snapshotChanged reports whether the current file set differs from the manifest by
// any add, delete, or modification (mtime+size). Equal length plus every current file
// matching an entry means the sets are identical.
func snapshotChanged(files []snapshotFile, old *snapshotManifest) bool {
	if len(files) != len(old.Entries) {
		return true
	}
	for _, f := range files {
		e, ok := old.Entries[filepath.ToSlash(f.rel)]
		if !ok || e.Size != f.size || e.ModTime != f.modTime {
			return true
		}
	}
	return false
}

// rebuildWarmSnapshot writes a fresh warm tar (copying unchanged members from the old
// one when available) and its gzipped upload copy in a single pass, then atomically
// replaces the cached tar and manifest.
func rebuildWarmSnapshot(repoPath, dirName string, files []snapshotFile, old *snapshotManifest, tarPath, outputTarball string) (err error) {
	gz, closeGz, err := newGzFile(outputTarball)
	if err != nil {
		return err
	}
	defer func() { err = firstErr(err, closeGz()) }()

	newTarPath := tarPath + ".tmp"
	tarFile, err := os.Create(newTarPath)
	if err != nil {
		return fmt.Errorf("failed to create warm tar: %w", err)
	}
	defer func() {
		if err != nil {
			os.Remove(newTarPath)
		}
	}()

	var oldTar io.ReaderAt
	if old != nil {
		if f, e := os.Open(tarPath); e == nil {
			defer f.Close()
			oldTar = f
		} else {
			old = nil // warm tar gone; rebuild every member from disk
		}
	}

	manifest, err := writeSnapshot(repoPath, dirName, files, old, oldTar, tarFile, gz)
	if err != nil {
		tarFile.Close()
		return err
	}
	if err = tarFile.Close(); err != nil {
		return fmt.Errorf("failed to finalize warm tar: %w", err)
	}
	if err = closeGz(); err != nil {
		return err
	}
	if err = os.Rename(newTarPath, tarPath); err != nil {
		return fmt.Errorf("failed to install warm tar: %w", err)
	}
	return saveSnapshotManifest(filepath.Join(filepath.Dir(tarPath), snapshotCacheManifestName), manifest)
}

// writeGzOnly packs files straight to a gzipped tarball without persisting a cache,
// used for trees below the cache threshold.
func writeGzOnly(repoPath, dirName string, files []snapshotFile, outputTarball string) (err error) {
	gz, closeGz, err := newGzFile(outputTarball)
	if err != nil {
		return err
	}
	defer func() { err = firstErr(err, closeGz()) }()
	_, err = writeSnapshot(repoPath, dirName, files, nil, nil, nil, gz)
	return err
}

// writeSnapshot streams every member (sorted for determinism) to gzDst, and to tarDst
// too when non-nil, returning the manifest that indexes each member's byte range in
// the tarDst stream. When old+oldTar are set, an unchanged member (matching size+mtime)
// is copied verbatim from oldTar rather than re-read from disk.
func writeSnapshot(repoPath, dirName string, files []snapshotFile, old *snapshotManifest, oldTar io.ReaderAt, tarDst, gzDst io.Writer) (snapshotManifest, error) {
	dst := gzDst
	if tarDst != nil {
		dst = io.MultiWriter(tarDst, gzDst)
	}
	cw := &countWriter{w: dst}

	sorted := slices.Clone(files)
	slices.SortFunc(sorted, func(a, b snapshotFile) int {
		return strings.Compare(filepath.ToSlash(a.rel), filepath.ToSlash(b.rel))
	})

	entries := make(map[string]cacheEntry, len(sorted))
	for _, f := range sorted {
		rel := filepath.ToSlash(f.rel)
		start := cw.n

		reused := false
		if old != nil && oldTar != nil {
			if e, ok := old.Entries[rel]; ok && e.Size == f.size && e.ModTime == f.modTime {
				if _, err := io.Copy(cw, io.NewSectionReader(oldTar, e.Offset, e.Length)); err != nil {
					return snapshotManifest{}, fmt.Errorf("failed to copy cached member %q: %w", rel, err)
				}
				reused = true
			}
		}
		if !reused {
			if err := writeMember(cw, repoPath, path.Join(dirName, rel), f.rel); err != nil {
				return snapshotManifest{}, err
			}
		}
		entries[rel] = cacheEntry{Size: f.size, ModTime: f.modTime, Offset: start, Length: cw.n - start}
	}
	if _, err := cw.Write(tarTrailer); err != nil {
		return snapshotManifest{}, err
	}
	return snapshotManifest{Version: snapshotCacheVersion, DirName: dirName, Entries: entries}, nil
}

// writeMember streams one framed tar member (header + content + block padding) for the
// file at repoPath/rel, named name inside the archive. It deliberately Flushes rather
// than Closes the tar.Writer, so no end-of-archive trailer is written between members.
func writeMember(w io.Writer, repoPath, name, rel string) error {
	full := filepath.Join(repoPath, filepath.FromSlash(rel))
	info, err := os.Lstat(full)
	if err != nil {
		return fmt.Errorf("failed to stat %q: %w", rel, err)
	}
	link := ""
	if info.Mode()&os.ModeSymlink != 0 {
		if link, err = os.Readlink(full); err != nil {
			return fmt.Errorf("failed to read symlink %q: %w", rel, err)
		}
	}
	hdr, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return fmt.Errorf("failed to build tar header for %q: %w", rel, err)
	}
	hdr.Name = name

	tw := tar.NewWriter(w)
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("failed to write tar header for %q: %w", rel, err)
	}
	if info.Mode().IsRegular() {
		f, err := os.Open(full)
		if err != nil {
			return fmt.Errorf("failed to open %q: %w", rel, err)
		}
		defer f.Close()
		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("failed to archive %q: %w", rel, err)
		}
	}
	return tw.Flush()
}

// countWriter counts the bytes written through it, to record member byte offsets.
type countWriter struct {
	w io.Writer
	n int64
}

func (c *countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// newGzFile creates outputTarball and returns a parallel gzip writer over it plus an
// idempotent close that flushes the gzip stream and the file.
//
// gzip is parallel (klauspost/pgzip): compressing the whole tar is the dominant
// packaging cost on a large tree and is paid on every run — even a no-change cache hit
// re-gzips the warm tar — so it is spread across cores (~18x faster than compress/gzip
// on a 470 MiB tar). The level is DefaultCompression, not BestSpeed: the tarball is
// re-uploaded every run so its size matters, and with parallel compression a normal
// level is nearly free (a few hundred ms for ~15-18% fewer bytes). pgzip buffers its
// own blocks, so the tar writer's small writes parallelize fine without extra buffering;
// its output is an ordinary gzip stream any gunzip/tar reads, and it falls back to serial
// below one block — fine, since the cache only engages above snapshotCacheMinBytes.
func newGzFile(outputTarball string) (io.Writer, func() error, error) {
	f, err := os.Create(outputTarball)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create tarball: %w", err)
	}
	gz, err := pgzip.NewWriterLevel(f, pgzip.DefaultCompression)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	closed := false
	closeFn := func() error {
		if closed {
			return nil
		}
		closed = true
		return firstErr(gz.Close(), f.Close())
	}
	return gz, closeFn, nil
}

// gzipFile writes a BestSpeed gzip of src to outputTarball. Used on a no-change cache
// hit to recompress the warm tar without re-reading the working tree.
func gzipFile(src, outputTarball string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open warm tar: %w", err)
	}
	defer in.Close()
	gz, closeGz, err := newGzFile(outputTarball)
	if err != nil {
		return err
	}
	defer func() { err = firstErr(err, closeGz()) }()
	_, err = io.Copy(gz, in)
	return err
}

func loadSnapshotManifest(path string) *snapshotManifest {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var m snapshotManifest
	if err := json.Unmarshal(data, &m); err != nil || m.Version != snapshotCacheVersion {
		return nil
	}
	return &m
}

func saveSnapshotManifest(path string, m snapshotManifest) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write snapshot manifest: %w", err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}
