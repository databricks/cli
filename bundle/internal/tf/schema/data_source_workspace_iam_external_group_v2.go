// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamExternalGroupV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamExternalGroupV2 struct {
	AccountId       string                                               `json:"account_id,omitempty"`
	DisplayName     string                                               `json:"display_name,omitempty"`
	ExternalGroupId string                                               `json:"external_group_id,omitempty"`
	InternalId      string                                               `json:"internal_id,omitempty"`
	Name            string                                               `json:"name"`
	ProviderConfig  *DataSourceWorkspaceIamExternalGroupV2ProviderConfig `json:"provider_config,omitempty"`
}
