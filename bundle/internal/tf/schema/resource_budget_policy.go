// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceBudgetPolicyCustomTags struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type ResourceBudgetPolicy struct {
	BindingWorkspaceIds []int                            `json:"binding_workspace_ids,omitempty"`
	CustomTags          []ResourceBudgetPolicyCustomTags `json:"custom_tags,omitempty"`
	PolicyId            string                           `json:"policy_id,omitempty"`
	PolicyName          string                           `json:"policy_name,omitempty"`
}
