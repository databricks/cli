package project

import (
	"fmt"
	"os"

	"github.com/databricks/bricks/folders"
)

const bricksRoot = "BRICKS_ROOT"

// getRoot returns the project root.
// If the `BRICKS_ROOT` environment variable is set, we assume its value
// to be a valid project root. Otherwise we try to find it by traversing
// the path and looking for a project configuration file.
func getRoot() (string, error) {
	path, ok := os.LookupEnv(bricksRoot)
	if ok {
		stat, err := os.Stat(path)
		if err == nil && !stat.IsDir() {
			err = fmt.Errorf("not a directory")
		}
		if err != nil {
			return "", fmt.Errorf(`invalid project root %s="%s": %w`, bricksRoot, path, err)
		}
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		path, err = folders.FindDirWithLeaf(wd, ConfigFile)
		if err != nil {
			return "", fmt.Errorf(`unable to locate project root`)
		}
	}

	return path, nil
}
