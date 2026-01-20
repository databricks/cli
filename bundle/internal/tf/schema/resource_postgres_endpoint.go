// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePostgresEndpointSpecSettings struct {
	PgSettings        map[string]string `json:"pg_settings,omitempty"`
	PgbouncerSettings map[string]string `json:"pgbouncer_settings,omitempty"`
}

type ResourcePostgresEndpointSpec struct {
	AutoscalingLimitMaxCu  int                                   `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int                                   `json:"autoscaling_limit_min_cu,omitempty"`
	Disabled               bool                                  `json:"disabled,omitempty"`
	EndpointType           string                                `json:"endpoint_type"`
	PoolerMode             string                                `json:"pooler_mode,omitempty"`
	Settings               *ResourcePostgresEndpointSpecSettings `json:"settings,omitempty"`
	SuspendTimeoutDuration string                                `json:"suspend_timeout_duration,omitempty"`
}

type ResourcePostgresEndpointStatusSettings struct {
	PgSettings        map[string]string `json:"pg_settings,omitempty"`
	PgbouncerSettings map[string]string `json:"pgbouncer_settings,omitempty"`
}

type ResourcePostgresEndpointStatus struct {
	AutoscalingLimitMaxCu  int                                     `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int                                     `json:"autoscaling_limit_min_cu,omitempty"`
	CurrentState           string                                  `json:"current_state,omitempty"`
	Disabled               bool                                    `json:"disabled,omitempty"`
	EndpointType           string                                  `json:"endpoint_type,omitempty"`
	Host                   string                                  `json:"host,omitempty"`
	LastActiveTime         string                                  `json:"last_active_time,omitempty"`
	PendingState           string                                  `json:"pending_state,omitempty"`
	PoolerMode             string                                  `json:"pooler_mode,omitempty"`
	Settings               *ResourcePostgresEndpointStatusSettings `json:"settings,omitempty"`
	StartTime              string                                  `json:"start_time,omitempty"`
	SuspendTime            string                                  `json:"suspend_time,omitempty"`
	SuspendTimeoutDuration string                                  `json:"suspend_timeout_duration,omitempty"`
}

type ResourcePostgresEndpoint struct {
	CreateTime string                          `json:"create_time,omitempty"`
	EndpointId string                          `json:"endpoint_id,omitempty"`
	Name       string                          `json:"name,omitempty"`
	Parent     string                          `json:"parent"`
	Spec       *ResourcePostgresEndpointSpec   `json:"spec,omitempty"`
	Status     *ResourcePostgresEndpointStatus `json:"status,omitempty"`
	Uid        string                          `json:"uid,omitempty"`
	UpdateTime string                          `json:"update_time,omitempty"`
}
