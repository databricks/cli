package mutator

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	pathlib "path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
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
//   - All relative paths are normalized to be relative to the bundle root.
//   - All relative paths are using forward slashes (including Windows paths).
func NormalizePaths() bundle.Mutator {
	return &normalizePaths{}
}

func (a normalizePaths) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Do not normalize job task paths if using git source
	gitSourcePaths := collectGitSourcePaths(b)

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return visitAllPaths(v, func(path dyn.Path, v dyn.Value) (dyn.Value, error) {
			for _, gitSourcePrefix := range gitSourcePaths {
				if path.HasPrefix(gitSourcePrefix) {
					return v, nil
				}
			}

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

func normalizePath(path string, location dyn.Location, bundleRootPath string) (string, error) {
	// If the path contains variables (like ${workspace.file_path}), skip normalization
	// Variables will be resolved later by other mutators
	if dynvar.ContainsVariableReference(path) {
		return path, nil
	}

	// Handle pip options with paths: only support specific known options
	// Check if this looks like a pip option
	if strings.HasPrefix(path, "-") {
		// Find the first space to identify the option
		spaceIndex := strings.Index(path, " ")
		if spaceIndex == -1 {
			// No space found, this might be a pip option without arguments
			// Let it pass through without normalization
			return path, nil
		}

		option := path[:spaceIndex+1] // Include the space

		// Check if this is a supported option
		supported := false
		for _, supportedOption := range libraries.PipOptionsAll {
			if option == supportedOption+" " {
				supported = true
				break
			}
		}

		if !supported {
			// This is an unknown pip option: fail with an error since we don't know if we should parse what follows as a relative path
			return "", fmt.Errorf("unsupported pip option '%s' in dependency '%s'. Supported options are: %s", option[:len(option)-1], path, strings.Join(libraries.PipOptionsAll, ", "))
		}

		// Handle the supported option
		optionPath := path[spaceIndex+1:]

		// Only normalize paths for options that actually take local path arguments
		if isPipOptionWithPath(path) {
			normalizedPath, err := normalizePath(strings.TrimSpace(optionPath), location, bundleRootPath)
			if err != nil {
				return "", err
			}
			return option + normalizedPath, nil
		} else {
			// For options that don't take local paths (like --extra-index-url, --trusted-host), return as-is
			return path, nil
		}
	}

	pathAsUrl, err := url.Parse(path)
	if err != nil {
		// If URL parsing fails, it might be a pip option with a URL that contains spaces
		// In that case, we should return the path as-is
		return path, nil
	}

	// if path has scheme, it's a full path and doesn't need to be relativized
	if pathAsUrl.Scheme != "" {
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

	dir, err := locationDirectory(location)
	if err != nil {
		return "", fmt.Errorf("unable to determine directory for a value at %s: %w", path, err)
	}

	relDir, err := filepath.Rel(bundleRootPath, dir)
	if err != nil {
		return "", err
	}

	result := filepath.ToSlash(filepath.Join(relDir, path))
	return result, nil
}

func locationDirectory(l dyn.Location) (string, error) {
	if l.File == "" {
		return "", errors.New("no file in location")
	}

	return filepath.Dir(l.File), nil
}

// visitAllPaths visits all paths in bundle configuration without skip logic
func visitAllPaths(root dyn.Value, fn func(path dyn.Path, value dyn.Value) (dyn.Value, error)) (dyn.Value, error) {
	// Define patterns for all paths that need normalization
	patterns := []dyn.Pattern{
		// Job paths
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("notebook_task"), dyn.Key("notebook_path")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("spark_python_task"), dyn.Key("python_file")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("dbt_task"), dyn.Key("project_directory")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("sql_task"), dyn.Key("file"), dyn.Key("path")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("requirements")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("notebook"), dyn.Key("path")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("file"), dyn.Key("path")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("glob"), dyn.Key("include")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("whl")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("jar")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("requirements")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("notebook"), dyn.Key("path")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("file"), dyn.Key("path")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("glob"), dyn.Key("include")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("whl")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("jar")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("environments"), dyn.AnyIndex(), dyn.Key("spec"), dyn.Key("dependencies"), dyn.AnyIndex()),

		// Pipeline paths
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("pipelines"), dyn.AnyKey(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("notebook"), dyn.Key("path")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("pipelines"), dyn.AnyKey(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("file"), dyn.Key("path")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("pipelines"), dyn.AnyKey(), dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("glob"), dyn.Key("include")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("pipelines"), dyn.AnyKey(), dyn.Key("root_path")),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("pipelines"), dyn.AnyKey(), dyn.Key("environment"), dyn.Key("dependencies"), dyn.AnyIndex()),

		// App paths
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("apps"), dyn.AnyKey(), dyn.Key("source_code_path")),

		// Artifact paths
		dyn.NewPattern(dyn.Key("artifacts"), dyn.AnyKey(), dyn.Key("path")),

		// Dashboard paths
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("dashboards"), dyn.AnyKey(), dyn.Key("file_path")),
	}

	newRoot := root
	for _, pattern := range patterns {
		updatedRoot, err := dyn.MapByPattern(newRoot, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// Skip normalization for environment dependencies that are not local libraries
			// but still process pip options with paths
			if isEnvironmentDependencyPattern(pattern) {
				value, ok := v.AsString()
				if !ok {
					return v, nil
				}
				// Skip if it's not a local library AND not a pip option (regardless of whether it has a path)
				// Also skip if it contains pip flags but is not a valid pip option format
				if !libraries.IsLibraryLocal(value) && !libraries.ContainsPipFlag(value) {
					return v, nil
				}
				// Skip invalid pip option formats (like "foobar --extra-index-url ...")
				if libraries.ContainsPipFlag(value) && !strings.HasPrefix(value, "-") {
					return v, nil
				}
			}
			return fn(p, v)
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
		newRoot = updatedRoot
	}

	return newRoot, nil
}

// isEnvironmentDependencyPattern checks if a pattern matches environment dependencies
func isEnvironmentDependencyPattern(pattern dyn.Pattern) bool {
	patternStr := pattern.String()

	// Check if this is a job environment dependency pattern
	// Pattern: resources.jobs.*.environments[*].spec.dependencies[*]
	jobEnvPatternStr := "resources.jobs.*.environments[*].spec.dependencies[*]"

	// Check if this is a pipeline environment dependency pattern
	// Pattern: resources.pipelines.*.environment.dependencies[*]
	pipelineEnvPatternStr := "resources.pipelines.*.environment.dependencies[*]"

	return patternStr == jobEnvPatternStr || patternStr == pipelineEnvPatternStr
}

// isPipOptionWithPath checks if a string is a pip option that has a path argument
func isPipOptionWithPath(value string) bool {
	// Only check options that actually take local path arguments
	for _, option := range libraries.PipOptionsWithPaths {
		if strings.HasPrefix(value, option+" ") {
			return true
		}
	}
	return false
}
