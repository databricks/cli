// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresProjectsProjectsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourcePostgresProjectsProjectsSpecCustomTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type DataSourcePostgresProjectsProjectsSpecDefaultEndpointSettings struct {
	AutoscalingLimitMaxCu  float64           `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64           `json:"autoscaling_limit_min_cu,omitempty"`
	NoSuspension           bool              `json:"no_suspension,omitempty"`
	PgSettings             map[string]string `json:"pg_settings,omitempty"`
	SuspendTimeoutDuration string            `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresProjectsProjectsSpec struct {
	BudgetPolicyId           string                                                         `json:"budget_policy_id,omitempty"`
	CustomTags               []DataSourcePostgresProjectsProjectsSpecCustomTags             `json:"custom_tags,omitempty"`
	DefaultEndpointSettings  *DataSourcePostgresProjectsProjectsSpecDefaultEndpointSettings `json:"default_endpoint_settings,omitempty"`
	DisplayName              string                                                         `json:"display_name,omitempty"`
	HistoryRetentionDuration string                                                         `json:"history_retention_duration,omitempty"`
	PgVersion                int                                                            `json:"pg_version,omitempty"`
}

type DataSourcePostgresProjectsProjectsStatusCustomTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type DataSourcePostgresProjectsProjectsStatusDefaultEndpointSettings struct {
	AutoscalingLimitMaxCu  float64           `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64           `json:"autoscaling_limit_min_cu,omitempty"`
	NoSuspension           bool              `json:"no_suspension,omitempty"`
	PgSettings             map[string]string `json:"pg_settings,omitempty"`
	SuspendTimeoutDuration string            `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresProjectsProjectsStatus struct {
	BranchLogicalSizeLimitBytes int                                                              `json:"branch_logical_size_limit_bytes,omitempty"`
	BudgetPolicyId              string                                                           `json:"budget_policy_id,omitempty"`
	CustomTags                  []DataSourcePostgresProjectsProjectsStatusCustomTags             `json:"custom_tags,omitempty"`
	DefaultEndpointSettings     *DataSourcePostgresProjectsProjectsStatusDefaultEndpointSettings `json:"default_endpoint_settings,omitempty"`
	DisplayName                 string                                                           `json:"display_name,omitempty"`
	HistoryRetentionDuration    string                                                           `json:"history_retention_duration,omitempty"`
	Owner                       string                                                           `json:"owner,omitempty"`
	PgVersion                   int                                                              `json:"pg_version,omitempty"`
	SyntheticStorageSizeBytes   int                                                              `json:"synthetic_storage_size_bytes,omitempty"`
}

type DataSourcePostgresProjectsProjects struct {
	CreateTime     string                                            `json:"create_time,omitempty"`
	Name           string                                            `json:"name"`
	ProviderConfig *DataSourcePostgresProjectsProjectsProviderConfig `json:"provider_config,omitempty"`
	Spec           *DataSourcePostgresProjectsProjectsSpec           `json:"spec,omitempty"`
	Status         *DataSourcePostgresProjectsProjectsStatus         `json:"status,omitempty"`
	Uid            string                                            `json:"uid,omitempty"`
	UpdateTime     string                                            `json:"update_time,omitempty"`
}

type DataSourcePostgresProjectsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourcePostgresProjects struct {
	PageSize       int                                       `json:"page_size,omitempty"`
	Projects       []DataSourcePostgresProjectsProjects      `json:"projects,omitempty"`
	ProviderConfig *DataSourcePostgresProjectsProviderConfig `json:"provider_config,omitempty"`
}
