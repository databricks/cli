// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospectionColumnTagValue struct {
	ColumnAlias string `json:"column_alias"`
	TagKey      string `json:"tag_key"`
}

type ResourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospectionTagValue struct {
	TagKey string `json:"tag_key"`
}

type ResourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospection struct {
	ColumnTagValue *ResourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospectionColumnTagValue `json:"column_tag_value,omitempty"`
	TagValue       *ResourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospectionTagValue       `json:"tag_value,omitempty"`
}

type ResourcePolicyInfoColumnMaskUsingFunctionArgExpression struct {
	TagIntrospection *ResourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospection `json:"tag_introspection,omitempty"`
}

type ResourcePolicyInfoColumnMaskUsing struct {
	Alias                 string                                                  `json:"alias,omitempty"`
	Constant              string                                                  `json:"constant,omitempty"`
	FunctionArgExpression *ResourcePolicyInfoColumnMaskUsingFunctionArgExpression `json:"function_arg_expression,omitempty"`
}

type ResourcePolicyInfoColumnMask struct {
	FunctionName string                              `json:"function_name"`
	OnColumn     string                              `json:"on_column"`
	Using        []ResourcePolicyInfoColumnMaskUsing `json:"using,omitempty"`
}

type ResourcePolicyInfoGrant struct {
	Privileges []string `json:"privileges"`
}

type ResourcePolicyInfoMatchColumns struct {
	Alias     string `json:"alias,omitempty"`
	Condition string `json:"condition,omitempty"`
}

type ResourcePolicyInfoProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospectionColumnTagValue struct {
	ColumnAlias string `json:"column_alias"`
	TagKey      string `json:"tag_key"`
}

type ResourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospectionTagValue struct {
	TagKey string `json:"tag_key"`
}

type ResourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospection struct {
	ColumnTagValue *ResourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospectionColumnTagValue `json:"column_tag_value,omitempty"`
	TagValue       *ResourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospectionTagValue       `json:"tag_value,omitempty"`
}

type ResourcePolicyInfoRowFilterUsingFunctionArgExpression struct {
	TagIntrospection *ResourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospection `json:"tag_introspection,omitempty"`
}

type ResourcePolicyInfoRowFilterUsing struct {
	Alias                 string                                                 `json:"alias,omitempty"`
	Constant              string                                                 `json:"constant,omitempty"`
	FunctionArgExpression *ResourcePolicyInfoRowFilterUsingFunctionArgExpression `json:"function_arg_expression,omitempty"`
}

type ResourcePolicyInfoRowFilter struct {
	FunctionName string                             `json:"function_name"`
	Using        []ResourcePolicyInfoRowFilterUsing `json:"using,omitempty"`
}

type ResourcePolicyInfo struct {
	ColumnMask          *ResourcePolicyInfoColumnMask     `json:"column_mask,omitempty"`
	Comment             string                            `json:"comment,omitempty"`
	CreatedAt           int                               `json:"created_at,omitempty"`
	CreatedBy           string                            `json:"created_by,omitempty"`
	ExceptPrincipals    []string                          `json:"except_principals,omitempty"`
	ForSecurableType    string                            `json:"for_securable_type"`
	Grant               *ResourcePolicyInfoGrant          `json:"grant,omitempty"`
	Id                  string                            `json:"id,omitempty"`
	MatchColumns        []ResourcePolicyInfoMatchColumns  `json:"match_columns,omitempty"`
	Name                string                            `json:"name,omitempty"`
	OnSecurableFullname string                            `json:"on_securable_fullname,omitempty"`
	OnSecurableType     string                            `json:"on_securable_type,omitempty"`
	PolicyType          string                            `json:"policy_type"`
	ProviderConfig      *ResourcePolicyInfoProviderConfig `json:"provider_config,omitempty"`
	RowFilter           *ResourcePolicyInfoRowFilter      `json:"row_filter,omitempty"`
	ToPrincipals        []string                          `json:"to_principals"`
	UpdatedAt           int                               `json:"updated_at,omitempty"`
	UpdatedBy           string                            `json:"updated_by,omitempty"`
	WhenCondition       string                            `json:"when_condition,omitempty"`
}
