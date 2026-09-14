// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePolicyInfosPoliciesColumnMaskUsingFunctionArgExpressionTagIntrospectionColumnTagValue struct {
	ColumnAlias string `json:"column_alias"`
	TagKey      string `json:"tag_key"`
}

type DataSourcePolicyInfosPoliciesColumnMaskUsingFunctionArgExpressionTagIntrospectionTagValue struct {
	TagKey string `json:"tag_key"`
}

type DataSourcePolicyInfosPoliciesColumnMaskUsingFunctionArgExpressionTagIntrospection struct {
	ColumnTagValue *DataSourcePolicyInfosPoliciesColumnMaskUsingFunctionArgExpressionTagIntrospectionColumnTagValue `json:"column_tag_value,omitempty"`
	TagValue       *DataSourcePolicyInfosPoliciesColumnMaskUsingFunctionArgExpressionTagIntrospectionTagValue       `json:"tag_value,omitempty"`
}

type DataSourcePolicyInfosPoliciesColumnMaskUsingFunctionArgExpression struct {
	TagIntrospection *DataSourcePolicyInfosPoliciesColumnMaskUsingFunctionArgExpressionTagIntrospection `json:"tag_introspection,omitempty"`
}

type DataSourcePolicyInfosPoliciesColumnMaskUsing struct {
	Alias                 string                                                             `json:"alias,omitempty"`
	Constant              string                                                             `json:"constant,omitempty"`
	FunctionArgExpression *DataSourcePolicyInfosPoliciesColumnMaskUsingFunctionArgExpression `json:"function_arg_expression,omitempty"`
}

type DataSourcePolicyInfosPoliciesColumnMask struct {
	FunctionName string                                         `json:"function_name"`
	OnColumn     string                                         `json:"on_column"`
	Using        []DataSourcePolicyInfosPoliciesColumnMaskUsing `json:"using,omitempty"`
}

type DataSourcePolicyInfosPoliciesGrant struct {
	Privileges []string `json:"privileges"`
}

type DataSourcePolicyInfosPoliciesMatchColumns struct {
	Alias     string `json:"alias,omitempty"`
	Condition string `json:"condition,omitempty"`
}

type DataSourcePolicyInfosPoliciesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePolicyInfosPoliciesRowFilterUsingFunctionArgExpressionTagIntrospectionColumnTagValue struct {
	ColumnAlias string `json:"column_alias"`
	TagKey      string `json:"tag_key"`
}

type DataSourcePolicyInfosPoliciesRowFilterUsingFunctionArgExpressionTagIntrospectionTagValue struct {
	TagKey string `json:"tag_key"`
}

type DataSourcePolicyInfosPoliciesRowFilterUsingFunctionArgExpressionTagIntrospection struct {
	ColumnTagValue *DataSourcePolicyInfosPoliciesRowFilterUsingFunctionArgExpressionTagIntrospectionColumnTagValue `json:"column_tag_value,omitempty"`
	TagValue       *DataSourcePolicyInfosPoliciesRowFilterUsingFunctionArgExpressionTagIntrospectionTagValue       `json:"tag_value,omitempty"`
}

type DataSourcePolicyInfosPoliciesRowFilterUsingFunctionArgExpression struct {
	TagIntrospection *DataSourcePolicyInfosPoliciesRowFilterUsingFunctionArgExpressionTagIntrospection `json:"tag_introspection,omitempty"`
}

type DataSourcePolicyInfosPoliciesRowFilterUsing struct {
	Alias                 string                                                            `json:"alias,omitempty"`
	Constant              string                                                            `json:"constant,omitempty"`
	FunctionArgExpression *DataSourcePolicyInfosPoliciesRowFilterUsingFunctionArgExpression `json:"function_arg_expression,omitempty"`
}

type DataSourcePolicyInfosPoliciesRowFilter struct {
	FunctionName string                                        `json:"function_name"`
	Using        []DataSourcePolicyInfosPoliciesRowFilterUsing `json:"using,omitempty"`
}

type DataSourcePolicyInfosPolicies struct {
	ColumnMask          *DataSourcePolicyInfosPoliciesColumnMask     `json:"column_mask,omitempty"`
	Comment             string                                       `json:"comment,omitempty"`
	CreatedAt           int                                          `json:"created_at,omitempty"`
	CreatedBy           string                                       `json:"created_by,omitempty"`
	ExceptPrincipals    []string                                     `json:"except_principals,omitempty"`
	ForSecurableType    string                                       `json:"for_securable_type,omitempty"`
	Grant               *DataSourcePolicyInfosPoliciesGrant          `json:"grant,omitempty"`
	Id                  string                                       `json:"id,omitempty"`
	MatchColumns        []DataSourcePolicyInfosPoliciesMatchColumns  `json:"match_columns,omitempty"`
	Name                string                                       `json:"name"`
	OnSecurableFullname string                                       `json:"on_securable_fullname"`
	OnSecurableType     string                                       `json:"on_securable_type"`
	PolicyType          string                                       `json:"policy_type,omitempty"`
	ProviderConfig      *DataSourcePolicyInfosPoliciesProviderConfig `json:"provider_config,omitempty"`
	RowFilter           *DataSourcePolicyInfosPoliciesRowFilter      `json:"row_filter,omitempty"`
	ToPrincipals        []string                                     `json:"to_principals,omitempty"`
	UpdatedAt           int                                          `json:"updated_at,omitempty"`
	UpdatedBy           string                                       `json:"updated_by,omitempty"`
	WhenCondition       string                                       `json:"when_condition,omitempty"`
}

type DataSourcePolicyInfosProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePolicyInfos struct {
	IncludeInherited    bool                                 `json:"include_inherited,omitempty"`
	MaxResults          int                                  `json:"max_results,omitempty"`
	OnSecurableFullname string                               `json:"on_securable_fullname"`
	OnSecurableType     string                               `json:"on_securable_type"`
	Policies            []DataSourcePolicyInfosPolicies      `json:"policies,omitempty"`
	ProviderConfig      *DataSourcePolicyInfosProviderConfig `json:"provider_config,omitempty"`
}
