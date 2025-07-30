// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceTagPolicyValues struct {
	Name string `json:"name"`
}

type DataSourceTagPolicy struct {
	Description string                      `json:"description,omitempty"`
	Id          string                      `json:"id,omitempty"`
	TagKey      string                      `json:"tag_key"`
	Values      []DataSourceTagPolicyValues `json:"values,omitempty"`
}
