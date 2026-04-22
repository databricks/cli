// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceCurrentConfigProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceCurrentConfig struct {
	AccountId      string                                 `json:"account_id,omitempty"`
	Api            string                                 `json:"api,omitempty"`
	AuthType       string                                 `json:"auth_type,omitempty"`
	Cloud          string                                 `json:"cloud,omitempty"`
	CloudType      string                                 `json:"cloud_type,omitempty"`
	Host           string                                 `json:"host,omitempty"`
	Id             string                                 `json:"id,omitempty"`
	IsAccount      bool                                   `json:"is_account,omitempty"`
	ProviderConfig *DataSourceCurrentConfigProviderConfig `json:"provider_config,omitempty"`
}
