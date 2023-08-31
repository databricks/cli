// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceModelServingConfigServedModels struct {
	EnvironmentVars    map[string]string `json:"environment_vars,omitempty"`
	InstanceProfileArn string            `json:"instance_profile_arn,omitempty"`
	ModelName          string            `json:"model_name"`
	ModelVersion       string            `json:"model_version"`
	Name               string            `json:"name,omitempty"`
	ScaleToZeroEnabled bool              `json:"scale_to_zero_enabled,omitempty"`
	WorkloadSize       string            `json:"workload_size"`
}

type ResourceModelServingConfigTrafficConfigRoutes struct {
	ServedModelName   string `json:"served_model_name"`
	TrafficPercentage int    `json:"traffic_percentage"`
}

type ResourceModelServingConfigTrafficConfig struct {
	Routes []ResourceModelServingConfigTrafficConfigRoutes `json:"routes,omitempty"`
}

type ResourceModelServingConfig struct {
	ServedModels  []ResourceModelServingConfigServedModels `json:"served_models,omitempty"`
	TrafficConfig *ResourceModelServingConfigTrafficConfig `json:"traffic_config,omitempty"`
}

type ResourceModelServing struct {
	Id                string                      `json:"id,omitempty"`
	Name              string                      `json:"name"`
	ServingEndpointId string                      `json:"serving_endpoint_id,omitempty"`
	Config            *ResourceModelServingConfig `json:"config,omitempty"`
}
