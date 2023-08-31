package mutator

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

var pipelineTransformers []transformFunc = []transformFunc{
	transformLibraryNotebook,
	transformLibraryFile,
}

func applyPipelineTransformers(m *translatePaths, b *bundle.Bundle) error {
	for key, pipeline := range b.Config.Resources.Pipelines {
		dir, err := pipeline.ConfigFileDirectory()
		if err != nil {
			return fmt.Errorf("unable to determine directory for pipeline %s: %w", key, err)
		}

		for i := 0; i < len(pipeline.Libraries); i++ {
			library := &pipeline.Libraries[i]
			err := applyTransformers(pipelineTransformers, m, b, library, dir)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func transformLibraryNotebook(resource any, dir string) *transformer {
	library, ok := resource.(*pipelines.PipelineLibrary)
	if !ok || library.Notebook == nil {
		return nil
	}

	return &transformer{
		dir,
		&library.Notebook.Path,
		"libraries.notebook.path",
		translateNotebookPath,
	}
}

func transformLibraryFile(resource any, dir string) *transformer {
	library, ok := resource.(*pipelines.PipelineLibrary)
	if !ok || library.File == nil {
		return nil
	}

	return &transformer{
		dir,
		&library.File.Path,
		"libraries.file.path",
		translateFilePath,
	}
}
