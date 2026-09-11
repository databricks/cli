// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountIamGroupsV2Groups struct {
	AccountId  string `json:"account_id,omitempty"`
	ExternalId string `json:"external_id,omitempty"`
	GroupId    string `json:"group_id"`
	GroupName  string `json:"group_name,omitempty"`
}

type DataSourceAccountIamGroupsV2 struct {
	Filter   string                               `json:"filter,omitempty"`
	Groups   []DataSourceAccountIamGroupsV2Groups `json:"groups,omitempty"`
	PageSize int                                  `json:"page_size,omitempty"`
}
