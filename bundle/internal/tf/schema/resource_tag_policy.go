// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceTagPolicyValues struct {
	Name string `json:"name"`
}

type ResourceTagPolicy struct {
	Description string                    `json:"description,omitempty"`
	Id          string                    `json:"id,omitempty"`
	TagKey      string                    `json:"tag_key"`
	Values      []ResourceTagPolicyValues `json:"values,omitempty"`
}
