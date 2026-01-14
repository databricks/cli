// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresProjectsProjectsSpecDefaultEndpointSettings struct {
	AutoscalingLimitMaxCu  int               `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int               `json:"autoscaling_limit_min_cu,omitempty"`
	PgSettings             map[string]string `json:"pg_settings,omitempty"`
	PgbouncerSettings      map[string]string `json:"pgbouncer_settings,omitempty"`
	SuspendTimeoutDuration string            `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresProjectsProjectsSpecSettings struct {
	EnableLogicalReplication bool `json:"enable_logical_replication,omitempty"`
}

type DataSourcePostgresProjectsProjectsSpec struct {
	DefaultEndpointSettings  *DataSourcePostgresProjectsProjectsSpecDefaultEndpointSettings `json:"default_endpoint_settings,omitempty"`
	DisplayName              string                                                         `json:"display_name,omitempty"`
	HistoryRetentionDuration string                                                         `json:"history_retention_duration,omitempty"`
	PgVersion                int                                                            `json:"pg_version,omitempty"`
	Settings                 *DataSourcePostgresProjectsProjectsSpecSettings                `json:"settings,omitempty"`
}

type DataSourcePostgresProjectsProjectsStatusDefaultEndpointSettings struct {
	AutoscalingLimitMaxCu  int               `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int               `json:"autoscaling_limit_min_cu,omitempty"`
	PgSettings             map[string]string `json:"pg_settings,omitempty"`
	PgbouncerSettings      map[string]string `json:"pgbouncer_settings,omitempty"`
	SuspendTimeoutDuration string            `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresProjectsProjectsStatusSettings struct {
	EnableLogicalReplication bool `json:"enable_logical_replication,omitempty"`
}

type DataSourcePostgresProjectsProjectsStatus struct {
	BranchLogicalSizeLimitBytes int                                                              `json:"branch_logical_size_limit_bytes,omitempty"`
	ComputeLastActiveTime       string                                                           `json:"compute_last_active_time,omitempty"`
	DefaultEndpointSettings     *DataSourcePostgresProjectsProjectsStatusDefaultEndpointSettings `json:"default_endpoint_settings,omitempty"`
	DisplayName                 string                                                           `json:"display_name,omitempty"`
	HistoryRetentionDuration    string                                                           `json:"history_retention_duration,omitempty"`
	PgVersion                   int                                                              `json:"pg_version,omitempty"`
	Settings                    *DataSourcePostgresProjectsProjectsStatusSettings                `json:"settings,omitempty"`
	SyntheticStorageSizeBytes   int                                                              `json:"synthetic_storage_size_bytes,omitempty"`
}

type DataSourcePostgresProjectsProjects struct {
	CreateTime string                                    `json:"create_time,omitempty"`
	Name       string                                    `json:"name"`
	Spec       *DataSourcePostgresProjectsProjectsSpec   `json:"spec,omitempty"`
	Status     *DataSourcePostgresProjectsProjectsStatus `json:"status,omitempty"`
	Uid        string                                    `json:"uid,omitempty"`
	UpdateTime string                                    `json:"update_time,omitempty"`
}

type DataSourcePostgresProjects struct {
	PageSize int                                  `json:"page_size,omitempty"`
	Projects []DataSourcePostgresProjectsProjects `json:"projects,omitempty"`
}
