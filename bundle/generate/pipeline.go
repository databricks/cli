package generate

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

var pipelineOrder = yamlsaver.NewOrder([]string{"name", "clusters", "configuration", "libraries"})

func ConvertPipelineToValue(pipeline *pipelines.PipelineSpec) (dyn.Value, error) {
	value := make(map[string]dyn.Value)

	// We ignore the following fields:
	// - id: this is a read-only field
	// - storage: changes to this field are rare because changing the storage recreates pipeline-related resources
	// - edition: this field is rarely changed
	return yamlsaver.ConvertToMapValue(pipeline, pipelineOrder, []string{"id", "storage", "edition"}, value)
}
