// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWarehousesDefaultWarehouseOverrideProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceWarehousesDefaultWarehouseOverride struct {
	DefaultWarehouseOverrideId string                                                      `json:"default_warehouse_override_id,omitempty"`
	Name                       string                                                      `json:"name"`
	ProviderConfig             *DataSourceWarehousesDefaultWarehouseOverrideProviderConfig `json:"provider_config,omitempty"`
	Type                       string                                                      `json:"type,omitempty"`
	WarehouseId                string                                                      `json:"warehouse_id,omitempty"`
}
