// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresDatabaseProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresDatabaseSpec struct {
	PostgresDatabase string `json:"postgres_database,omitempty"`
	Role             string `json:"role,omitempty"`
}

type DataSourcePostgresDatabaseStatus struct {
	DatabaseId       string `json:"database_id,omitempty"`
	PostgresDatabase string `json:"postgres_database,omitempty"`
	Role             string `json:"role,omitempty"`
}

type DataSourcePostgresDatabase struct {
	CreateTime     string                                    `json:"create_time,omitempty"`
	Name           string                                    `json:"name"`
	Parent         string                                    `json:"parent,omitempty"`
	ProviderConfig *DataSourcePostgresDatabaseProviderConfig `json:"provider_config,omitempty"`
	Spec           *DataSourcePostgresDatabaseSpec           `json:"spec,omitempty"`
	Status         *DataSourcePostgresDatabaseStatus         `json:"status,omitempty"`
	UpdateTime     string                                    `json:"update_time,omitempty"`
}
