// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDbfsFilePathsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceDbfsFilePaths struct {
	Id             string                                 `json:"id,omitempty"`
	Path           string                                 `json:"path"`
	PathList       []any                                  `json:"path_list,omitempty"`
	Recursive      bool                                   `json:"recursive"`
	ProviderConfig *DataSourceDbfsFilePathsProviderConfig `json:"provider_config,omitempty"`
}
