// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceUserRoleProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceUserRole struct {
	Api            string                          `json:"api,omitempty"`
	Id             string                          `json:"id,omitempty"`
	Role           string                          `json:"role"`
	UserId         string                          `json:"user_id"`
	ProviderConfig *ResourceUserRoleProviderConfig `json:"provider_config,omitempty"`
}
