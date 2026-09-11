// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountIamUsersV2UsersFullName struct {
	FamilyName string `json:"family_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
}

type DataSourceAccountIamUsersV2Users struct {
	AccountId         string                                    `json:"account_id,omitempty"`
	AccountUserStatus string                                    `json:"account_user_status,omitempty"`
	ExternalId        string                                    `json:"external_id,omitempty"`
	FullName          *DataSourceAccountIamUsersV2UsersFullName `json:"full_name,omitempty"`
	UserId            string                                    `json:"user_id"`
	Username          string                                    `json:"username,omitempty"`
}

type DataSourceAccountIamUsersV2 struct {
	Filter   string                             `json:"filter,omitempty"`
	PageSize int                                `json:"page_size,omitempty"`
	Users    []DataSourceAccountIamUsersV2Users `json:"users,omitempty"`
}
