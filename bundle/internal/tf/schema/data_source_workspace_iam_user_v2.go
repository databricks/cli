// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamUserV2FullName struct {
	FamilyName string `json:"family_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
}

type DataSourceWorkspaceIamUserV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamUserV2 struct {
	AccountId         string                                      `json:"account_id,omitempty"`
	AccountUserStatus string                                      `json:"account_user_status,omitempty"`
	ExternalId        string                                      `json:"external_id,omitempty"`
	FullName          *DataSourceWorkspaceIamUserV2FullName       `json:"full_name,omitempty"`
	ProviderConfig    *DataSourceWorkspaceIamUserV2ProviderConfig `json:"provider_config,omitempty"`
	UserId            string                                      `json:"user_id"`
	Username          string                                      `json:"username,omitempty"`
}
