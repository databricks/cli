// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceTagPoliciesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceTagPoliciesTagPoliciesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceTagPoliciesTagPoliciesValues struct {
	Name string `json:"name"`
}

type DataSourceTagPoliciesTagPolicies struct {
	CreateTime     string                                          `json:"create_time,omitempty"`
	Description    string                                          `json:"description,omitempty"`
	Id             string                                          `json:"id,omitempty"`
	ProviderConfig *DataSourceTagPoliciesTagPoliciesProviderConfig `json:"provider_config,omitempty"`
	TagKey         string                                          `json:"tag_key"`
	UpdateTime     string                                          `json:"update_time,omitempty"`
	Values         []DataSourceTagPoliciesTagPoliciesValues        `json:"values,omitempty"`
}

type DataSourceTagPolicies struct {
	PageSize       int                                  `json:"page_size,omitempty"`
	ProviderConfig *DataSourceTagPoliciesProviderConfig `json:"provider_config,omitempty"`
	TagPolicies    []DataSourceTagPoliciesTagPolicies   `json:"tag_policies,omitempty"`
}
