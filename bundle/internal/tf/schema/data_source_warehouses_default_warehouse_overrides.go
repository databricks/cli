// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWarehousesDefaultWarehouseOverridesDefaultWarehouseOverrides struct {
	DefaultWarehouseOverrideId string `json:"default_warehouse_override_id,omitempty"`
	Name                       string `json:"name"`
	Type                       string `json:"type,omitempty"`
	WarehouseId                string `json:"warehouse_id,omitempty"`
}

type DataSourceWarehousesDefaultWarehouseOverrides struct {
	DefaultWarehouseOverrides []DataSourceWarehousesDefaultWarehouseOverridesDefaultWarehouseOverrides `json:"default_warehouse_overrides,omitempty"`
	PageSize                  int                                                                      `json:"page_size,omitempty"`
}
