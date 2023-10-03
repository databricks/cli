package mutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type expandGlobPaths struct{}

func ExpandGlobPaths() bundle.Mutator {
	return &expandGlobPaths{}
}

func (m *expandGlobPaths) Apply(_ context.Context, b *bundle.Bundle) error {
	for key, pipeline := range b.Config.Resources.Pipelines {
		expandedLibraries := make([]pipelines.PipelineLibrary, 0)
		for i := 0; i < len(pipeline.Libraries); i++ {
			dir, err := pipeline.ConfigFileDirectory()
			if err != nil {
				return fmt.Errorf("unable to determine directory for pipeline %s: %w", key, err)
			}

			library := &pipeline.Libraries[i]
			path := getLibraryPath(library)

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

func getLibraryPath(library *pipelines.PipelineLibrary) string {
	if library.File != nil {
		return library.File.Path
	}

	if library.Jar != "" {
		return library.Jar
	}

	if library.Notebook != nil {
		return library.Notebook.Path
	}

	return ""
}

func cloneWithPath(library *pipelines.PipelineLibrary, path string) pipelines.PipelineLibrary {
	newLib := pipelines.PipelineLibrary{}
	if library.File != nil {
		newLib.File = &pipelines.FileLibrary{
			Path: path,
		}
	}

	if library.Jar != "" {
		newLib.Jar = path
	}

	if library.Notebook != nil {
		newLib.Notebook = &pipelines.NotebookLibrary{
			Path: path,
		}
	}

	return newLib
}

func (*expandGlobPaths) Name() string {
	return "ExpandGlobPaths"
}
