// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamDirectGroupMemberV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamDirectGroupMemberV2 struct {
	DisplayName      string                                                   `json:"display_name,omitempty"`
	ExternalId       string                                                   `json:"external_id,omitempty"`
	GroupId          int                                                      `json:"group_id"`
	MembershipSource string                                                   `json:"membership_source,omitempty"`
	PrincipalId      int                                                      `json:"principal_id"`
	PrincipalType    string                                                   `json:"principal_type,omitempty"`
	ProviderConfig   *DataSourceWorkspaceIamDirectGroupMemberV2ProviderConfig `json:"provider_config,omitempty"`
}
