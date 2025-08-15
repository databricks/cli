// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePolicyInfosPoliciesColumnMaskUsing struct {
	Alias    string `json:"alias,omitempty"`
	Constant string `json:"constant,omitempty"`
}

type DataSourcePolicyInfosPoliciesColumnMask struct {
	FunctionName string                                         `json:"function_name"`
	OnColumn     string                                         `json:"on_column"`
	Using        []DataSourcePolicyInfosPoliciesColumnMaskUsing `json:"using,omitempty"`
}

type DataSourcePolicyInfosPoliciesMatchColumns struct {
	Alias     string `json:"alias,omitempty"`
	Condition string `json:"condition,omitempty"`
}

type DataSourcePolicyInfosPoliciesRowFilterUsing struct {
	Alias    string `json:"alias,omitempty"`
	Constant string `json:"constant,omitempty"`
}

type DataSourcePolicyInfosPoliciesRowFilter struct {
	FunctionName string                                        `json:"function_name"`
	Using        []DataSourcePolicyInfosPoliciesRowFilterUsing `json:"using,omitempty"`
}

type DataSourcePolicyInfosPolicies struct {
	ColumnMask          *DataSourcePolicyInfosPoliciesColumnMask    `json:"column_mask,omitempty"`
	Comment             string                                      `json:"comment,omitempty"`
	CreatedAt           int                                         `json:"created_at,omitempty"`
	CreatedBy           string                                      `json:"created_by,omitempty"`
	ExceptPrincipals    []string                                    `json:"except_principals,omitempty"`
	ForSecurableType    string                                      `json:"for_securable_type"`
	Id                  string                                      `json:"id,omitempty"`
	MatchColumns        []DataSourcePolicyInfosPoliciesMatchColumns `json:"match_columns,omitempty"`
	Name                string                                      `json:"name,omitempty"`
	OnSecurableFullname string                                      `json:"on_securable_fullname,omitempty"`
	OnSecurableType     string                                      `json:"on_securable_type,omitempty"`
	PolicyType          string                                      `json:"policy_type"`
	RowFilter           *DataSourcePolicyInfosPoliciesRowFilter     `json:"row_filter,omitempty"`
	ToPrincipals        []string                                    `json:"to_principals"`
	UpdatedAt           int                                         `json:"updated_at,omitempty"`
	UpdatedBy           string                                      `json:"updated_by,omitempty"`
	WhenCondition       string                                      `json:"when_condition,omitempty"`
}

type DataSourcePolicyInfos struct {
	Policies []DataSourcePolicyInfosPolicies `json:"policies,omitempty"`
}
