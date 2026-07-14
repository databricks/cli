package aicode

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
)

// snapshotPackagingVersion is bumped when packaging logic changes in a way that
// invalidates existing caches.
const snapshotPackagingVersion = "v1"

// computeSnapshotCacheKey returns a stable cache key for a snapshot tarball: the
// SHA-256 digest of (commitSHA, normalized includePaths, snapshotPackagingVersion).
// Changing any input yields a different entry. Ported from the Python CLI's
// cli/utils/snapshot.py so cache keys are byte-identical across the two CLIs.
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
