// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceCatalogsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceCatalogs struct {
	Id             string                            `json:"id,omitempty"`
	Ids            []string                          `json:"ids,omitempty"`
	ProviderConfig *DataSourceCatalogsProviderConfig `json:"provider_config,omitempty"`
}
