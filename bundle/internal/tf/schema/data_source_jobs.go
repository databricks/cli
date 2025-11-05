// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceJobsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceJobs struct {
	Id              string                        `json:"id,omitempty"`
	Ids             map[string]string             `json:"ids,omitempty"`
	JobNameContains string                        `json:"job_name_contains,omitempty"`
	Key             string                        `json:"key,omitempty"`
	ProviderConfig  *DataSourceJobsProviderConfig `json:"provider_config,omitempty"`
}
