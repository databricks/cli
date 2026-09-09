// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospectionColumnTagValue struct {
	ColumnAlias string `json:"column_alias"`
	TagKey      string `json:"tag_key"`
}

type DataSourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospectionTagValue struct {
	TagKey string `json:"tag_key"`
}

type DataSourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospection struct {
	ColumnTagValue *DataSourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospectionColumnTagValue `json:"column_tag_value,omitempty"`
	TagValue       *DataSourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospectionTagValue       `json:"tag_value,omitempty"`
}

type DataSourcePolicyInfoColumnMaskUsingFunctionArgExpression struct {
	TagIntrospection *DataSourcePolicyInfoColumnMaskUsingFunctionArgExpressionTagIntrospection `json:"tag_introspection,omitempty"`
}

type DataSourcePolicyInfoColumnMaskUsing struct {
	Alias                 string                                                    `json:"alias,omitempty"`
	Constant              string                                                    `json:"constant,omitempty"`
	FunctionArgExpression *DataSourcePolicyInfoColumnMaskUsingFunctionArgExpression `json:"function_arg_expression,omitempty"`
}

type DataSourcePolicyInfoColumnMask struct {
	FunctionName string                                `json:"function_name"`
	OnColumn     string                                `json:"on_column"`
	Using        []DataSourcePolicyInfoColumnMaskUsing `json:"using,omitempty"`
}

type DataSourcePolicyInfoGrant struct {
	Privileges []string `json:"privileges"`
}

type DataSourcePolicyInfoMatchColumns struct {
	Alias     string `json:"alias,omitempty"`
	Condition string `json:"condition,omitempty"`
}

type DataSourcePolicyInfoProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospectionColumnTagValue struct {
	ColumnAlias string `json:"column_alias"`
	TagKey      string `json:"tag_key"`
}

type DataSourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospectionTagValue struct {
	TagKey string `json:"tag_key"`
}

type DataSourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospection struct {
	ColumnTagValue *DataSourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospectionColumnTagValue `json:"column_tag_value,omitempty"`
	TagValue       *DataSourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospectionTagValue       `json:"tag_value,omitempty"`
}

type DataSourcePolicyInfoRowFilterUsingFunctionArgExpression struct {
	TagIntrospection *DataSourcePolicyInfoRowFilterUsingFunctionArgExpressionTagIntrospection `json:"tag_introspection,omitempty"`
}

type DataSourcePolicyInfoRowFilterUsing struct {
	Alias                 string                                                   `json:"alias,omitempty"`
	Constant              string                                                   `json:"constant,omitempty"`
	FunctionArgExpression *DataSourcePolicyInfoRowFilterUsingFunctionArgExpression `json:"function_arg_expression,omitempty"`
}

type DataSourcePolicyInfoRowFilter struct {
	FunctionName string                               `json:"function_name"`
	Using        []DataSourcePolicyInfoRowFilterUsing `json:"using,omitempty"`
}

type DataSourcePolicyInfo struct {
	ColumnMask          *DataSourcePolicyInfoColumnMask     `json:"column_mask,omitempty"`
	Comment             string                              `json:"comment,omitempty"`
	CreatedAt           int                                 `json:"created_at,omitempty"`
	CreatedBy           string                              `json:"created_by,omitempty"`
	ExceptPrincipals    []string                            `json:"except_principals,omitempty"`
	ForSecurableType    string                              `json:"for_securable_type,omitempty"`
	Grant               *DataSourcePolicyInfoGrant          `json:"grant,omitempty"`
	Id                  string                              `json:"id,omitempty"`
	MatchColumns        []DataSourcePolicyInfoMatchColumns  `json:"match_columns,omitempty"`
	Name                string                              `json:"name"`
	OnSecurableFullname string                              `json:"on_securable_fullname"`
	OnSecurableType     string                              `json:"on_securable_type"`
	PolicyType          string                              `json:"policy_type,omitempty"`
	ProviderConfig      *DataSourcePolicyInfoProviderConfig `json:"provider_config,omitempty"`
	RowFilter           *DataSourcePolicyInfoRowFilter      `json:"row_filter,omitempty"`
	ToPrincipals        []string                            `json:"to_principals,omitempty"`
	UpdatedAt           int                                 `json:"updated_at,omitempty"`
	UpdatedBy           string                              `json:"updated_by,omitempty"`
	WhenCondition       string                              `json:"when_condition,omitempty"`
}
