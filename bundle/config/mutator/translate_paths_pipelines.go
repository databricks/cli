package mutator

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

func getPipelineTransformers(m *translatePaths, b *bundle.Bundle) ([]*transformer, error) {
	var transformers []*transformer = make([]*transformer, 0)

	for key, pipeline := range b.Config.Resources.Pipelines {
		dir, err := pipeline.ConfigFileDirectory()
		if err != nil {
			return nil, fmt.Errorf("unable to determine directory for pipeline %s: %w", key, err)
		}

		for i := 0; i < len(pipeline.Libraries); i++ {
			library := &pipeline.Libraries[i]
			transformers = addTransformerForResource(transformers, m, library, dir)
		}
	}

	return transformers, nil
}

func selectLibraryNotebook(resource interface{}, m *translatePaths) *selector {
	library, ok := resource.(*pipelines.PipelineLibrary)
	if !ok || library.Notebook == nil {
		return nil
	}

	return &selector{
		&library.Notebook.Path,
		"libraries.notebook.path",
		m.translateNotebookPath,
	}
}

func selectLibraryFile(resource interface{}, m *translatePaths) *selector {
	library, ok := resource.(*pipelines.PipelineLibrary)
	if !ok || library.File == nil {
		return nil
	}

	return &selector{
		&library.File.Path,
		"libraries.file.path",
		m.translateFilePath,
	}
}
