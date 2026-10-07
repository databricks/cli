package resourcemutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/patchwheel"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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
	p := structpath.MustParsePattern("resources.pipelines.*.libraries")

	// Visit each pipeline's "libraries" field and expand any glob patterns.
	err := structvar.ForEach(b.Config.View(), p, func(path *structpath.PathNode, value structvar.View) error {
		if value.Kind() != structvar.KindSequence {
			return fmt.Errorf("expected sequence, got %s", value.Kind())
		}

		pipeline := b.Config.Resources.Pipelines[path.KeyAt(2)]
		if len(pipeline.Libraries) == 0 {
			return nil
		}

		var expanded []pipelines.PipelineLibrary
		var sources [][]int
		for i, lib := range pipeline.Libraries {
			libs, err := m.expandLibrary(ctx, b.BundleRootPath, lib)
			if err != nil {
				return err
			}

			expanded = append(expanded, libs...)
			for range libs {
				sources = append(sources, []int{i})
			}
		}

		pipeline.Libraries = expanded
		b.Config.UpdateSequence(path, sources)
		return nil
	})

	return diag.FromErr(err)
}

func (*expandPipelineGlobPaths) Name() string {
	return "ExpandPipelineGlobPaths"
}
