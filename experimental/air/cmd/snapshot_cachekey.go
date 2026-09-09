package aircmd

// This file packages a local code directory into a tarball, uploads it to the
// workspace (or a Volume), and records git provenance sidecars for cache
// invalidation — the Go port of the Python CLI's code_source snapshot path.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// snapshotPackagingVersion is bumped when packaging logic changes in a way that invalidates existing caches
const snapshotPackagingVersion = "v1"

// plainTarKeyVersion namespaces the plain_tar working-tree key (so it can never collide
// with a git_archive key) and lets us invalidate it if the fingerprint scheme changes.
// computePlainTarKey also folds in the shared snapshotPackagingVersion, so a
// packaging-logic bump invalidates both modes' keys.
const plainTarKeyVersion = "plaintar-v1"

// computePlainTarKey returns a content-addressed key for a working-tree snapshot: the
// SHA-256 over every file's path, size and mtime (sorted for stability). An unchanged
// tree yields the same key, so an already-uploaded tarball can be reused instead of
// re-packaged and re-uploaded. The fingerprint is size+mtime, not content — the same
// trade-off DABs file-sync makes — so an edit preserving both size and mtime is not seen.
func computePlainTarKey(files []snapshotFile) string {
	sorted := slices.Clone(files)
	slices.SortFunc(sorted, func(a, b snapshotFile) int {
		return strings.Compare(a.rel, b.rel)
	})

	h := sha256.New()
	for _, f := range sorted {
		fmt.Fprintf(h, "%s\x00%d\x00%d\n", filepath.ToSlash(f.rel), f.size, f.modTime)
	}
	fmt.Fprintf(h, "%s\x00%s", plainTarKeyVersion, snapshotPackagingVersion)
	return hex.EncodeToString(h.Sum(nil))
}

// computeSnapshotCacheKey returns a stable cache key for a snapshot tarball: the
// SHA-256 digest of (commitSHA, normalized includePaths, snapshotPackagingVersion).
// Changing any input yields a different entry.
func computeSnapshotCacheKey(commitSHA string, includePaths []string) string {
	var normalizedPaths string
	if len(includePaths) > 0 {
		trimmed := make([]string, len(includePaths))
		for i, p := range includePaths {
			trimmed[i] = strings.TrimSpace(p)
		}
		slices.Sort(trimmed)
		normalizedPaths = strings.Join(trimmed, "\n")
	}

	keyMaterial := commitSHA + "\n" + normalizedPaths + "\n" + snapshotPackagingVersion
	sum := sha256.Sum256([]byte(keyMaterial))
	return hex.EncodeToString(sum[:])
}
