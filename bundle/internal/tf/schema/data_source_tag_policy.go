// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceTagPolicyProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceTagPolicyValues struct {
	Name string `json:"name"`
}

type DataSourceTagPolicy struct {
	CreateTime     string                             `json:"create_time,omitempty"`
	Description    string                             `json:"description,omitempty"`
	Id             string                             `json:"id,omitempty"`
	ProviderConfig *DataSourceTagPolicyProviderConfig `json:"provider_config,omitempty"`
	TagKey         string                             `json:"tag_key"`
	UpdateTime     string                             `json:"update_time,omitempty"`
	Values         []DataSourceTagPolicyValues        `json:"values,omitempty"`
}
