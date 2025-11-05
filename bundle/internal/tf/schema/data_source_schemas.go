// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSchemasProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceSchemas struct {
	CatalogName    string                           `json:"catalog_name"`
	Id             string                           `json:"id,omitempty"`
	Ids            []string                         `json:"ids,omitempty"`
	ProviderConfig *DataSourceSchemasProviderConfig `json:"provider_config,omitempty"`
}
