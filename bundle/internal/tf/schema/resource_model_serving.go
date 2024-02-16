// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceModelServingConfigAutoCaptureConfig struct {
	CatalogName     string `json:"catalog_name,omitempty"`
	Enabled         bool   `json:"enabled,omitempty"`
	SchemaName      string `json:"schema_name,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type ResourceModelServingConfigServedModels struct {
	EnvironmentVars    map[string]string `json:"environment_vars,omitempty"`
	InstanceProfileArn string            `json:"instance_profile_arn,omitempty"`
	ModelName          string            `json:"model_name"`
	ModelVersion       string            `json:"model_version"`
	Name               string            `json:"name,omitempty"`
	ScaleToZeroEnabled bool              `json:"scale_to_zero_enabled,omitempty"`
	WorkloadSize       string            `json:"workload_size"`
	WorkloadType       string            `json:"workload_type,omitempty"`
}

type ResourceModelServingConfigTrafficConfigRoutes struct {
	ServedModelName   string `json:"served_model_name"`
	TrafficPercentage int    `json:"traffic_percentage"`
}

type ResourceModelServingConfigTrafficConfig struct {
	Routes []ResourceModelServingConfigTrafficConfigRoutes `json:"routes,omitempty"`
}

type ResourceModelServingConfig struct {
	AutoCaptureConfig *ResourceModelServingConfigAutoCaptureConfig `json:"auto_capture_config,omitempty"`
	ServedModels      []ResourceModelServingConfigServedModels     `json:"served_models,omitempty"`
	TrafficConfig     *ResourceModelServingConfigTrafficConfig     `json:"traffic_config,omitempty"`
}

type ResourceModelServingRateLimits struct {
	Calls         int    `json:"calls"`
	Key           string `json:"key,omitempty"`
	RenewalPeriod string `json:"renewal_period"`
}

type ResourceModelServingTags struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type ResourceModelServing struct {
	Id                string                           `json:"id,omitempty"`
	Name              string                           `json:"name"`
	ServingEndpointId string                           `json:"serving_endpoint_id,omitempty"`
	Config            *ResourceModelServingConfig      `json:"config,omitempty"`
	RateLimits        []ResourceModelServingRateLimits `json:"rate_limits,omitempty"`
	Tags              []ResourceModelServingTags       `json:"tags,omitempty"`
}
