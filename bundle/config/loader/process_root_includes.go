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

	return bundle.Apply(ctx, b, bundle.Seq(out...))
}
