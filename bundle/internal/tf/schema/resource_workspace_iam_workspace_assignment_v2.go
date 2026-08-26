// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWorkspaceIamWorkspaceAssignmentV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceWorkspaceIamWorkspaceAssignmentV2 struct {
	AccountId             string                                                   `json:"account_id,omitempty"`
	EffectiveEntitlements []string                                                 `json:"effective_entitlements,omitempty"`
	Entitlements          []string                                                 `json:"entitlements,omitempty"`
	PrincipalId           int                                                      `json:"principal_id"`
	PrincipalType         string                                                   `json:"principal_type,omitempty"`
	ProviderConfig        *ResourceWorkspaceIamWorkspaceAssignmentV2ProviderConfig `json:"provider_config,omitempty"`
	WorkspaceId           int                                                      `json:"workspace_id,omitempty"`
}
