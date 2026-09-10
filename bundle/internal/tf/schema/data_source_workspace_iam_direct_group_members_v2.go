// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamDirectGroupMembersV2DirectGroupMembersProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamDirectGroupMembersV2DirectGroupMembers struct {
	DisplayName      string                                                                      `json:"display_name,omitempty"`
	ExternalId       string                                                                      `json:"external_id,omitempty"`
	GroupId          int                                                                         `json:"group_id"`
	MembershipSource string                                                                      `json:"membership_source,omitempty"`
	PrincipalId      int                                                                         `json:"principal_id"`
	PrincipalType    string                                                                      `json:"principal_type,omitempty"`
	ProviderConfig   *DataSourceWorkspaceIamDirectGroupMembersV2DirectGroupMembersProviderConfig `json:"provider_config,omitempty"`
}

type DataSourceWorkspaceIamDirectGroupMembersV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamDirectGroupMembersV2 struct {
	DirectGroupMembers []DataSourceWorkspaceIamDirectGroupMembersV2DirectGroupMembers `json:"direct_group_members,omitempty"`
	GroupId            int                                                            `json:"group_id"`
	PageSize           int                                                            `json:"page_size,omitempty"`
	ProviderConfig     *DataSourceWorkspaceIamDirectGroupMembersV2ProviderConfig      `json:"provider_config,omitempty"`
}
