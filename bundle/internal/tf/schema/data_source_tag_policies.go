// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceTagPoliciesTagPoliciesValues struct {
	Name string `json:"name"`
}

type DataSourceTagPoliciesTagPolicies struct {
	CreateTime  string                                   `json:"create_time,omitempty"`
	Description string                                   `json:"description,omitempty"`
	Id          string                                   `json:"id,omitempty"`
	TagKey      string                                   `json:"tag_key"`
	UpdateTime  string                                   `json:"update_time,omitempty"`
	Values      []DataSourceTagPoliciesTagPoliciesValues `json:"values,omitempty"`
}

type DataSourceTagPolicies struct {
	PageSize    int                                `json:"page_size,omitempty"`
	TagPolicies []DataSourceTagPoliciesTagPolicies `json:"tag_policies,omitempty"`
}
