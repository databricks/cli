// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresEndpointsEndpointsSpecSettings struct {
	PgSettings map[string]string `json:"pg_settings,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsSpec struct {
	AutoscalingLimitMaxCu  int                                               `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int                                               `json:"autoscaling_limit_min_cu,omitempty"`
	Disabled               bool                                              `json:"disabled,omitempty"`
	EndpointType           string                                            `json:"endpoint_type"`
	NoSuspension           bool                                              `json:"no_suspension,omitempty"`
	Settings               *DataSourcePostgresEndpointsEndpointsSpecSettings `json:"settings,omitempty"`
	SuspendTimeoutDuration string                                            `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsStatusHosts struct {
	Host string `json:"host,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsStatusSettings struct {
	PgSettings map[string]string `json:"pg_settings,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsStatus struct {
	AutoscalingLimitMaxCu  int                                                 `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  int                                                 `json:"autoscaling_limit_min_cu,omitempty"`
	CurrentState           string                                              `json:"current_state,omitempty"`
	Disabled               bool                                                `json:"disabled,omitempty"`
	EndpointType           string                                              `json:"endpoint_type,omitempty"`
	Hosts                  *DataSourcePostgresEndpointsEndpointsStatusHosts    `json:"hosts,omitempty"`
	PendingState           string                                              `json:"pending_state,omitempty"`
	Settings               *DataSourcePostgresEndpointsEndpointsStatusSettings `json:"settings,omitempty"`
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
