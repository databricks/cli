// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWorkspaceIamUserV2FullName struct {
	FamilyName string `json:"family_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
}

type ResourceWorkspaceIamUserV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceWorkspaceIamUserV2 struct {
	AccountId         string                                    `json:"account_id,omitempty"`
	AccountUserStatus string                                    `json:"account_user_status"`
	ExternalId        string                                    `json:"external_id,omitempty"`
	FullName          *ResourceWorkspaceIamUserV2FullName       `json:"full_name,omitempty"`
	ProviderConfig    *ResourceWorkspaceIamUserV2ProviderConfig `json:"provider_config,omitempty"`
	UserId            string                                    `json:"user_id,omitempty"`
	Username          string                                    `json:"username"`
}
