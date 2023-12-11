// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMwsCredentials struct {
	AccountId       string `json:"account_id,omitempty"`
	CreationTime    int    `json:"creation_time,omitempty"`
	CredentialsId   string `json:"credentials_id,omitempty"`
	CredentialsName string `json:"credentials_name"`
	ExternalId      string `json:"external_id,omitempty"`
	Id              string `json:"id,omitempty"`
	RoleArn         string `json:"role_arn"`
}
