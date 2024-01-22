package generate

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

var pipelineOrder = yamlsaver.NewOrder([]string{"name", "new_cluster", "existing_cluster_id", "libraries"})

func ConvertPipelineToValue(pipeline *pipelines.PipelineSpec) (dyn.Value, error) {
	value := make(map[string]dyn.Value)
	return yamlsaver.ConvertToMapValue(pipeline, pipelineOrder, []string{"format", "id", "storage", "edition", "channel"}, value)
}
