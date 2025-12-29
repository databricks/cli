package cache

import (
	"context"
	"os"
)

// ClearFileCache removes all cached files from the Databricks cache directory.
// This clears the cache for ALL CLI versions, not just the current version.
//
// The cache directory structure is:
//
//	~/.cache/databricks/           (or %LOCALAPPDATA%\databricks\ on Windows)
//	└── <cli-version>/
//	    └── <component>/
//	        ├── <sha256-hash>.json
//	        └── ...
//
// This function removes the entire databricks cache directory (all versions and components).
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
