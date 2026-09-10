// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamGroupV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamGroupV2 struct {
	AccountId      string                                       `json:"account_id,omitempty"`
	ExternalId     string                                       `json:"external_id,omitempty"`
	GroupId        string                                       `json:"group_id"`
	GroupName      string                                       `json:"group_name,omitempty"`
	ProviderConfig *DataSourceWorkspaceIamGroupV2ProviderConfig `json:"provider_config,omitempty"`
}
