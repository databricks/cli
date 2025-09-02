// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceBudgetPoliciesPoliciesCustomTags struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type DataSourceBudgetPoliciesPolicies struct {
	BindingWorkspaceIds []int                                        `json:"binding_workspace_ids,omitempty"`
	CustomTags          []DataSourceBudgetPoliciesPoliciesCustomTags `json:"custom_tags,omitempty"`
	PolicyId            string                                       `json:"policy_id,omitempty"`
	PolicyName          string                                       `json:"policy_name,omitempty"`
}

type DataSourceBudgetPolicies struct {
	Policies []DataSourceBudgetPoliciesPolicies `json:"policies,omitempty"`
}
