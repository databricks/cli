// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceBudgetPoliciesFilterBy struct {
	CreatorUserId   int    `json:"creator_user_id,omitempty"`
	CreatorUserName string `json:"creator_user_name,omitempty"`
	PolicyName      string `json:"policy_name,omitempty"`
}

type DataSourceBudgetPoliciesPoliciesCustomTags struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type DataSourceBudgetPoliciesPolicies struct {
	BindingWorkspaceIds []int                                        `json:"binding_workspace_ids,omitempty"`
	CustomTags          []DataSourceBudgetPoliciesPoliciesCustomTags `json:"custom_tags,omitempty"`
	PolicyId            string                                       `json:"policy_id"`
	PolicyName          string                                       `json:"policy_name,omitempty"`
}

type DataSourceBudgetPoliciesSortSpec struct {
	Descending bool   `json:"descending,omitempty"`
	Field      string `json:"field,omitempty"`
}

type DataSourceBudgetPolicies struct {
	FilterBy *DataSourceBudgetPoliciesFilterBy  `json:"filter_by,omitempty"`
	PageSize int                                `json:"page_size,omitempty"`
	Policies []DataSourceBudgetPoliciesPolicies `json:"policies,omitempty"`
	SortSpec *DataSourceBudgetPoliciesSortSpec  `json:"sort_spec,omitempty"`
}
