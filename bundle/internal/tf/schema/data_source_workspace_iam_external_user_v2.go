// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamExternalUserV2FullName struct {
	FamilyName string `json:"family_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
}

type DataSourceWorkspaceIamExternalUserV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamExternalUserV2 struct {
	AccountId         string                                              `json:"account_id,omitempty"`
	AccountUserStatus string                                              `json:"account_user_status,omitempty"`
	DisplayName       string                                              `json:"display_name,omitempty"`
	ExternalUserId    string                                              `json:"external_user_id,omitempty"`
	FullName          *DataSourceWorkspaceIamExternalUserV2FullName       `json:"full_name,omitempty"`
	InternalId        string                                              `json:"internal_id,omitempty"`
	Name              string                                              `json:"name"`
	ProviderConfig    *DataSourceWorkspaceIamExternalUserV2ProviderConfig `json:"provider_config,omitempty"`
	Username          string                                              `json:"username,omitempty"`
}
