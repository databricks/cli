// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAccountIamDirectGroupMemberV2 struct {
	DisplayName      string `json:"display_name,omitempty"`
	ExternalId       string `json:"external_id,omitempty"`
	GroupId          int    `json:"group_id,omitempty"`
	MembershipSource string `json:"membership_source,omitempty"`
	PrincipalId      int    `json:"principal_id"`
	PrincipalType    string `json:"principal_type,omitempty"`
}
