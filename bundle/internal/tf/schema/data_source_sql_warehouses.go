// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSqlWarehousesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceSqlWarehouses struct {
	Id                    string                                 `json:"id,omitempty"`
	Ids                   []string                               `json:"ids,omitempty"`
	WarehouseNameContains string                                 `json:"warehouse_name_contains,omitempty"`
	ProviderConfig        *DataSourceSqlWarehousesProviderConfig `json:"provider_config,omitempty"`
}
