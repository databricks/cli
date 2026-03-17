// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWarehousesDefaultWarehouseOverridesDefaultWarehouseOverridesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceWarehousesDefaultWarehouseOverridesDefaultWarehouseOverrides struct {
	DefaultWarehouseOverrideId string                                                                                `json:"default_warehouse_override_id,omitempty"`
	Name                       string                                                                                `json:"name"`
	ProviderConfig             *DataSourceWarehousesDefaultWarehouseOverridesDefaultWarehouseOverridesProviderConfig `json:"provider_config,omitempty"`
	Type                       string                                                                                `json:"type,omitempty"`
	WarehouseId                string                                                                                `json:"warehouse_id,omitempty"`
}

type DataSourceWarehousesDefaultWarehouseOverridesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceWarehousesDefaultWarehouseOverrides struct {
	DefaultWarehouseOverrides []DataSourceWarehousesDefaultWarehouseOverridesDefaultWarehouseOverrides `json:"default_warehouse_overrides,omitempty"`
	PageSize                  int                                                                      `json:"page_size,omitempty"`
	ProviderConfig            *DataSourceWarehousesDefaultWarehouseOverridesProviderConfig             `json:"provider_config,omitempty"`
}
