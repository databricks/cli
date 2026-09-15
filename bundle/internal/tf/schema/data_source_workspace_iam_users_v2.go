// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamUsersV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamUsersV2UsersFullName struct {
	FamilyName string `json:"family_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
}

type DataSourceWorkspaceIamUsersV2UsersProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamUsersV2Users struct {
	AccountId         string                                            `json:"account_id,omitempty"`
	AccountUserStatus string                                            `json:"account_user_status,omitempty"`
	ExternalId        string                                            `json:"external_id,omitempty"`
	FullName          *DataSourceWorkspaceIamUsersV2UsersFullName       `json:"full_name,omitempty"`
	ProviderConfig    *DataSourceWorkspaceIamUsersV2UsersProviderConfig `json:"provider_config,omitempty"`
	UserId            string                                            `json:"user_id"`
	Username          string                                            `json:"username,omitempty"`
}

type DataSourceWorkspaceIamUsersV2 struct {
	Filter         string                                       `json:"filter,omitempty"`
	PageSize       int                                          `json:"page_size,omitempty"`
	ProviderConfig *DataSourceWorkspaceIamUsersV2ProviderConfig `json:"provider_config,omitempty"`
	Users          []DataSourceWorkspaceIamUsersV2Users         `json:"users,omitempty"`
}
