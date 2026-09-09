// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountIamDirectGroupMembersV2DirectGroupMembers struct {
	DisplayName      string `json:"display_name,omitempty"`
	ExternalId       string `json:"external_id,omitempty"`
	GroupId          int    `json:"group_id"`
	MembershipSource string `json:"membership_source,omitempty"`
	PrincipalId      int    `json:"principal_id"`
	PrincipalType    string `json:"principal_type,omitempty"`
}

type DataSourceAccountIamDirectGroupMembersV2 struct {
	DirectGroupMembers []DataSourceAccountIamDirectGroupMembersV2DirectGroupMembers `json:"direct_group_members,omitempty"`
	GroupId            int                                                          `json:"group_id"`
	PageSize           int                                                          `json:"page_size,omitempty"`
}
