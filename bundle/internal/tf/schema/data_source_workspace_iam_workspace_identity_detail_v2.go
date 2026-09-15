// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamWorkspaceIdentityDetailV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamWorkspaceIdentityDetailV2 struct {
	AssignmentType          string                                                         `json:"assignment_type,omitempty"`
	PrincipalId             int                                                            `json:"principal_id"`
	PrincipalType           string                                                         `json:"principal_type,omitempty"`
	ProviderConfig          *DataSourceWorkspaceIamWorkspaceIdentityDetailV2ProviderConfig `json:"provider_config,omitempty"`
	WorkspaceIdentityStatus string                                                         `json:"workspace_identity_status,omitempty"`
}
