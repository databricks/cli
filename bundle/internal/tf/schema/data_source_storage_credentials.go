// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceStorageCredentialsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceStorageCredentials struct {
	Id             string                                      `json:"id,omitempty"`
	Names          []string                                    `json:"names,omitempty"`
	ProviderConfig *DataSourceStorageCredentialsProviderConfig `json:"provider_config,omitempty"`
}
