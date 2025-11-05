// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceZonesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceZones struct {
	DefaultZone    string                         `json:"default_zone,omitempty"`
	Id             string                         `json:"id,omitempty"`
	Zones          []string                       `json:"zones,omitempty"`
	ProviderConfig *DataSourceZonesProviderConfig `json:"provider_config,omitempty"`
}
