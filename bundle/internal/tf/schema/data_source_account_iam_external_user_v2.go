// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountIamExternalUserV2FullName struct {
	FamilyName string `json:"family_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
}

type DataSourceAccountIamExternalUserV2 struct {
	AccountId         string                                      `json:"account_id,omitempty"`
	AccountUserStatus string                                      `json:"account_user_status,omitempty"`
	DisplayName       string                                      `json:"display_name,omitempty"`
	ExternalUserId    string                                      `json:"external_user_id,omitempty"`
	FullName          *DataSourceAccountIamExternalUserV2FullName `json:"full_name,omitempty"`
	InternalId        string                                      `json:"internal_id,omitempty"`
	Name              string                                      `json:"name"`
	Username          string                                      `json:"username,omitempty"`
}
