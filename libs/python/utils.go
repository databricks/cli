package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/log"
)

func CleanupWheelFolder(dir string) {
	// there or not there - we don't care
	os.RemoveAll(filepath.Join(dir, "__pycache__"))
	os.RemoveAll(filepath.Join(dir, "build"))
	eggInfo := FindFilesWithSuffixInPath(dir, ".egg-info")
	if len(eggInfo) == 0 {
		return
	}
	for _, f := range eggInfo {
		os.RemoveAll(f)
	}
}

func FindFilesWithSuffixInPath(dir, suffix string) []string {
	f, err := os.Open(dir)
	if err != nil {
		log.Debugf(context.Background(), "open dir %s: %s", dir, err)
		return nil
	}
	defer f.Close()

	entries, err := f.ReadDir(0)
	if err != nil {
		log.Debugf(context.Background(), "read dir %s: %s", dir, err)
		// todo: log
		return nil
	}

	var files []string
	for _, child := range entries {
		if !strings.HasSuffix(child.Name(), suffix) {
			continue
		}
		files = append(files, filepath.Join(dir, child.Name()))
	}
	return files
}
