package loader

import (
	"context"
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
		for _, match := range matches {
			rel, err := filepath.Rel(b.BundleRootPath, match)
			if err != nil {
				return diag.FromErr(err)
			}
			if _, ok := seen[rel]; ok {
				continue
			}
			seen[rel] = true
			if filepath.Ext(rel) != ".yaml" && filepath.Ext(rel) != ".yml" {
				return diag.Errorf("file %s included in 'include' section but only YAML files are supported. If you want to explicitly include files to sync, use 'sync.include' configuration section", rel)
			}
			includes = append(includes, rel)
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
