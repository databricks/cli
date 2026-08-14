// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDisasterRecoveryStableUrlsStableUrls struct {
	FailoverGroupName  string `json:"failover_group_name,omitempty"`
	InitialWorkspaceId string `json:"initial_workspace_id,omitempty"`
	Name               string `json:"name"`
	Url                string `json:"url,omitempty"`
}

type DataSourceDisasterRecoveryStableUrls struct {
	PageSize   int                                              `json:"page_size,omitempty"`
	Parent     string                                           `json:"parent"`
	StableUrls []DataSourceDisasterRecoveryStableUrlsStableUrls `json:"stable_urls,omitempty"`
}
