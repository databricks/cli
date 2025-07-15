package loader

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
)

type processRootIncludes struct{}

// ProcessRootIncludes expands the patterns in the configuration's include list
// into a list of mutators for each matching file.
func ProcessRootIncludes() bundle.Mutator {
	return &processRootIncludes{}
}

func (m *processRootIncludes) Name() string {
	return "ProcessRootIncludes"
}

// hasGlobCharacters checks if a path contains any glob characters.
func hasGlobCharacters(path string) (string, bool) {
	// List of glob characters supported by the filepath package in [filepath.Match]
	globCharacters := []string{"*", "?", "[", "]", "^"}
	for _, char := range globCharacters {
		if strings.Contains(path, char) {
			return char, true
		}
	}
	return "", false
}

func (m *processRootIncludes) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var out []bundle.Mutator

	// Map with files we've already seen to avoid loading them twice.
	seen := map[string]bool{}

	for _, file := range config.FileNames {
		seen[file] = true
	}

	// Maintain list of files in order of files being loaded.
	// This is stored in the bundle configuration for observability.
	var files []string
	var diags diag.Diagnostics

	// We error on glob characters in the bundle root path since they are
	// parsed by [filepath.Glob] as being glob patterns and thus can cause
	// unexpected behavior.
	//
	// The standard library does not support globbing relative paths from a specified
	// base directory. To support this, we can either:
	// 1. Change CWD to the bundle root path before calling [filepath.Glob]
	// 2. Implement our own custom globbing function. We can use [filepath.Match] to do so.
	if char, ok := hasGlobCharacters(b.BundleRootPath); ok {
		diags = diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Bundle root path contains glob pattern characters",
			Detail:   fmt.Sprintf("The path to the bundle root %s contains glob pattern character %q. Please remove the character from this path to use bundle commands.", b.BundleRootPath, char),
		})

		return diags
	}

	// For each glob, find all files to load.
	// Ordering of the list of globs is maintained in the output.
	// For matches that appear in multiple globs, only the first is kept.
	for _, entry := range b.Config.Include {
		// Include paths must be relative.
		if filepath.IsAbs(entry) {
			return diag.Errorf("%s: includes must be relative paths", entry)
		}

		// Anchor includes to the bundle root path.
		matches, err := filepath.Glob(filepath.Join(b.BundleRootPath, entry))
		if err != nil {
			return diag.FromErr(err)
		}

		// If the entry is not a glob pattern and no matches found,
		// return an error because the file defined is not found
		if len(matches) == 0 && !strings.ContainsAny(entry, "*?[") {
			return diag.Errorf("%s defined in 'include' section does not match any files", entry)
		}

		// Filter matches to ones we haven't seen yet.
		var includes []string
		for i, match := range matches {
			rel, err := filepath.Rel(b.BundleRootPath, match)
			if err != nil {
				return diag.FromErr(err)
			}
			if _, ok := seen[rel]; ok {
				continue
			}
			seen[rel] = true
			if filepath.Ext(rel) != ".yaml" && filepath.Ext(rel) != ".yml" && filepath.Ext(rel) != ".json" {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "Files in the 'include' configuration section must be YAML or JSON files.",
					Detail:    fmt.Sprintf("The file %s in the 'include' configuration section is not a YAML or JSON file, and only such files are supported. To include files to sync, specify them in the 'sync.include' configuration section instead.", rel),
					Locations: b.Config.GetLocations(fmt.Sprintf("include[%d]", i)),
				})
				continue
			}
			includes = append(includes, rel)
		}

		if len(diags) > 0 {
			return diags
		}

		// Add matches to list of mutators to return.
		slices.Sort(includes)
		files = append(files, includes...)
		for _, include := range includes {
			out = append(out, ProcessInclude(filepath.Join(b.BundleRootPath, include), include))
		}
	}

	// Swap out the original includes list with the expanded globs.
	b.Config.Include = files

	// Track number of bundle YAML (or JSON) files in the configuration. The +1 is there
	// to account for the root databricks.yaml file.
	b.Metrics.ConfigurationFileCount = int64(len(files)) + 1

	bundle.ApplySeqContext(ctx, b, out...)
	return nil
}
