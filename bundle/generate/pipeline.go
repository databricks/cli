package generate

import (
	"strings"

	"github.com/databricks/cli/libs/structs/structyaml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

var pipelineOrder = []string{"name", "clusters", "configuration", "libraries"}

func ConvertPipelineToValue(pipeline *pipelines.PipelineSpec, rootPath, remoteRootPath string) (structyaml.Map, error) {
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

	// We ignore the following fields:
	// - id: this is a read-only field
	// - storage: changes to this field are rare because changing the storage recreates pipeline-related resources
	// - edition: this field is rarely changed
	// - development: this field is specific to the mode where it's used and does not need to be saved to the bundle configuration.
	value, err := structyaml.Struct(pipeline, "id", "storage", "edition", "development")
	if err != nil {
		return nil, err
	}
	return value.Order(pipelineOrder...), nil
}
