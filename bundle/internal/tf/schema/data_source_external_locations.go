// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceExternalLocationsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceExternalLocations struct {
	Id             string                                     `json:"id,omitempty"`
	Names          []string                                   `json:"names,omitempty"`
	ProviderConfig *DataSourceExternalLocationsProviderConfig `json:"provider_config,omitempty"`
}
