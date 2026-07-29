// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePostgresCdfConfigProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourcePostgresCdfConfig struct {
	Catalog        string                                   `json:"catalog"`
	CdfConfigId    string                                   `json:"cdf_config_id,omitempty"`
	CreateTime     string                                   `json:"create_time,omitempty"`
	Name           string                                   `json:"name,omitempty"`
	Parent         string                                   `json:"parent"`
	PostgresSchema string                                   `json:"postgres_schema"`
	ProviderConfig *ResourcePostgresCdfConfigProviderConfig `json:"provider_config,omitempty"`
	Schema         string                                   `json:"schema"`
}
