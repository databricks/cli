// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceModelServingConfigAutoCaptureConfig struct {
	CatalogName     string `json:"catalog_name,omitempty"`
	Enabled         bool   `json:"enabled,omitempty"`
	SchemaName      string `json:"schema_name,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModelAi21LabsConfig struct {
	Ai21LabsApiKey string `json:"ai21labs_api_key"`
}

type ResourceModelServingConfigServedEntitiesExternalModelAmazonBedrockConfig struct {
	AwsAccessKeyId     string `json:"aws_access_key_id"`
	AwsRegion          string `json:"aws_region"`
	AwsSecretAccessKey string `json:"aws_secret_access_key"`
	BedrockProvider    string `json:"bedrock_provider"`
}

type ResourceModelServingConfigServedEntitiesExternalModelAnthropicConfig struct {
	AnthropicApiKey string `json:"anthropic_api_key"`
}

type ResourceModelServingConfigServedEntitiesExternalModelCohereConfig struct {
	CohereApiKey string `json:"cohere_api_key"`
}

type ResourceModelServingConfigServedEntitiesExternalModelDatabricksModelServingConfig struct {
	DatabricksApiToken     string `json:"databricks_api_token"`
	DatabricksWorkspaceUrl string `json:"databricks_workspace_url"`
}

type ResourceModelServingConfigServedEntitiesExternalModelOpenaiConfig struct {
	MicrosoftEntraClientId     string `json:"microsoft_entra_client_id,omitempty"`
	MicrosoftEntraClientSecret string `json:"microsoft_entra_client_secret,omitempty"`
	MicrosoftEntraTenantId     string `json:"microsoft_entra_tenant_id,omitempty"`
	OpenaiApiBase              string `json:"openai_api_base,omitempty"`
	OpenaiApiKey               string `json:"openai_api_key,omitempty"`
	OpenaiApiType              string `json:"openai_api_type,omitempty"`
	OpenaiApiVersion           string `json:"openai_api_version,omitempty"`
	OpenaiDeploymentName       string `json:"openai_deployment_name,omitempty"`
	OpenaiOrganization         string `json:"openai_organization,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModelPalmConfig struct {
	PalmApiKey string `json:"palm_api_key"`
}

type ResourceModelServingConfigServedEntitiesExternalModel struct {
	Name                         string                                                                             `json:"name"`
	Provider                     string                                                                             `json:"provider"`
	Task                         string                                                                             `json:"task"`
	Ai21LabsConfig               *ResourceModelServingConfigServedEntitiesExternalModelAi21LabsConfig               `json:"ai21labs_config,omitempty"`
	AmazonBedrockConfig          *ResourceModelServingConfigServedEntitiesExternalModelAmazonBedrockConfig          `json:"amazon_bedrock_config,omitempty"`
	AnthropicConfig              *ResourceModelServingConfigServedEntitiesExternalModelAnthropicConfig              `json:"anthropic_config,omitempty"`
	CohereConfig                 *ResourceModelServingConfigServedEntitiesExternalModelCohereConfig                 `json:"cohere_config,omitempty"`
	DatabricksModelServingConfig *ResourceModelServingConfigServedEntitiesExternalModelDatabricksModelServingConfig `json:"databricks_model_serving_config,omitempty"`
	OpenaiConfig                 *ResourceModelServingConfigServedEntitiesExternalModelOpenaiConfig                 `json:"openai_config,omitempty"`
	PalmConfig                   *ResourceModelServingConfigServedEntitiesExternalModelPalmConfig                   `json:"palm_config,omitempty"`
}

type ResourceModelServingConfigServedEntities struct {
	EntityName               string                                                 `json:"entity_name,omitempty"`
	EntityVersion            string                                                 `json:"entity_version,omitempty"`
	EnvironmentVars          map[string]string                                      `json:"environment_vars,omitempty"`
	InstanceProfileArn       string                                                 `json:"instance_profile_arn,omitempty"`
	MaxProvisionedThroughput int                                                    `json:"max_provisioned_throughput,omitempty"`
	MinProvisionedThroughput int                                                    `json:"min_provisioned_throughput,omitempty"`
	Name                     string                                                 `json:"name,omitempty"`
	ScaleToZeroEnabled       bool                                                   `json:"scale_to_zero_enabled,omitempty"`
	WorkloadSize             string                                                 `json:"workload_size,omitempty"`
	WorkloadType             string                                                 `json:"workload_type,omitempty"`
	ExternalModel            *ResourceModelServingConfigServedEntitiesExternalModel `json:"external_model,omitempty"`
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
	ServedEntities    []ResourceModelServingConfigServedEntities   `json:"served_entities,omitempty"`
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
	RouteOptimized    bool                             `json:"route_optimized,omitempty"`
	ServingEndpointId string                           `json:"serving_endpoint_id,omitempty"`
	Config            *ResourceModelServingConfig      `json:"config,omitempty"`
	RateLimits        []ResourceModelServingRateLimits `json:"rate_limits,omitempty"`
	Tags              []ResourceModelServingTags       `json:"tags,omitempty"`
}
