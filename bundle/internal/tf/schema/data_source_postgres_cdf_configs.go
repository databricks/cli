// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresCdfConfigsCdfConfigsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresCdfConfigsCdfConfigs struct {
	Catalog        string                                                `json:"catalog,omitempty"`
	CdfConfigId    string                                                `json:"cdf_config_id,omitempty"`
	CreateTime     string                                                `json:"create_time,omitempty"`
	Name           string                                                `json:"name"`
	PostgresSchema string                                                `json:"postgres_schema,omitempty"`
	ProviderConfig *DataSourcePostgresCdfConfigsCdfConfigsProviderConfig `json:"provider_config,omitempty"`
	Schema         string                                                `json:"schema,omitempty"`
}

type DataSourcePostgresCdfConfigsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresCdfConfigs struct {
	CdfConfigs     []DataSourcePostgresCdfConfigsCdfConfigs    `json:"cdf_configs,omitempty"`
	PageSize       int                                         `json:"page_size,omitempty"`
	Parent         string                                      `json:"parent"`
	ProviderConfig *DataSourcePostgresCdfConfigsProviderConfig `json:"provider_config,omitempty"`
}
