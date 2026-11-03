package bundle

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/folders"
)

// getRootEnv returns the value of the bundle root environment variable
// if it is set and points to a directory that contains a bundle configuration
// file. If the environment variable is set but does not point to such a
// directory, it returns diagnostics. If the environment variable is not set,
// it returns an empty string.
func getRootEnv(ctx context.Context) (string, diag.Diagnostics) {
	path, ok := env.Root(ctx)
	if !ok {
		return "", nil
	}
	stat, err := os.Stat(path)
	if err == nil && !stat.IsDir() {
		err = errors.New("not a directory")
	}
	if err == nil {
		_, err = config.FileNames.FindInPath(path)
	}
	if err != nil {
		return "", diag.Diagnostics{{
			Severity: diag.Error,
			Summary:  fmt.Sprintf(`Invalid bundle root %s="%s": %s`, env.RootVariable, path, err),
			Detail:   fmt.Sprintf("The %s environment variable must point to an existing directory that contains a bundle configuration file.", env.RootVariable),
		}}
	}
	return path, nil
}

// getRootWithTraversal returns the bundle root by traversing the filesystem
// from the working directory to the root looking for a configuration file.
func getRootWithTraversal() (string, diag.Diagnostics) {
	wd, err := os.Getwd()
	if err != nil {
		return "", diag.FromErr(err)
	}

	for _, file := range config.FileNames {
		path, err := folders.FindDirWithLeaf(wd, file)
		if err == nil {
			return path, nil
		}
	}

	return "", diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("Unable to locate the bundle root: %s not found.", config.FileNames[0]),
		Detail:   fmt.Sprintf("Run this command from a directory that contains a bundle, or set the %s environment variable to the bundle root directory.\nAlternatively, create a new bundle with 'databricks bundle init'.", env.RootVariable),
	}}
}

// mustGetRoot returns a bundle root or diagnostics if one cannot be found.
func mustGetRoot(ctx context.Context) (string, diag.Diagnostics) {
	path, diags := getRootEnv(ctx)
	if path != "" || diags.HasError() {
		return path, diags
	}
	return getRootWithTraversal()
}

// tryGetRoot returns a bundle root or an empty string if one cannot be found.
func tryGetRoot(ctx context.Context) (string, diag.Diagnostics) {
	// Note: an invalid value in the environment variable is still an error.
	path, diags := getRootEnv(ctx)
	if path != "" || diags.HasError() {
		return path, diags
	}
	// Note: traversal failing means the bundle root cannot be found.
	path, _ = getRootWithTraversal()
	return path, nil
}
