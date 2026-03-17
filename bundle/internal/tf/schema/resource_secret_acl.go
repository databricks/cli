// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSecretAclProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceSecretAcl struct {
	Id             string                           `json:"id,omitempty"`
	Permission     string                           `json:"permission"`
	Principal      string                           `json:"principal"`
	Scope          string                           `json:"scope"`
	ProviderConfig *ResourceSecretAclProviderConfig `json:"provider_config,omitempty"`
}
