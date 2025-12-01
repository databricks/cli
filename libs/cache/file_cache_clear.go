package cache

import (
	"context"
	"os"
)

// ClearFileCache removes all cached files from the Databricks cache directory.
// This clears the cache for ALL CLI versions, not just the current version.
// The cache is organized as: <cache-base>/<version>/<component>/
// This function removes the entire <cache-base> directory.
// Returns the path of the cleared directory on success.
func ClearFileCache(ctx context.Context) (string, error) {
	databricksCacheDir, err := getCacheBaseDir(ctx)
	if err != nil {
		return "", err
	}

	// Remove the entire databricks cache directory (all versions)
	err = os.RemoveAll(databricksCacheDir)
	if err != nil {
		return "", err
	}

	return databricksCacheDir, nil
}
