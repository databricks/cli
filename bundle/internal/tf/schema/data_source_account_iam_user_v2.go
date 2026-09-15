// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountIamUserV2FullName struct {
	FamilyName string `json:"family_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
}

type DataSourceAccountIamUserV2 struct {
	AccountId         string                              `json:"account_id,omitempty"`
	AccountUserStatus string                              `json:"account_user_status,omitempty"`
	ExternalId        string                              `json:"external_id,omitempty"`
	FullName          *DataSourceAccountIamUserV2FullName `json:"full_name,omitempty"`
	UserId            string                              `json:"user_id"`
	Username          string                              `json:"username,omitempty"`
}
