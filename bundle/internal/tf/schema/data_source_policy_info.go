// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePolicyInfoColumnMaskUsing struct {
	Alias    string `json:"alias,omitempty"`
	Constant string `json:"constant,omitempty"`
}

type DataSourcePolicyInfoColumnMask struct {
	FunctionName string                                `json:"function_name"`
	OnColumn     string                                `json:"on_column"`
	Using        []DataSourcePolicyInfoColumnMaskUsing `json:"using,omitempty"`
}

type DataSourcePolicyInfoMatchColumns struct {
	Alias     string `json:"alias,omitempty"`
	Condition string `json:"condition,omitempty"`
}

type DataSourcePolicyInfoRowFilterUsing struct {
	Alias    string `json:"alias,omitempty"`
	Constant string `json:"constant,omitempty"`
}

type DataSourcePolicyInfoRowFilter struct {
	FunctionName string                               `json:"function_name"`
	Using        []DataSourcePolicyInfoRowFilterUsing `json:"using,omitempty"`
}

type DataSourcePolicyInfo struct {
	ColumnMask          *DataSourcePolicyInfoColumnMask    `json:"column_mask,omitempty"`
	Comment             string                             `json:"comment,omitempty"`
	CreatedAt           int                                `json:"created_at,omitempty"`
	CreatedBy           string                             `json:"created_by,omitempty"`
	ExceptPrincipals    []string                           `json:"except_principals,omitempty"`
	ForSecurableType    string                             `json:"for_securable_type"`
	Id                  string                             `json:"id,omitempty"`
	MatchColumns        []DataSourcePolicyInfoMatchColumns `json:"match_columns,omitempty"`
	Name                string                             `json:"name,omitempty"`
	OnSecurableFullname string                             `json:"on_securable_fullname,omitempty"`
	OnSecurableType     string                             `json:"on_securable_type,omitempty"`
	PolicyType          string                             `json:"policy_type"`
	RowFilter           *DataSourcePolicyInfoRowFilter     `json:"row_filter,omitempty"`
	ToPrincipals        []string                           `json:"to_principals"`
	UpdatedAt           int                                `json:"updated_at,omitempty"`
	UpdatedBy           string                             `json:"updated_by,omitempty"`
	WhenCondition       string                             `json:"when_condition,omitempty"`
}
