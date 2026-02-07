// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceCurrentUserProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceCurrentUser struct {
	AclPrincipalId string                               `json:"acl_principal_id,omitempty"`
	Alphanumeric   string                               `json:"alphanumeric,omitempty"`
	ExternalId     string                               `json:"external_id,omitempty"`
	Home           string                               `json:"home,omitempty"`
	Id             string                               `json:"id,omitempty"`
	Repos          string                               `json:"repos,omitempty"`
	UserName       string                               `json:"user_name,omitempty"`
	WorkspaceUrl   string                               `json:"workspace_url,omitempty"`
	ProviderConfig *DataSourceCurrentUserProviderConfig `json:"provider_config,omitempty"`
}
