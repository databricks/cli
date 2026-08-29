// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceGroupMemberProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceGroupMember struct {
	Api            string                             `json:"api,omitempty"`
	GroupId        string                             `json:"group_id"`
	Id             string                             `json:"id,omitempty"`
	MemberId       string                             `json:"member_id"`
	ProviderConfig *ResourceGroupMemberProviderConfig `json:"provider_config,omitempty"`
}
