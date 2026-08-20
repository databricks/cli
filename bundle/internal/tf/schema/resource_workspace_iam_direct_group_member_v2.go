// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWorkspaceIamDirectGroupMemberV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceWorkspaceIamDirectGroupMemberV2 struct {
	DisplayName      string                                                 `json:"display_name,omitempty"`
	ExternalId       string                                                 `json:"external_id,omitempty"`
	GroupId          int                                                    `json:"group_id,omitempty"`
	MembershipSource string                                                 `json:"membership_source,omitempty"`
	PrincipalId      int                                                    `json:"principal_id"`
	PrincipalType    string                                                 `json:"principal_type,omitempty"`
	ProviderConfig   *ResourceWorkspaceIamDirectGroupMemberV2ProviderConfig `json:"provider_config,omitempty"`
}
