package fileutil

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
)

const (
	SuffixOriginalBak = ".original.bak"
	SuffixLatestBak   = ".latest.bak"
)

// BackupFile saves data to path+".original.bak" on the first call, and
// path+".latest.bak" on subsequent calls. Skips if data is empty.
func BackupFile(ctx context.Context, path string, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	originalBak := path + SuffixOriginalBak
	latestBak := path + SuffixLatestBak
	var bakPath string
	if _, err := os.Stat(originalBak); os.IsNotExist(err) {
		bakPath = originalBak
	} else {
		bakPath = latestBak
	}
	if err := os.WriteFile(bakPath, data, 0o600); err != nil {
		return err
	}
	log.Infof(ctx, "Backed up %s to %s", filepath.ToSlash(path), filepath.ToSlash(bakPath))
	return nil
}
