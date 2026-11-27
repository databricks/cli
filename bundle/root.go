package bundle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/folders"
)

func resolveConfigFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return config.FileNames.FindInPath(path)
	}
	if !info.Mode().IsRegular() || !config.IsConfigFile(path) {
		return "", errors.New("expected a .yml, .yaml, or .json configuration file")
	}
	return filepath.Abs(path)
}

// ConfigFileFromEnv returns the explicitly selected configuration file, or an empty path if unset.
func ConfigFileFromEnv(ctx context.Context) (string, error) {
	path, ok := env.Root(ctx)
	if !ok {
		return "", nil
	}
	file, err := resolveConfigFile(path)
	if err != nil {
		return "", fmt.Errorf("%s=%q: %w", env.RootVariable, filepath.ToSlash(path), err)
	}
	return file, nil
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
		Detail:   fmt.Sprintf("Run this command from a directory that contains a bundle, or set %s to the bundle directory or configuration file.\nAlternatively, create a new bundle with 'databricks bundle init'.", env.RootVariable),
	}}
}

func findConfigFile(ctx context.Context, required bool) (string, diag.Diagnostics) {
	file, err := ConfigFileFromEnv(ctx)
	if err != nil {
		return "", diag.Diagnostics{{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Invalid bundle root %s", err),
			Detail:   fmt.Sprintf("Set %s to a bundle directory or a .yml, .yaml, or .json configuration file.", env.RootVariable),
		}}
	}
	if file != "" {
		return file, nil
	}
	dir, diags := getRootWithTraversal()
	if dir == "" {
		if required {
			return "", diags
		}
		return "", nil
	}
	file, err = config.FileNames.FindInPath(dir)
	return file, diag.FromErr(err)
}
