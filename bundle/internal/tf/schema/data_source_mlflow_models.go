// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMlflowModelsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceMlflowModels struct {
	Id             string                                `json:"id,omitempty"`
	Names          []string                              `json:"names,omitempty"`
	ProviderConfig *DataSourceMlflowModelsProviderConfig `json:"provider_config,omitempty"`
}
