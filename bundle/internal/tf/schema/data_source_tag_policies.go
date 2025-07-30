// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceTagPoliciesTagPoliciesValues struct {
	Name string `json:"name"`
}

type DataSourceTagPoliciesTagPolicies struct {
	Description string                                   `json:"description,omitempty"`
	Id          string                                   `json:"id,omitempty"`
	TagKey      string                                   `json:"tag_key"`
	Values      []DataSourceTagPoliciesTagPoliciesValues `json:"values,omitempty"`
}

type DataSourceTagPolicies struct {
	TagPolicies []DataSourceTagPoliciesTagPolicies `json:"tag_policies,omitempty"`
}
