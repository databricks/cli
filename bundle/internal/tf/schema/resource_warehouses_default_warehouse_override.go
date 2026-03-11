// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWarehousesDefaultWarehouseOverrideProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceWarehousesDefaultWarehouseOverride struct {
	DefaultWarehouseOverrideId string                                                    `json:"default_warehouse_override_id"`
	Name                       string                                                    `json:"name,omitempty"`
	ProviderConfig             *ResourceWarehousesDefaultWarehouseOverrideProviderConfig `json:"provider_config,omitempty"`
	Type                       string                                                    `json:"type"`
	WarehouseId                string                                                    `json:"warehouse_id,omitempty"`
}
