// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSecretUcProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceSecretUc struct {
	BrowseOnly       bool                            `json:"browse_only,omitempty"`
	CatalogName      string                          `json:"catalog_name"`
	Comment          string                          `json:"comment,omitempty"`
	CreateTime       string                          `json:"create_time,omitempty"`
	CreatedBy        string                          `json:"created_by,omitempty"`
	EffectiveOwner   string                          `json:"effective_owner,omitempty"`
	EffectiveValue   string                          `json:"effective_value,omitempty"`
	ExpireTime       string                          `json:"expire_time,omitempty"`
	ExternalSecretId string                          `json:"external_secret_id,omitempty"`
	FullName         string                          `json:"full_name,omitempty"`
	MetastoreId      string                          `json:"metastore_id,omitempty"`
	Name             string                          `json:"name"`
	Owner            string                          `json:"owner,omitempty"`
	ProviderConfig   *ResourceSecretUcProviderConfig `json:"provider_config,omitempty"`
	SchemaName       string                          `json:"schema_name"`
	UpdateTime       string                          `json:"update_time,omitempty"`
	UpdatedBy        string                          `json:"updated_by,omitempty"`
	Value            string                          `json:"value"`
}
