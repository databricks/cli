// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceGroupRoleProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceGroupRole struct {
	Api            string                           `json:"api,omitempty"`
	GroupId        string                           `json:"group_id"`
	Id             string                           `json:"id,omitempty"`
	Role           string                           `json:"role"`
	ProviderConfig *ResourceGroupRoleProviderConfig `json:"provider_config,omitempty"`
}
