package artifacts

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/patchwheel"
	"github.com/databricks/cli/libs/structs/structpath"
)

func createGlobError(source string, locs []diag.Location, p *structpath.PathNode, message string) diag.Diagnostic {
	var loc diag.Location
	if len(locs) > 0 {
		loc = locs[0]
	}
	// The pattern is an absolute path.
	// Make it relative to the value's location to make it more readable.
	if loc.File != "" {
		rel, err := filepath.Rel(filepath.Dir(loc.File), source)
		if err == nil {
			source = rel
		}
	}

	return diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("%s: %s", source, message),
		Locations: []diag.Location{loc},
		Paths:     []*structpath.PathNode{p},
	}
}

type expandGlobs struct {
	name string
}

func (e expandGlobs) Name() string {
	return "expandGlobs"
}

func (e expandGlobs) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Base path for this mutator.
	// This path is set with the list of expanded globs when done.
	base := structpath.NewPath(nil, "artifacts", e.name, "files")

	artifact := b.Config.Artifacts[e.name]
	if artifact == nil {
		return nil
	}

	var diags diag.Diagnostics
	var output []config.ArtifactFile
	var sources [][]int
	var locations [][]diag.Location
	for index, f := range artifact.Files {
		np := structpath.NewPath(base, index, "source")
		locs := b.Config.GetLocationsOfExact(np)
		// source has no omitempty: an explicit empty value is present too.
		if f.Source == "" && len(locs) == 0 {
			continue
		}
		source := f.Source

		// Expand any glob reference in files source path
		matches, err := filepath.Glob(source)
		if err != nil {
			diags = diags.Append(createGlobError(source, locs, np, err.Error()))

			// Drop this value from the list; this does not matter since we've raised an error anyway
			continue
		}

		// Note, we're applying this for all artifact types, not just "whl".
		// Rationale:
		//  1. type is optional
		//  2. if you have wheels in other artifact type, maybe you still want the filter logic? impossible to say.
		matches = patchwheel.FilterLatestWheels(ctx, matches)

		if len(matches) == 1 && matches[0] == source {
			// No glob expansion was performed.
			// Keep node unchanged. We need to ensure that "patched" field remains and not wiped out by code below.
			output = append(output, artifact.Files[index])
			sources = append(sources, []int{index})
			locations = append(locations, nil)
			continue
		}

		if len(matches) == 0 {
			diags = diags.Append(createGlobError(source, locs, np, "no matching files"))

			// Drop this value from the list; this does not matter since we've raised an error anyway
			continue
		}

		for _, match := range matches {
			output = append(output, config.ArtifactFile{Source: match})
			sources = append(sources, []int{index})
			locations = append(locations, locs)
		}
	}
	if diags.HasError() {
		return diags
	}

	// Set the expanded globs back into the configuration.
	artifact.Files = output
	b.Config.UpdateSequence(base, sources)
	for i, locs := range locations {
		if locs != nil {
			b.Config.SetLocations(structpath.NewPath(base, i, "source"), locs)
		}
	}

	return diags
}
