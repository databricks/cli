package mutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type expandPipelineGlobPaths struct{}

func ExpandPipelineGlobPaths() bundle.Mutator {
	return &expandPipelineGlobPaths{}
}

func (m *expandPipelineGlobPaths) Apply(_ context.Context, b *bundle.Bundle) error {
	for key, pipeline := range b.Config.Resources.Pipelines {
		expandedLibraries := make([]pipelines.PipelineLibrary, 0)
		for i := 0; i < len(pipeline.Libraries); i++ {
			dir, err := pipeline.ConfigFileDirectory()
			if err != nil {
				return fmt.Errorf("unable to determine directory for pipeline %s: %w", key, err)
			}

			library := &pipeline.Libraries[i]
			path := getGlobPatternToExpand(library)
			if path == "" || !libraries.IsLocalPath(path) {
				expandedLibraries = append(expandedLibraries, *library)
				continue
			}

			matches, err := filepath.Glob(filepath.Join(dir, path))
			if err != nil {
				return err
			}

			for _, match := range matches {
				m, err := filepath.Rel(dir, match)
				if err != nil {
					return err
				}
				expandedLibraries = append(expandedLibraries, cloneWithPath(library, m))
			}
		}
		pipeline.Libraries = expandedLibraries
	}

	return nil
}

func getGlobPatternToExpand(library *pipelines.PipelineLibrary) string {
	if library.File != nil {
		return library.File.Path
	}

	if library.Notebook != nil {
		return library.Notebook.Path
	}

	return ""
}

func cloneWithPath(library *pipelines.PipelineLibrary, path string) pipelines.PipelineLibrary {
	if library.File != nil {
		return pipelines.PipelineLibrary{
			File: &pipelines.FileLibrary{
				Path: path,
			},
		}
	}

	if library.Notebook != nil {
		return pipelines.PipelineLibrary{
			Notebook: &pipelines.NotebookLibrary{
				Path: path,
			},
		}
	}

	return pipelines.PipelineLibrary{}
}

func (*expandPipelineGlobPaths) Name() string {
	return "ExpandPipelineGlobPaths"
}
