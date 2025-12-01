// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceVolumesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceVolumes struct {
	CatalogName    string                           `json:"catalog_name"`
	Ids            []string                         `json:"ids,omitempty"`
	ProviderConfig *DataSourceVolumesProviderConfig `json:"provider_config,omitempty"`
	SchemaName     string                           `json:"schema_name"`
}
