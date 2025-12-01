// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDirectoryProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceDirectory struct {
	Id             string                             `json:"id,omitempty"`
	ObjectId       int                                `json:"object_id,omitempty"`
	Path           string                             `json:"path"`
	WorkspacePath  string                             `json:"workspace_path,omitempty"`
	ProviderConfig *DataSourceDirectoryProviderConfig `json:"provider_config,omitempty"`
}
