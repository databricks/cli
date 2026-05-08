// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresCatalogProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourcePostgresCatalogSpec struct {
	Branch                  string `json:"branch,omitempty"`
	CreateDatabaseIfMissing bool   `json:"create_database_if_missing,omitempty"`
	PostgresDatabase        string `json:"postgres_database"`
}

type DataSourcePostgresCatalogStatus struct {
	Branch           string `json:"branch,omitempty"`
	PostgresDatabase string `json:"postgres_database,omitempty"`
	Project          string `json:"project,omitempty"`
}

type DataSourcePostgresCatalog struct {
	CreateTime     string                                   `json:"create_time,omitempty"`
	Name           string                                   `json:"name"`
	ProviderConfig *DataSourcePostgresCatalogProviderConfig `json:"provider_config,omitempty"`
	Spec           *DataSourcePostgresCatalogSpec           `json:"spec,omitempty"`
	Status         *DataSourcePostgresCatalogStatus         `json:"status,omitempty"`
	Uid            string                                   `json:"uid,omitempty"`
	UpdateTime     string                                   `json:"update_time,omitempty"`
}
