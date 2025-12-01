// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDatabaseDatabaseCatalog struct {
	CreateDatabaseIfNotExists bool   `json:"create_database_if_not_exists,omitempty"`
	DatabaseInstanceName      string `json:"database_instance_name,omitempty"`
	DatabaseName              string `json:"database_name,omitempty"`
	Name                      string `json:"name"`
	Uid                       string `json:"uid,omitempty"`
}
