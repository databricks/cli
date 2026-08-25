// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAccountIamUserV2FullName struct {
	FamilyName string `json:"family_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
}

type ResourceAccountIamUserV2 struct {
	AccountId         string                            `json:"account_id,omitempty"`
	AccountUserStatus string                            `json:"account_user_status"`
	ExternalId        string                            `json:"external_id,omitempty"`
	FullName          *ResourceAccountIamUserV2FullName `json:"full_name,omitempty"`
	UserId            string                            `json:"user_id,omitempty"`
	Username          string                            `json:"username"`
}
