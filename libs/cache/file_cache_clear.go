package cache

import (
	"context"
	"fmt"
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
		cmdio.LogString(ctx, fmt.Sprintf("No cache directory found at %s, nothing to clear", databricksCacheDir))
		return nil
	}

	// Remove the entire databricks cache directory
	err = os.RemoveAll(databricksCacheDir)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "Cache cleared successfully from "+databricksCacheDir)
	return nil
}
