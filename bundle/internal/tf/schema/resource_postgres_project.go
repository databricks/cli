// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePostgresProjectProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourcePostgresProjectSpecCustomTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type ResourcePostgresProjectSpecDefaultEndpointSettings struct {
	AutoscalingLimitMaxCu  float64           `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64           `json:"autoscaling_limit_min_cu,omitempty"`
	NoSuspension           bool              `json:"no_suspension,omitempty"`
	PgSettings             map[string]string `json:"pg_settings,omitempty"`
	SuspendTimeoutDuration string            `json:"suspend_timeout_duration,omitempty"`
}

type ResourcePostgresProjectSpec struct {
	BudgetPolicyId           string                                              `json:"budget_policy_id,omitempty"`
	CustomTags               []ResourcePostgresProjectSpecCustomTags             `json:"custom_tags,omitempty"`
	DefaultEndpointSettings  *ResourcePostgresProjectSpecDefaultEndpointSettings `json:"default_endpoint_settings,omitempty"`
	DisplayName              string                                              `json:"display_name,omitempty"`
	HistoryRetentionDuration string                                              `json:"history_retention_duration,omitempty"`
	PgVersion                int                                                 `json:"pg_version,omitempty"`
}

type ResourcePostgresProjectStatusCustomTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type ResourcePostgresProjectStatusDefaultEndpointSettings struct {
	AutoscalingLimitMaxCu  float64           `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64           `json:"autoscaling_limit_min_cu,omitempty"`
	NoSuspension           bool              `json:"no_suspension,omitempty"`
	PgSettings             map[string]string `json:"pg_settings,omitempty"`
	SuspendTimeoutDuration string            `json:"suspend_timeout_duration,omitempty"`
}

type ResourcePostgresProjectStatus struct {
	BranchLogicalSizeLimitBytes int                                                   `json:"branch_logical_size_limit_bytes,omitempty"`
	BudgetPolicyId              string                                                `json:"budget_policy_id,omitempty"`
	CustomTags                  []ResourcePostgresProjectStatusCustomTags             `json:"custom_tags,omitempty"`
	DefaultEndpointSettings     *ResourcePostgresProjectStatusDefaultEndpointSettings `json:"default_endpoint_settings,omitempty"`
	DisplayName                 string                                                `json:"display_name,omitempty"`
	HistoryRetentionDuration    string                                                `json:"history_retention_duration,omitempty"`
	Owner                       string                                                `json:"owner,omitempty"`
	PgVersion                   int                                                   `json:"pg_version,omitempty"`
	SyntheticStorageSizeBytes   int                                                   `json:"synthetic_storage_size_bytes,omitempty"`
}

type ResourcePostgresProject struct {
	CreateTime     string                                 `json:"create_time,omitempty"`
	Name           string                                 `json:"name,omitempty"`
	ProjectId      string                                 `json:"project_id"`
	ProviderConfig *ResourcePostgresProjectProviderConfig `json:"provider_config,omitempty"`
	Spec           *ResourcePostgresProjectSpec           `json:"spec,omitempty"`
	Status         *ResourcePostgresProjectStatus         `json:"status,omitempty"`
	Uid            string                                 `json:"uid,omitempty"`
	UpdateTime     string                                 `json:"update_time,omitempty"`
}
