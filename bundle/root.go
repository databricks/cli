package bundle

import (
	"fmt"
	"os"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/folders"
)

const envBundleRoot = "BUNDLE_ROOT"

// getRoot returns the bundle root.
// If the `BUNDLE_ROOT` environment variable is set, we assume its value
// to be a valid bundle root. Otherwise we try to find it by traversing
// the path and looking for a project configuration file.
func getRoot() (string, error) {
	path, ok := os.LookupEnv(envBundleRoot)
	if ok {
		stat, err := os.Stat(path)
		if err == nil && !stat.IsDir() {
			err = fmt.Errorf("not a directory")
		}
		if err != nil {
			return "", fmt.Errorf(`invalid bundle root %s="%s": %w`, envBundleRoot, path, err)
		}
		return path, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	path, err = folders.FindDirWithLeaf(wd, config.FileName)
	if err != nil {
		return "", fmt.Errorf(`unable to locate bundle root`)
	}
	return path, nil
}
