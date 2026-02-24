// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresProjectProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourcePostgresProjectSpecCustomTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type DataSourcePostgresProjectSpecDefaultEndpointSettings struct {
	AutoscalingLimitMaxCu  float64           `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64           `json:"autoscaling_limit_min_cu,omitempty"`
	NoSuspension           bool              `json:"no_suspension,omitempty"`
	PgSettings             map[string]string `json:"pg_settings,omitempty"`
	SuspendTimeoutDuration string            `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresProjectSpec struct {
	BudgetPolicyId           string                                                `json:"budget_policy_id,omitempty"`
	CustomTags               []DataSourcePostgresProjectSpecCustomTags             `json:"custom_tags,omitempty"`
	DefaultEndpointSettings  *DataSourcePostgresProjectSpecDefaultEndpointSettings `json:"default_endpoint_settings,omitempty"`
	DisplayName              string                                                `json:"display_name,omitempty"`
	HistoryRetentionDuration string                                                `json:"history_retention_duration,omitempty"`
	PgVersion                int                                                   `json:"pg_version,omitempty"`
}

type DataSourcePostgresProjectStatusCustomTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type DataSourcePostgresProjectStatusDefaultEndpointSettings struct {
	AutoscalingLimitMaxCu  float64           `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64           `json:"autoscaling_limit_min_cu,omitempty"`
	NoSuspension           bool              `json:"no_suspension,omitempty"`
	PgSettings             map[string]string `json:"pg_settings,omitempty"`
	SuspendTimeoutDuration string            `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresProjectStatus struct {
	BranchLogicalSizeLimitBytes int                                                     `json:"branch_logical_size_limit_bytes,omitempty"`
	BudgetPolicyId              string                                                  `json:"budget_policy_id,omitempty"`
	CustomTags                  []DataSourcePostgresProjectStatusCustomTags             `json:"custom_tags,omitempty"`
	DefaultEndpointSettings     *DataSourcePostgresProjectStatusDefaultEndpointSettings `json:"default_endpoint_settings,omitempty"`
	DisplayName                 string                                                  `json:"display_name,omitempty"`
	HistoryRetentionDuration    string                                                  `json:"history_retention_duration,omitempty"`
	Owner                       string                                                  `json:"owner,omitempty"`
	PgVersion                   int                                                     `json:"pg_version,omitempty"`
	SyntheticStorageSizeBytes   int                                                     `json:"synthetic_storage_size_bytes,omitempty"`
}

type DataSourcePostgresProject struct {
	CreateTime     string                                   `json:"create_time,omitempty"`
	Name           string                                   `json:"name"`
	ProviderConfig *DataSourcePostgresProjectProviderConfig `json:"provider_config,omitempty"`
	Spec           *DataSourcePostgresProjectSpec           `json:"spec,omitempty"`
	Status         *DataSourcePostgresProjectStatus         `json:"status,omitempty"`
	Uid            string                                   `json:"uid,omitempty"`
	UpdateTime     string                                   `json:"update_time,omitempty"`
}
