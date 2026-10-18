package resourcemutator

import (
	"context"
	"errors"
	"maps"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/patchwheel"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type expandPipelineGlobPaths struct{}

func ExpandPipelineGlobPaths() bundle.Mutator {
	return &expandPipelineGlobPaths{}
}

// expandLibrary returns the libraries lib is replaced with.
func (m *expandPipelineGlobPaths) expandLibrary(ctx context.Context, dir string, lib pipelines.PipelineLibrary) ([]pipelines.PipelineLibrary, error) {
	// Probe for the path field in the library.
	var path string
	switch {
	case lib.Notebook != nil && lib.Notebook.Path != "":
		path = lib.Notebook.Path
	case lib.File != nil && lib.File.Path != "":
		path = lib.File.Path
	default:
		// Neither of the library paths were found (or the path is empty). This is likely an invalid node,
		// but it isn't this mutator's job to enforce that. Return the original value.
		return []pipelines.PipelineLibrary{lib}, nil
	}

	// If the path is not a local path, return the original value.
	if !libraries.IsLocalPath(path) {
		return []pipelines.PipelineLibrary{lib}, nil
	}

	matches, err := filepath.Glob(filepath.Join(dir, path))
	if err != nil {
		return nil, err
	}

	// If there are no matches, return the original value.
	if len(matches) == 0 {
		return []pipelines.PipelineLibrary{lib}, nil
	}

	matches = patchwheel.FilterLatestWheels(ctx, matches)

	// Emit a new value for each match.
	var expanded []pipelines.PipelineLibrary
	for _, match := range matches {
		m, err := filepath.Rel(dir, match)
		if err != nil {
			return nil, err
		}
		nl := lib
		if lib.Notebook != nil {
			notebook := *lib.Notebook
			notebook.Path = filepath.ToSlash(m)
			nl.Notebook = &notebook
		} else {
			file := *lib.File
			file.Path = filepath.ToSlash(m)
			nl.File = &file
		}
		expanded = append(expanded, nl)
	}

	return expanded, nil
}

func (m *expandPipelineGlobPaths) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	pls := b.Config.Resources.Pipelines
	for _, name := range slices.Sorted(maps.Keys(pls)) {
		pipeline := pls[name]
		if pipeline == nil {
			continue
		}
		path := structpath.NewPath(nil, "resources", "pipelines", name, "libraries")
		if b.Config.IsReferenceAt(path) {
			return diag.FromErr(errors.New("expected sequence, got string"))
		}
		if len(pipeline.Libraries) == 0 {
			continue
		}

		// Expand any glob patterns in the pipeline's libraries.
		var expanded []pipelines.PipelineLibrary
		var sources [][]int
		for i, lib := range pipeline.Libraries {
			libs, err := m.expandLibrary(ctx, b.BundleRootPath, lib)
			if err != nil {
				return diag.FromErr(err)
			}

			expanded = append(expanded, libs...)
			for range libs {
				sources = append(sources, []int{i})
			}
		}

		pipeline.Libraries = expanded
		b.Config.UpdateSequence(path, sources)
	}

	return nil
}

func (*expandPipelineGlobPaths) Name() string {
	return "ExpandPipelineGlobPaths"
}
