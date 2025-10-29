// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceTagPolicyValues struct {
	Name string `json:"name"`
}

type ResourceTagPolicy struct {
	CreateTime  string                    `json:"create_time,omitempty"`
	Description string                    `json:"description,omitempty"`
	Id          string                    `json:"id,omitempty"`
	TagKey      string                    `json:"tag_key"`
	UpdateTime  string                    `json:"update_time,omitempty"`
	Values      []ResourceTagPolicyValues `json:"values,omitempty"`
}
