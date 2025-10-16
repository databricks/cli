// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDatabaseDatabaseCatalogsDatabaseCatalogs struct {
	CreateDatabaseIfNotExists bool   `json:"create_database_if_not_exists,omitempty"`
	DatabaseInstanceName      string `json:"database_instance_name"`
	DatabaseName              string `json:"database_name"`
	Name                      string `json:"name"`
	Uid                       string `json:"uid,omitempty"`
}

type DataSourceDatabaseDatabaseCatalogs struct {
	DatabaseCatalogs []DataSourceDatabaseDatabaseCatalogsDatabaseCatalogs `json:"database_catalogs,omitempty"`
}
