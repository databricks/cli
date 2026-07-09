// Package snapshot packages a local code directory into a tarball, uploads it to
// the workspace (or a Volume), and records git provenance sidecars for cache invalidation.
package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
)

// PackagingVersion is bumped when packaging logic changes in a way that
// invalidates existing caches (tarball structure, metadata format, path-handling
// semantics). It is folded into the cache key so a bump forces fresh entries.
const PackagingVersion = "v1"

// ComputeCacheKey returns a stable cache key for a snapshot tarball: the SHA-256
// digest of (commitSHA, normalized includePaths, PackagingVersion). Changing any
// input yields a different entry.
func ComputeCacheKey(commitSHA string, includePaths []string) string {
	var normalizedPaths string
	if len(includePaths) > 0 {
		trimmed := make([]string, len(includePaths))
		for i, p := range includePaths {
			trimmed[i] = strings.TrimSpace(p)
		}
		slices.Sort(trimmed)
		normalizedPaths = strings.Join(trimmed, "\n")
	}

	keyMaterial := commitSHA + "\n" + normalizedPaths + "\n" + PackagingVersion
	sum := sha256.Sum256([]byte(keyMaterial))
	return hex.EncodeToString(sum[:])
}
