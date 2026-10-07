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
	"github.com/databricks/cli/libs/structs/structvar"
)

func createGlobError(v structvar.View, p *structpath.PathNode, message string) diag.Diagnostic {
	// The pattern contained in v is an absolute path.
	// Make it relative to the value's location to make it more readable.
	source, _ := v.AsString()
	if l := v.Location(); l.File != "" {
		rel, err := filepath.Rel(filepath.Dir(l.File), source)
		if err == nil {
			source = rel
		}
	}

	return diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("%s: %s", source, message),
		Locations: []diag.Location{v.Location()},
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
	base := structpath.NewStringKeys(nil, "artifacts", e.name, "files")

	// Pattern to match the source key in the files sequence.
	pattern := structpath.NewPatternStringKey(structpath.NewPatternBracketStar(structpath.NewPatternStringKey(structpath.NewPatternStringKey(structpath.NewPatternStringKey(nil, "artifacts"), e.name), "files")), "source")

	artifact := b.Config.Artifacts[e.name]
	if artifact == nil {
		return nil
	}

	var diags diag.Diagnostics
	var output []config.ArtifactFile
	var sources [][]int
	var locations [][]diag.Location
	err := structvar.ForEach(b.Config.View(), pattern, func(np *structpath.PathNode, v structvar.View) error {
		if v.Kind() != structvar.KindString {
			return nil
		}

		index, _ := np.Parent().Index()
		source, _ := v.AsString()

		// Expand any glob reference in files source path
		matches, err := filepath.Glob(source)
		if err != nil {
			diags = diags.Append(createGlobError(v, np, err.Error()))

			// Drop this value from the list; this does not matter since we've raised an error anyway
			return nil
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
			return nil
		}

		if len(matches) == 0 {
			diags = diags.Append(createGlobError(v, np, "no matching files"))

			// Drop this value from the list; this does not matter since we've raised an error anyway
			return nil
		}

		for _, match := range matches {
			output = append(output, config.ArtifactFile{Source: match})
			sources = append(sources, []int{index})
			locations = append(locations, v.Locations())
		}

		return nil
	})
	if err != nil || diags.HasError() {
		return diags.Extend(diag.FromErr(err))
	}

	// Set the expanded globs back into the configuration.
	artifact.Files = output
	b.Config.UpdateSequence(base, sources)
	for i, locs := range locations {
		if locs != nil {
			b.Config.SetLocations(structpath.NewStringKey(structpath.NewIndex(base, i), "source"), locs)
		}
	}

	return diags
}
