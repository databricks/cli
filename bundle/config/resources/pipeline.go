package resources

import "github.com/databricks/databricks-sdk-go/service/pipelines"

type Pipeline struct {
	ID          string       `json:"id,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`

	*pipelines.PipelineSpec
}
