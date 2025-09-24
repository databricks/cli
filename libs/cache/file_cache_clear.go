package cache

import (
	"context"
	"os"

	"github.com/databricks/cli/libs/cmdio"
)

func ClearFileCache(ctx context.Context) error {
	databricksCacheDir, err := getCacheBaseDir()
	if err != nil {
		return err
	}

	// Check if the cache directory exists
	if _, err := os.Stat(databricksCacheDir); os.IsNotExist(err) {
		cmdio.LogString(ctx, "No cache directory found, nothing to clear")
		return nil
	}

	// Remove the entire databricks cache directory
	err = os.RemoveAll(databricksCacheDir)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "Cache cleared successfully")
	return nil
}
