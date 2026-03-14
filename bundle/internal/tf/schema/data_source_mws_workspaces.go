// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMwsWorkspacesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceMwsWorkspaces struct {
	Id             string                                 `json:"id,omitempty"`
	Ids            map[string]int                         `json:"ids,omitempty"`
	ProviderConfig *DataSourceMwsWorkspacesProviderConfig `json:"provider_config,omitempty"`
}
