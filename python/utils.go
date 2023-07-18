package python

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
	eggInfo := FindFileWithSuffixInPath(dir, ".egg-info")
	if eggInfo == "" {
		return
	}
	os.RemoveAll(eggInfo)
}

func FindFileWithSuffixInPath(dir, suffix string) string {
	f, err := os.Open(dir)
	if err != nil {
		log.Debugf(context.Background(), "open dir %s: %s", dir, err)
		return ""
	}
	entries, err := f.ReadDir(0)
	if err != nil {
		log.Debugf(context.Background(), "read dir %s: %s", dir, err)
		// todo: log
		return ""
	}
	for _, child := range entries {
		if !strings.HasSuffix(child.Name(), suffix) {
			continue
		}
		return path.Join(dir, child.Name())
	}
	return ""
}
