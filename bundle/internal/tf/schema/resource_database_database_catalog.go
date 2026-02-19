// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDatabaseDatabaseCatalogProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceDatabaseDatabaseCatalog struct {
	CreateDatabaseIfNotExists bool                                           `json:"create_database_if_not_exists,omitempty"`
	DatabaseInstanceName      string                                         `json:"database_instance_name"`
	DatabaseName              string                                         `json:"database_name"`
	Name                      string                                         `json:"name"`
	ProviderConfig            *ResourceDatabaseDatabaseCatalogProviderConfig `json:"provider_config,omitempty"`
	Uid                       string                                         `json:"uid,omitempty"`
}
