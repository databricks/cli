package mutator

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	pathlib "path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator/paths"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type normalizePaths struct{}

func (a normalizePaths) Name() string {
	return "NormalizePaths"
}

// NormalizePaths is applied to resources declared in YAML to translate
// paths that are relative to YAML file locations to paths that are relative
// to the bundle root. Paths in values from the variable overrides file
// (.databricks/bundle/<target>/variable-overrides.json) are already relative
// to the bundle root.
//
// Pre-conditions:
//   - Resources and artifacts have resolved all variables where relative paths are
//     used (including complex variables).
//   - Each path value is a string and has a location. Locations are absolute paths.
//
// Post-conditions:
//   - All relative paths are normalized to be relative to the bundle root.
//   - All relative paths are using forward slashes (including Windows paths).
func NormalizePaths() bundle.Mutator {
	return &normalizePaths{}
}

func (a normalizePaths) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Do not normalize job task paths if using git source
	gitSourcePaths := collectGitSourcePaths(b)
	overridesFile := filepath.Join(b.BundleRootPath, getDefaultVariableFilePath(b.Config.Bundle.Target))

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return paths.VisitPaths(v, func(path dyn.Path, kind paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
			if slices.ContainsFunc(gitSourcePaths, path.HasPrefix) {
				return v, nil
			}

			value, ok := v.AsString()
			if !ok {
				return dyn.InvalidValue, fmt.Errorf("value at %s is not a string", path.String())
			}

			dir, dirErr := baseDirectory(v.Location(), overridesFile, b.BundleRootPath)
			newValue, err := normalizePath(value, dir, dirErr, b.BundleRootPath)
			if err != nil {
				return dyn.InvalidValue, err
			}

			return dyn.NewValue(newValue, v.Locations()), nil
		})
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to normalize paths: %w", err))
	}

	return diag.FromErr(err)
}

func collectGitSourcePaths(b *bundle.Bundle) []dyn.Path {
	var jobs []dyn.Path

	for name, job := range b.Config.Resources.Jobs {
		if job == nil {
			continue
		}
		if job.GitSource != nil {
			jobs = append(jobs, dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"), dyn.Key(name)))
		}
	}

	return jobs
}

// baseDirectory returns the directory that relative paths in a value at the given location resolve against.
func baseDirectory(location diag.Location, overridesFile, bundleRootPath string) (string, error) {
	// The variable overrides file lives under .databricks/bundle/<target>/, but users write
	// its paths relative to the bundle root.
	if location.File == overridesFile {
		return bundleRootPath, nil
	}
	return locationDirectory(location)
}

// normalizePath makes a relative path relative to the bundle root. dir is the directory it is relative to now;
// dirErr is the reason dir is unknown, reported only if the path needs it.
func normalizePath(path, dir string, dirErr error, bundleRootPath string) (string, error) {
	// Handle local file paths used inside pip flags
	for _, flag := range libraries.PipFlagsWithLocalPaths {
		reqPath, ok := strings.CutPrefix(path, flag+" ")
		if ok {
			// Normalize the path part
			reqPath = strings.TrimSpace(reqPath)
			normalizedPath, err := normalizePath(reqPath, dir, dirErr, bundleRootPath)
			if err != nil {
				return "", err
			}

			// Reconstruct the path with -r flag
			return flag + " " + normalizedPath, nil
		}
	}

	// if path has scheme, it's a full path and doesn't need to be relativized.
	// A path that is not a valid URL, such as "New Notebook 12:17:43.ipynb", is a local path.
	if pathAsUrl, err := url.Parse(path); err == nil && pathAsUrl.Scheme != "" {
		return path, nil
	}

	// absolute paths don't need to be relativized, keep them as-is
	if filepath.IsAbs(path) {
		return path, nil
	}

	// if we use Windows, a path can be a POSIX path, and should remain as-is
	if pathlib.IsAbs(path) {
		return path, nil
	}

	if dirErr != nil {
		return "", fmt.Errorf("unable to determine directory for a value at %s: %w", path, dirErr)
	}

	relDir, err := filepath.Rel(bundleRootPath, dir)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(filepath.Join(relDir, path)), nil
}

func locationDirectory(l diag.Location) (string, error) {
	if l.File == "" {
		return "", errors.New("no file in location")
	}

	return filepath.Dir(l.File), nil
}
