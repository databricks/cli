package generate

import (
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

var pipelineOrder = yamlsaver.NewOrder([]string{"name", "clusters", "configuration", "libraries"})

func ConvertPipelineToValue(pipeline *pipelines.PipelineSpec, rootPath string, remoteRootPath string) (dyn.Value, error) {
	if pipeline.RootPath != "" {
		pipeline.RootPath = rootPath
	}

	if pipeline.Libraries != nil {
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
	return yamlsaver.ConvertToMapValue(pipeline, pipelineOrder, []string{"id", "storage", "edition"}, value)
}
