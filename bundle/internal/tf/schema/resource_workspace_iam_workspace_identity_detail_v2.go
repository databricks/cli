// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWorkspaceIamWorkspaceIdentityDetailV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceWorkspaceIamWorkspaceIdentityDetailV2 struct {
	AssignmentType          string                                                       `json:"assignment_type,omitempty"`
	PrincipalId             int                                                          `json:"principal_id,omitempty"`
	PrincipalType           string                                                       `json:"principal_type,omitempty"`
	ProviderConfig          *ResourceWorkspaceIamWorkspaceIdentityDetailV2ProviderConfig `json:"provider_config,omitempty"`
	WorkspaceIdentityStatus string                                                       `json:"workspace_identity_status,omitempty"`
}
