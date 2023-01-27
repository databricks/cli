package bundle

import (
	"fmt"
	"os"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/folders"
)

const envBundleRoot = "BUNDLE_ROOT"

// getRootEnv returns the value of the `BUNDLE_ROOT` environment variable
// if it set and is a directory. If the environment variable is set but
// is not a directory, it returns an error. If the environment variable is
// not set, it returns an empty string.
func getRootEnv() (string, error) {
	path, ok := os.LookupEnv(envBundleRoot)
	if !ok {
		return "", nil
	}
	stat, err := os.Stat(path)
	if err == nil && !stat.IsDir() {
		err = fmt.Errorf("not a directory")
	}
	if err != nil {
		return "", fmt.Errorf(`invalid bundle root %s="%s": %w`, envBundleRoot, path, err)
	}
	return path, nil
}

// getRootWithTraversal returns the bundle root by traversing the filesystem
// from the working directory to the root looking for a configuration file.
func getRootWithTraversal() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	path, err := folders.FindDirWithLeaf(wd, config.FileName)
	if err != nil {
		return "", fmt.Errorf(`unable to locate bundle root: %s not found`, config.FileName)
	}
	return path, nil
}

// mustGetRoot returns a bundle root or an error if one cannot be found.
func mustGetRoot() (string, error) {
	path, err := getRootEnv()
	if path != "" || err != nil {
		return path, err
	}
	return getRootWithTraversal()
}

// tryGetRoot returns a bundle root or an empty string if one cannot be found.
func tryGetRoot() (string, error) {
	// Note: an invalid value in the environment variable is still an error.
	path, err := getRootEnv()
	if path != "" || err != nil {
		return path, err
	}
	// Note: traversal failing means the bundle root cannot be found.
	path, _ = getRootWithTraversal()
	return path, nil
}
