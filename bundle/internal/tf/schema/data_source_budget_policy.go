// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceBudgetPolicyCustomTags struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type DataSourceBudgetPolicy struct {
	BindingWorkspaceIds []int                              `json:"binding_workspace_ids,omitempty"`
	CustomTags          []DataSourceBudgetPolicyCustomTags `json:"custom_tags,omitempty"`
	PolicyId            string                             `json:"policy_id,omitempty"`
	PolicyName          string                             `json:"policy_name,omitempty"`
}
