// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePostgresEndpointSpecSettings struct {
	PgSettings map[string]string `json:"pg_settings,omitempty"`
}

type ResourcePostgresEndpointSpec struct {
	AutoscalingLimitMaxCu  float64                               `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64                               `json:"autoscaling_limit_min_cu,omitempty"`
	Disabled               bool                                  `json:"disabled,omitempty"`
	EndpointType           string                                `json:"endpoint_type"`
	NoSuspension           bool                                  `json:"no_suspension,omitempty"`
	Settings               *ResourcePostgresEndpointSpecSettings `json:"settings,omitempty"`
	SuspendTimeoutDuration string                                `json:"suspend_timeout_duration,omitempty"`
}

type ResourcePostgresEndpointStatusHosts struct {
	Host string `json:"host,omitempty"`
}

type ResourcePostgresEndpointStatusSettings struct {
	PgSettings map[string]string `json:"pg_settings,omitempty"`
}

type ResourcePostgresEndpointStatus struct {
	AutoscalingLimitMaxCu  float64                                 `json:"autoscaling_limit_max_cu,omitempty"`
	AutoscalingLimitMinCu  float64                                 `json:"autoscaling_limit_min_cu,omitempty"`
	CurrentState           string                                  `json:"current_state,omitempty"`
	Disabled               bool                                    `json:"disabled,omitempty"`
	EndpointType           string                                  `json:"endpoint_type,omitempty"`
	Hosts                  *ResourcePostgresEndpointStatusHosts    `json:"hosts,omitempty"`
	PendingState           string                                  `json:"pending_state,omitempty"`
	Settings               *ResourcePostgresEndpointStatusSettings `json:"settings,omitempty"`
	SuspendTimeoutDuration string                                  `json:"suspend_timeout_duration,omitempty"`
}

type ResourcePostgresEndpoint struct {
	CreateTime string                          `json:"create_time,omitempty"`
	EndpointId string                          `json:"endpoint_id"`
	Name       string                          `json:"name,omitempty"`
	Parent     string                          `json:"parent"`
	Spec       *ResourcePostgresEndpointSpec   `json:"spec,omitempty"`
	Status     *ResourcePostgresEndpointStatus `json:"status,omitempty"`
	Uid        string                          `json:"uid,omitempty"`
	UpdateTime string                          `json:"update_time,omitempty"`
}
