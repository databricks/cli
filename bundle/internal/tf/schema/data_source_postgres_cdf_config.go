// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresCdfConfigProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresCdfConfig struct {
	Catalog        string                                     `json:"catalog,omitempty"`
	CdfConfigId    string                                     `json:"cdf_config_id,omitempty"`
	CreateTime     string                                     `json:"create_time,omitempty"`
	Name           string                                     `json:"name"`
	PostgresSchema string                                     `json:"postgres_schema,omitempty"`
	ProviderConfig *DataSourcePostgresCdfConfigProviderConfig `json:"provider_config,omitempty"`
	Schema         string                                     `json:"schema,omitempty"`
}
