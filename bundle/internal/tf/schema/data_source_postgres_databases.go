// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresDatabasesDatabasesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresDatabasesDatabasesSpec struct {
	PostgresDatabase string `json:"postgres_database,omitempty"`
	Role             string `json:"role,omitempty"`
}

type DataSourcePostgresDatabasesDatabasesStatus struct {
	DatabaseId       string `json:"database_id,omitempty"`
	PostgresDatabase string `json:"postgres_database,omitempty"`
	Role             string `json:"role,omitempty"`
}

type DataSourcePostgresDatabasesDatabases struct {
	CreateTime     string                                              `json:"create_time,omitempty"`
	Name           string                                              `json:"name"`
	Parent         string                                              `json:"parent,omitempty"`
	ProviderConfig *DataSourcePostgresDatabasesDatabasesProviderConfig `json:"provider_config,omitempty"`
	Spec           *DataSourcePostgresDatabasesDatabasesSpec           `json:"spec,omitempty"`
	Status         *DataSourcePostgresDatabasesDatabasesStatus         `json:"status,omitempty"`
	UpdateTime     string                                              `json:"update_time,omitempty"`
}

type DataSourcePostgresDatabasesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresDatabases struct {
	Databases      []DataSourcePostgresDatabasesDatabases     `json:"databases,omitempty"`
	PageSize       int                                        `json:"page_size,omitempty"`
	Parent         string                                     `json:"parent"`
	ProviderConfig *DataSourcePostgresDatabasesProviderConfig `json:"provider_config,omitempty"`
}
