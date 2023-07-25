package python

// TODO: move this package into the libs

import (
	"context"
	"os"
	"path"
	"strings"

	"github.com/databricks/cli/libs/log"
)

func CleanupWheelFolder(dir string) {
	// there or not there - we don't care
	os.RemoveAll(path.Join(dir, "__pycache__"))
	os.RemoveAll(path.Join(dir, "build"))
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
	entries, err := f.ReadDir(0)
	if err != nil {
		log.Debugf(context.Background(), "read dir %s: %s", dir, err)
		// todo: log
		return nil
	}

	files := make([]string, 0)
	for _, child := range entries {
		if !strings.HasSuffix(child.Name(), suffix) {
			continue
		}
		files = append(files, path.Join(dir, child.Name()))
	}
	return files
}
