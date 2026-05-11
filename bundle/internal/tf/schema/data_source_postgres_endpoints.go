// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresEndpointsEndpointsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsSpecGroup struct {
	EnableReadableSecondaries bool `json:"enable_readable_secondaries,omitempty"`
	Max                       int  `json:"max"`
	Min                       int  `json:"min"`
}

type DataSourcePostgresEndpointsEndpointsSpecSettings struct {
	PgSettings map[string]string `json:"pg_settings,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsSpec struct {
	AutoscalingLimitMaxCu  float64                                           `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64                                           `json:"autoscaling_limit_min_cu,omitempty"`
	Disabled               bool                                              `json:"disabled,omitempty"`
	EndpointType           string                                            `json:"endpoint_type"`
	Group                  *DataSourcePostgresEndpointsEndpointsSpecGroup    `json:"group,omitempty"`
	NoSuspension           bool                                              `json:"no_suspension,omitempty"`
	Settings               *DataSourcePostgresEndpointsEndpointsSpecSettings `json:"settings,omitempty"`
	SuspendTimeoutDuration string                                            `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsStatusGroup struct {
	EnableReadableSecondaries bool `json:"enable_readable_secondaries,omitempty"`
	Max                       int  `json:"max"`
	Min                       int  `json:"min"`
}

type DataSourcePostgresEndpointsEndpointsStatusHosts struct {
	Host         string `json:"host,omitempty"`
	ReadOnlyHost string `json:"read_only_host,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsStatusSettings struct {
	PgSettings map[string]string `json:"pg_settings,omitempty"`
}

type DataSourcePostgresEndpointsEndpointsStatus struct {
	AutoscalingLimitMaxCu  float64                                             `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64                                             `json:"autoscaling_limit_min_cu,omitempty"`
	CurrentState           string                                              `json:"current_state,omitempty"`
	Disabled               bool                                                `json:"disabled,omitempty"`
	EndpointId             string                                              `json:"endpoint_id,omitempty"`
	EndpointType           string                                              `json:"endpoint_type,omitempty"`
	Group                  *DataSourcePostgresEndpointsEndpointsStatusGroup    `json:"group,omitempty"`
	Hosts                  *DataSourcePostgresEndpointsEndpointsStatusHosts    `json:"hosts,omitempty"`
	PendingState           string                                              `json:"pending_state,omitempty"`
	Settings               *DataSourcePostgresEndpointsEndpointsStatusSettings `json:"settings,omitempty"`
	SuspendTimeoutDuration string                                              `json:"suspend_timeout_duration,omitempty"`
}

type DataSourcePostgresEndpointsEndpoints struct {
	CreateTime     string                                              `json:"create_time,omitempty"`
	Name           string                                              `json:"name"`
	Parent         string                                              `json:"parent,omitempty"`
	ProviderConfig *DataSourcePostgresEndpointsEndpointsProviderConfig `json:"provider_config,omitempty"`
	Spec           *DataSourcePostgresEndpointsEndpointsSpec           `json:"spec,omitempty"`
	Status         *DataSourcePostgresEndpointsEndpointsStatus         `json:"status,omitempty"`
	Uid            string                                              `json:"uid,omitempty"`
	UpdateTime     string                                              `json:"update_time,omitempty"`
}

type DataSourcePostgresEndpointsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresEndpoints struct {
	Endpoints      []DataSourcePostgresEndpointsEndpoints     `json:"endpoints,omitempty"`
	PageSize       int                                        `json:"page_size,omitempty"`
	Parent         string                                     `json:"parent"`
	ProviderConfig *DataSourcePostgresEndpointsProviderConfig `json:"provider_config,omitempty"`
}
