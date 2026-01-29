// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresEndpointSpecSettings struct {
	PgSettings map[string]string `json:"pg_settings,omitempty"`
}

type DataSourcePostgresEndpointSpec struct {
	AutoscalingLimitMaxCu  float64                                     `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64                                     `json:"autoscaling_limit_min_cu,omitempty"`
	Disabled               bool                                    `json:"disabled,omitempty"`
	EndpointType           string                                  `json:"endpoint_type"`
	NoSuspension           bool                                    `json:"no_suspension,omitempty"`
	Settings               *DataSourcePostgresEndpointSpecSettings `json:"settings,omitempty"`
	SuspendTimeoutDuration string                                  `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresEndpointStatusHosts struct {
	Host string `json:"host,omitempty"`
}

type DataSourcePostgresEndpointStatusSettings struct {
	PgSettings map[string]string `json:"pg_settings,omitempty"`
}

type DataSourcePostgresEndpointStatus struct {
	AutoscalingLimitMaxCu  float64                                       `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64                                       `json:"autoscaling_limit_min_cu,omitempty"`
	CurrentState           string                                    `json:"current_state,omitempty"`
	Disabled               bool                                      `json:"disabled,omitempty"`
	EndpointType           string                                    `json:"endpoint_type,omitempty"`
	Hosts                  *DataSourcePostgresEndpointStatusHosts    `json:"hosts,omitempty"`
	PendingState           string                                    `json:"pending_state,omitempty"`
	Settings               *DataSourcePostgresEndpointStatusSettings `json:"settings,omitempty"`
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
