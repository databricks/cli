package resourcemutator

import (
	"context"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/libs/structs/structpath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/patchwheel"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type expandPipelineGlobPaths struct{}

func ExpandPipelineGlobPaths() bundle.Mutator {
	return &expandPipelineGlobPaths{}
}

// expandLibrary returns the libraries that replace lib: one per match of its glob path.
func (m *expandPipelineGlobPaths) expandLibrary(ctx context.Context, dir string, lib pipelines.PipelineLibrary) ([]pipelines.PipelineLibrary, error) {
	// Probe for the path field in the library.
	var path string
	var withPath func(path string) pipelines.PipelineLibrary
	switch {
	case lib.Notebook != nil && lib.Notebook.Path != "":
		path = lib.Notebook.Path
		withPath = func(path string) pipelines.PipelineLibrary {
			nl, notebook := lib, *lib.Notebook
			notebook.Path = path
			nl.Notebook = &notebook
			return nl
		}
	case lib.File != nil && lib.File.Path != "":
		path = lib.File.Path
		withPath = func(path string) pipelines.PipelineLibrary {
			nl, file := lib, *lib.File
			file.Path = path
			nl.File = &file
			return nl
		}
	default:
		// Neither of the library paths were found (or the path is empty). This is likely
		// an invalid node, but it isn't this mutator's job to enforce that. Return the original value.
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
		expanded = append(expanded, withPath(filepath.ToSlash(m)))
	}

	return expanded, nil
}

func (m *expandPipelineGlobPaths) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Visit each pipeline's "libraries" field and expand any glob patterns.
	for name, pipeline := range b.Config.Resources.Pipelines {
		if len(pipeline.Libraries) == 0 {
			continue
		}

		var expanded []pipelines.PipelineLibrary
		// origin[i] is the index of the library that expanded[i] comes from.
		var origin []int
		for i, lib := range pipeline.Libraries {
			libs, err := m.expandLibrary(ctx, b.BundleRootPath, lib)
			if err != nil {
				return diag.FromErr(err)
			}
			expanded = append(expanded, libs...)
			for range libs {
				origin = append(origin, i)
			}
		}
		pipeline.Libraries = expanded

		// An expanded library has the locations of the library it comes from, e.g. the
		// location of the glob. A library never moves to a lower index, so going from the
		// last one down copies from locations that are not overwritten yet.
		libraries := structpath.NewPath(nil, "resources", "pipelines", name, "libraries")
		for i, o := range slices.Backward(origin) {
			if o != i {
				b.Config.CopyMetadataAt(structpath.NewIndex(libraries, o), structpath.NewIndex(libraries, i))
			}
		}
	}

	return nil
}

func (*expandPipelineGlobPaths) Name() string {
	return "ExpandPipelineGlobPaths"
}
