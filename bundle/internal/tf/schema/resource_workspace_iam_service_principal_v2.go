// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWorkspaceIamServicePrincipalV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceWorkspaceIamServicePrincipalV2 struct {
	AccountId          string                                                `json:"account_id,omitempty"`
	AccountSpStatus    string                                                `json:"account_sp_status"`
	ApplicationId      string                                                `json:"application_id,omitempty"`
	DisplayName        string                                                `json:"display_name"`
	ExternalId         string                                                `json:"external_id,omitempty"`
	ProviderConfig     *ResourceWorkspaceIamServicePrincipalV2ProviderConfig `json:"provider_config,omitempty"`
	ServicePrincipalId string                                                `json:"service_principal_id,omitempty"`
}
