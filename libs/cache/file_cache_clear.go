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

	// Remove the entire databricks cache directory
	err = os.RemoveAll(databricksCacheDir)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "Cache cleared successfully from "+databricksCacheDir)
	return nil
}
