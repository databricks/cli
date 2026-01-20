// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresEndpointsEndpointsSpecSettings struct {
	PgSettings        map[string]string `json:"pg_settings,omitempty"`
	PgbouncerSettings map[string]string `json:"pgbouncer_settings,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsSpec struct {
	AutoscalingLimitMaxCu  int                                               `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int                                               `json:"autoscaling_limit_min_cu,omitempty"`
	Disabled               bool                                              `json:"disabled,omitempty"`
	EndpointType           string                                            `json:"endpoint_type"`
	PoolerMode             string                                            `json:"pooler_mode,omitempty"`
	Settings               *DataSourcePostgresEndpointsEndpointsSpecSettings `json:"settings,omitempty"`
	SuspendTimeoutDuration string                                            `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsStatusSettings struct {
	PgSettings        map[string]string `json:"pg_settings,omitempty"`
	PgbouncerSettings map[string]string `json:"pgbouncer_settings,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsStatus struct {
	AutoscalingLimitMaxCu  int                                                 `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int                                                 `json:"autoscaling_limit_min_cu,omitempty"`
	CurrentState           string                                              `json:"current_state,omitempty"`
	Disabled               bool                                                `json:"disabled,omitempty"`
	EndpointType           string                                              `json:"endpoint_type,omitempty"`
	Host                   string                                              `json:"host,omitempty"`
	LastActiveTime         string                                              `json:"last_active_time,omitempty"`
	PendingState           string                                              `json:"pending_state,omitempty"`
	PoolerMode             string                                              `json:"pooler_mode,omitempty"`
	Settings               *DataSourcePostgresEndpointsEndpointsStatusSettings `json:"settings,omitempty"`
	StartTime              string                                              `json:"start_time,omitempty"`
	SuspendTime            string                                              `json:"suspend_time,omitempty"`
	SuspendTimeoutDuration string                                              `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresEndpointsEndpoints struct {
	CreateTime string                                      `json:"create_time,omitempty"`
	Name       string                                      `json:"name"`
	Parent     string                                      `json:"parent,omitempty"`
	Spec       *DataSourcePostgresEndpointsEndpointsSpec   `json:"spec,omitempty"`
	Status     *DataSourcePostgresEndpointsEndpointsStatus `json:"status,omitempty"`
	Uid        string                                      `json:"uid,omitempty"`
	UpdateTime string                                      `json:"update_time,omitempty"`
}

type DataSourcePostgresEndpoints struct {
	Endpoints []DataSourcePostgresEndpointsEndpoints `json:"endpoints,omitempty"`
	PageSize  int                                    `json:"page_size,omitempty"`
	Parent    string                                 `json:"parent"`
}
