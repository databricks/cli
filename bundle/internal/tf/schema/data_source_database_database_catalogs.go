// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDatabaseDatabaseCatalogsDatabaseCatalogsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceDatabaseDatabaseCatalogsDatabaseCatalogs struct {
	CreateDatabaseIfNotExists bool                                                              `json:"create_database_if_not_exists,omitempty"`
	DatabaseInstanceName      string                                                            `json:"database_instance_name,omitempty"`
	DatabaseName              string                                                            `json:"database_name,omitempty"`
	Name                      string                                                            `json:"name"`
	ProviderConfig            *DataSourceDatabaseDatabaseCatalogsDatabaseCatalogsProviderConfig `json:"provider_config,omitempty"`
	Uid                       string                                                            `json:"uid,omitempty"`
}

type DataSourceDatabaseDatabaseCatalogsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceDatabaseDatabaseCatalogs struct {
	DatabaseCatalogs []DataSourceDatabaseDatabaseCatalogsDatabaseCatalogs `json:"database_catalogs,omitempty"`
	InstanceName     string                                               `json:"instance_name"`
	PageSize         int                                                  `json:"page_size,omitempty"`
	ProviderConfig   *DataSourceDatabaseDatabaseCatalogsProviderConfig    `json:"provider_config,omitempty"`
}
