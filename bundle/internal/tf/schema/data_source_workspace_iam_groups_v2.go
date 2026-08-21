// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamGroupsV2GroupsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamGroupsV2Groups struct {
	AccountId      string                                              `json:"account_id,omitempty"`
	ExternalId     string                                              `json:"external_id,omitempty"`
	GroupId        string                                              `json:"group_id"`
	GroupName      string                                              `json:"group_name,omitempty"`
	ProviderConfig *DataSourceWorkspaceIamGroupsV2GroupsProviderConfig `json:"provider_config,omitempty"`
}

type DataSourceWorkspaceIamGroupsV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamGroupsV2 struct {
	Filter         string                                        `json:"filter,omitempty"`
	Groups         []DataSourceWorkspaceIamGroupsV2Groups        `json:"groups,omitempty"`
	PageSize       int                                           `json:"page_size,omitempty"`
	ProviderConfig *DataSourceWorkspaceIamGroupsV2ProviderConfig `json:"provider_config,omitempty"`
}
