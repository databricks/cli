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
	overridesFile := variableOverridesFilePath(b)

	var diags diag.Diagnostics
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return paths.VisitPaths(v, func(path dyn.Path, kind paths.TranslateMode, v dyn.Value) (dyn.Value, error) {
			if slices.ContainsFunc(gitSourcePaths, path.HasPrefix) {
				return v, nil
			}

			value, ok := v.AsString()
			if !ok {
				return dyn.InvalidValue, fmt.Errorf("value at %s is not a string", path.String())
			}

			if v.Location().File == overridesFile {
				if suggestion, ok := legacyOverridesPathSuggestion(value, overridesFile, b.BundleRootPath); ok {
					diags = diags.Append(legacyOverridesPathWarning(value, suggestion, overridesFile, b.BundleRootPath, v, path))
				}
			}

			newValue, err := normalizePath(value, v.Location(), overridesFile, b.BundleRootPath)
			if err != nil {
				return dyn.InvalidValue, err
			}

			return dyn.NewValue(newValue, v.Locations()), nil
		})
	})
	if err != nil {
		return diags.Extend(diag.FromErr(fmt.Errorf("failed to normalize paths: %w", err)))
	}

	return diags
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

func normalizePath(path string, location diag.Location, overridesFile, bundleRootPath string) (string, error) {
	// Handle local file paths used inside pip flags
	for _, flag := range libraries.PipFlagsWithLocalPaths {
		reqPath, ok := strings.CutPrefix(path, flag+" ")
		if ok {
			// Normalize the path part
			reqPath = strings.TrimSpace(reqPath)
			normalizedPath, err := normalizePath(reqPath, location, overridesFile, bundleRootPath)
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

	dir, err := baseDirectory(location, overridesFile, bundleRootPath)
	if err != nil {
		return "", fmt.Errorf("unable to determine directory for a value at %s: %w", path, err)
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

// legacyOverridesPathSuggestion detects a path in the variable overrides file that is written relative to the
// file's own directory, which is how such paths resolved before they became relative to the bundle root: it
// points outside the bundle root, but read from the file's directory it points into the bundle. It returns the
// value rewritten relative to the bundle root. It only feeds a warning, and a path that exists relative to the
// bundle root (a file outside the bundle that the user means) is not flagged.
func legacyOverridesPathSuggestion(value, overridesFile, bundleRootPath string) (string, bool) {
	for _, flag := range libraries.PipFlagsWithLocalPaths {
		if rest, ok := strings.CutPrefix(value, flag+" "); ok {
			suggestion, ok := legacyOverridesPathSuggestion(strings.TrimSpace(rest), overridesFile, bundleRootPath)
			return flag + " " + suggestion, ok
		}
	}

	if u, err := url.Parse(value); (err == nil && u.Scheme != "") || filepath.IsAbs(value) || pathlib.IsAbs(value) {
		return "", false
	}

	newPath := filepath.Join(bundleRootPath, value)
	if !escapesDir(bundleRootPath, newPath) || pathExists(newPath) {
		return "", false
	}

	legacyPath := filepath.Join(filepath.Dir(overridesFile), value)
	rel, err := filepath.Rel(bundleRootPath, legacyPath)
	if err != nil || escapesDir(bundleRootPath, legacyPath) || !escapesDir(filepath.Join(bundleRootPath, ".databricks"), legacyPath) {
		return "", false
	}
	return "./" + filepath.ToSlash(rel), true
}

// escapesDir reports whether path is outside dir.
func escapesDir(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	return err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func legacyOverridesPathWarning(value, suggestion, overridesFile, bundleRootPath string, v dyn.Value, path dyn.Path) diag.Diagnostic {
	relOverridesFile, err := filepath.Rel(bundleRootPath, overridesFile)
	if err != nil {
		relOverridesFile = overridesFile
	}
	return diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   fmt.Sprintf("path %s in %s points outside the bundle root", value, filepath.ToSlash(relOverridesFile)),
		Detail:    fmt.Sprintf("Relative paths in complex variable values in this file are relative to the bundle root, not to the file's directory. If it refers to a file in the bundle, write %s instead.", suggestion),
		Locations: v.Locations(),
		Paths:     dyn.ToStructPaths(path),
	}
}

// pathExists reports whether path, which may be a glob pattern, matches an existing file.
func pathExists(path string) bool {
	matches, err := filepath.Glob(path)
	return err == nil && len(matches) > 0
}
