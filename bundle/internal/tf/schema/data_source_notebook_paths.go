// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceNotebookPathsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceNotebookPaths struct {
	Id               string                                 `json:"id,omitempty"`
	NotebookPathList []any                                  `json:"notebook_path_list,omitempty"`
	Path             string                                 `json:"path"`
	Recursive        bool                                   `json:"recursive"`
	ProviderConfig   *DataSourceNotebookPathsProviderConfig `json:"provider_config,omitempty"`
}
