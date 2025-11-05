// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePipelinesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourcePipelines struct {
	Id             string                             `json:"id,omitempty"`
	Ids            []string                           `json:"ids,omitempty"`
	PipelineName   string                             `json:"pipeline_name,omitempty"`
	ProviderConfig *DataSourcePipelinesProviderConfig `json:"provider_config,omitempty"`
}
