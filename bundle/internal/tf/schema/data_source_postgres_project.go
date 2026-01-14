// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresProjectSpecDefaultEndpointSettings struct {
	AutoscalingLimitMaxCu  int               `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int               `json:"autoscaling_limit_min_cu,omitempty"`
	PgSettings             map[string]string `json:"pg_settings,omitempty"`
	PgbouncerSettings      map[string]string `json:"pgbouncer_settings,omitempty"`
	SuspendTimeoutDuration string            `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresProjectSpecSettings struct {
	EnableLogicalReplication bool `json:"enable_logical_replication,omitempty"`
}

type DataSourcePostgresProjectSpec struct {
	DefaultEndpointSettings  *DataSourcePostgresProjectSpecDefaultEndpointSettings `json:"default_endpoint_settings,omitempty"`
	DisplayName              string                                                `json:"display_name,omitempty"`
	HistoryRetentionDuration string                                                `json:"history_retention_duration,omitempty"`
	PgVersion                int                                                   `json:"pg_version,omitempty"`
	Settings                 *DataSourcePostgresProjectSpecSettings                `json:"settings,omitempty"`
}

type DataSourcePostgresProjectStatusDefaultEndpointSettings struct {
	AutoscalingLimitMaxCu  int               `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int               `json:"autoscaling_limit_min_cu,omitempty"`
	PgSettings             map[string]string `json:"pg_settings,omitempty"`
	PgbouncerSettings      map[string]string `json:"pgbouncer_settings,omitempty"`
	SuspendTimeoutDuration string            `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresProjectStatusSettings struct {
	EnableLogicalReplication bool `json:"enable_logical_replication,omitempty"`
}

type DataSourcePostgresProjectStatus struct {
	BranchLogicalSizeLimitBytes int                                                     `json:"branch_logical_size_limit_bytes,omitempty"`
	ComputeLastActiveTime       string                                                  `json:"compute_last_active_time,omitempty"`
	DefaultEndpointSettings     *DataSourcePostgresProjectStatusDefaultEndpointSettings `json:"default_endpoint_settings,omitempty"`
	DisplayName                 string                                                  `json:"display_name,omitempty"`
	HistoryRetentionDuration    string                                                  `json:"history_retention_duration,omitempty"`
	PgVersion                   int                                                     `json:"pg_version,omitempty"`
	Settings                    *DataSourcePostgresProjectStatusSettings                `json:"settings,omitempty"`
	SyntheticStorageSizeBytes   int                                                     `json:"synthetic_storage_size_bytes,omitempty"`
}

type DataSourcePostgresProject struct {
	CreateTime string                           `json:"create_time,omitempty"`
	Name       string                           `json:"name"`
	Spec       *DataSourcePostgresProjectSpec   `json:"spec,omitempty"`
	Status     *DataSourcePostgresProjectStatus `json:"status,omitempty"`
	Uid        string                           `json:"uid,omitempty"`
	UpdateTime string                           `json:"update_time,omitempty"`
}
