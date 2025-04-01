package mutator

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator/paths"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type normalizePaths struct{}

func (a normalizePaths) Name() string {
	return "NormalizePaths"
}

// NormalizePaths is applied to resources declared in YAML to translate
// paths that are relative to YAML file locations to paths that are relative
// to the bundle root.
//
// Pre-conditions:
//   - Resources and artifacts have resolved all variables where relative paths are
//     used (including complex variables).
//   - Each path value is a string and has a location. Locations are absolute paths.
//
// Post-conditions:
//   - All paths are normalized to be relative to the bundle root.
//   - All paths are cleaned.
func NormalizePaths() bundle.Mutator {
	return &normalizePaths{}
}

func (a normalizePaths) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return paths.VisitPaths(v, func(path dyn.Path, kind paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
			value, ok := v.AsString()
			if !ok {
				return dyn.InvalidValue, fmt.Errorf("value at %s is not a string", path.String())
			}

			newValue, err := normalizePath(value, v.Location(), b.BundleRootPath)
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

func normalizePath(path string, location dyn.Location, bundleRootPath string) (string, error) {
	pathAsUrl, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	// if path has scheme, it's a full path and doesn't need to be relativized
	if pathAsUrl.Scheme != "" {
		return path, nil
	}

	// absolute paths don't need to be relativized
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	dir, err := locationDirectory(location)
	if err != nil {
		return "", fmt.Errorf("unable to determine directory for a value at %s: %w", path, err)
	}

	relDir, err := filepath.Rel(bundleRootPath, dir)
	if err != nil {
		return "", err
	}

	return filepath.Join(relDir, path), nil
}

func locationDirectory(l dyn.Location) (string, error) {
	if l.File == "" {
		return "", errors.New("no file in location")
	}

	return filepath.Dir(l.File), nil
}
