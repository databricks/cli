// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSharesProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceShares struct {
	ProviderConfig *DataSourceSharesProviderConfig `json:"provider_config,omitempty"`
	Shares         []string                        `json:"shares,omitempty"`
}
