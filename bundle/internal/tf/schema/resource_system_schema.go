// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSystemSchemaProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceSystemSchema struct {
	AutoEnabled    bool                                `json:"auto_enabled,omitempty"`
	FullName       string                              `json:"full_name,omitempty"`
	Id             string                              `json:"id,omitempty"`
	MetastoreId    string                              `json:"metastore_id,omitempty"`
	Schema         string                              `json:"schema"`
	State          string                              `json:"state,omitempty"`
	ProviderConfig *ResourceSystemSchemaProviderConfig `json:"provider_config,omitempty"`
}
