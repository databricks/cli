// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceServicePrincipalRoleProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceServicePrincipalRole struct {
	Api                string                                      `json:"api,omitempty"`
	Id                 string                                      `json:"id,omitempty"`
	Role               string                                      `json:"role"`
	ServicePrincipalId string                                      `json:"service_principal_id"`
	ProviderConfig     *ResourceServicePrincipalRoleProviderConfig `json:"provider_config,omitempty"`
}
