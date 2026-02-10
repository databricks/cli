// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMwsCredentialsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceMwsCredentials struct {
	Id             string                                  `json:"id,omitempty"`
	Ids            map[string]string                       `json:"ids,omitempty"`
	ProviderConfig *DataSourceMwsCredentialsProviderConfig `json:"provider_config,omitempty"`
}
