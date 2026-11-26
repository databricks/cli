package generate

import (
	"context"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

var pipelineOrder = yamlsaver.NewOrder([]string{"name", "clusters", "configuration", "libraries"})

func ConvertPipelineToValue(pipeline *pipelines.PipelineSpec, rootPath, remoteRootPath string) (dyn.Value, error) {
	if pipeline.RootPath != "" {
		pipeline.RootPath = rootPath
	}

	if pipeline.Libraries != nil && remoteRootPath != "" {
		for i := range pipeline.Libraries {
			lib := &pipeline.Libraries[i]
			if lib.Glob != nil {
				lib.Glob.Include = strings.ReplaceAll(lib.Glob.Include, remoteRootPath, rootPath)
			}
		}
	}

	value := make(map[string]dyn.Value)
	// We ignore the following fields:
	// - id: this is a read-only field
	// - storage: changes to this field are rare because changing the storage recreates pipeline-related resources
	// - edition: this field is rarely changed
	// - development: this field is specific to the mode where it's used and does not need to be saved to the bundle configuration.
	return yamlsaver.ConvertToMapValue(pipeline, pipelineOrder, []string{"id", "storage", "edition", "development", "deployment"}, value)
}

func (n *Downloader) MarkPipelineLibraryForDownload(ctx context.Context, lib *pipelines.PipelineLibrary) error {
	if lib.Notebook != nil {
		return n.markNotebookForDownload(ctx, &lib.Notebook.Path)
	}

	if lib.File != nil {
		return n.markFileForDownload(ctx, &lib.File.Path)
	}

	return nil
}
