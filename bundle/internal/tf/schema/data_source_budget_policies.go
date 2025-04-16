// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceBudgetPoliciesBudgetPoliciesCustomTags struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type DataSourceBudgetPoliciesBudgetPolicies struct {
	BindingWorkspaceIds []int                                              `json:"binding_workspace_ids,omitempty"`
	CustomTags          []DataSourceBudgetPoliciesBudgetPoliciesCustomTags `json:"custom_tags,omitempty"`
	PolicyId            string                                             `json:"policy_id,omitempty"`
	PolicyName          string                                             `json:"policy_name,omitempty"`
}

type DataSourceBudgetPolicies struct {
	BudgetPolicies []DataSourceBudgetPoliciesBudgetPolicies `json:"budget_policies,omitempty"`
}
