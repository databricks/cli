// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceTablesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceTables struct {
	CatalogName    string                          `json:"catalog_name"`
	Id             string                          `json:"id,omitempty"`
	Ids            []string                        `json:"ids,omitempty"`
	SchemaName     string                          `json:"schema_name"`
	ProviderConfig *DataSourceTablesProviderConfig `json:"provider_config,omitempty"`
}
