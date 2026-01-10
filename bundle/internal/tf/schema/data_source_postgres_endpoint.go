// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresEndpointSpecSettings struct {
	PgSettings        map[string]string `json:"pg_settings,omitempty"`
	PgbouncerSettings map[string]string `json:"pgbouncer_settings,omitempty"`
}

type DataSourcePostgresEndpointSpec struct {
	AutoscalingLimitMaxCu  int                                     `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int                                     `json:"autoscaling_limit_min_cu,omitempty"`
	Disabled               bool                                    `json:"disabled,omitempty"`
	EndpointType           string                                  `json:"endpoint_type"`
	PoolerMode             string                                  `json:"pooler_mode,omitempty"`
	Settings               *DataSourcePostgresEndpointSpecSettings `json:"settings,omitempty"`
	SuspendTimeoutDuration string                                  `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresEndpointStatusSettings struct {
	PgSettings        map[string]string `json:"pg_settings,omitempty"`
	PgbouncerSettings map[string]string `json:"pgbouncer_settings,omitempty"`
}

type DataSourcePostgresEndpointStatus struct {
	AutoscalingLimitMaxCu  int                                       `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int                                       `json:"autoscaling_limit_min_cu,omitempty"`
	CurrentState           string                                    `json:"current_state,omitempty"`
	Disabled               bool                                      `json:"disabled,omitempty"`
	EndpointType           string                                    `json:"endpoint_type,omitempty"`
	Host                   string                                    `json:"host,omitempty"`
	LastActiveTime         string                                    `json:"last_active_time,omitempty"`
	PendingState           string                                    `json:"pending_state,omitempty"`
	PoolerMode             string                                    `json:"pooler_mode,omitempty"`
	Settings               *DataSourcePostgresEndpointStatusSettings `json:"settings,omitempty"`
	StartTime              string                                    `json:"start_time,omitempty"`
	SuspendTime            string                                    `json:"suspend_time,omitempty"`
	SuspendTimeoutDuration string                                    `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresEndpoint struct {
	CreateTime string                            `json:"create_time,omitempty"`
	Name       string                            `json:"name"`
	Parent     string                            `json:"parent,omitempty"`
	Spec       *DataSourcePostgresEndpointSpec   `json:"spec,omitempty"`
	Status     *DataSourcePostgresEndpointStatus `json:"status,omitempty"`
	Uid        string                            `json:"uid,omitempty"`
	UpdateTime string                            `json:"update_time,omitempty"`
}
