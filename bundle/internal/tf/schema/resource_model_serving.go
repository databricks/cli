// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceModelServingAiGatewayFallbackConfig struct {
	Enabled bool `json:"enabled"`
}

type ResourceModelServingAiGatewayGuardrailsInputPii struct {
	Behavior string `json:"behavior,omitempty"`
}

type ResourceModelServingAiGatewayGuardrailsInput struct {
	InvalidKeywords []string                                         `json:"invalid_keywords,omitempty"`
	Safety          bool                                             `json:"safety,omitempty"`
	ValidTopics     []string                                         `json:"valid_topics,omitempty"`
	Pii             *ResourceModelServingAiGatewayGuardrailsInputPii `json:"pii,omitempty"`
}

type ResourceModelServingAiGatewayGuardrailsOutputPii struct {
	Behavior string `json:"behavior,omitempty"`
}

type ResourceModelServingAiGatewayGuardrailsOutput struct {
	InvalidKeywords []string                                          `json:"invalid_keywords,omitempty"`
	Safety          bool                                              `json:"safety,omitempty"`
	ValidTopics     []string                                          `json:"valid_topics,omitempty"`
	Pii             *ResourceModelServingAiGatewayGuardrailsOutputPii `json:"pii,omitempty"`
}

type ResourceModelServingAiGatewayGuardrails struct {
	Input  *ResourceModelServingAiGatewayGuardrailsInput  `json:"input,omitempty"`
	Output *ResourceModelServingAiGatewayGuardrailsOutput `json:"output,omitempty"`
}

type ResourceModelServingAiGatewayInferenceTableConfig struct {
	CatalogName     string `json:"catalog_name,omitempty"`
	Enabled         bool   `json:"enabled,omitempty"`
	SchemaName      string `json:"schema_name,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type ResourceModelServingAiGatewayRateLimits struct {
	Calls         int    `json:"calls,omitempty"`
	Key           string `json:"key,omitempty"`
	Principal     string `json:"principal,omitempty"`
	RenewalPeriod string `json:"renewal_period"`
}

type ResourceModelServingAiGatewayUsageTrackingConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

type ResourceModelServingAiGateway struct {
	FallbackConfig       *ResourceModelServingAiGatewayFallbackConfig       `json:"fallback_config,omitempty"`
	Guardrails           *ResourceModelServingAiGatewayGuardrails           `json:"guardrails,omitempty"`
	InferenceTableConfig *ResourceModelServingAiGatewayInferenceTableConfig `json:"inference_table_config,omitempty"`
	RateLimits           []ResourceModelServingAiGatewayRateLimits          `json:"rate_limits,omitempty"`
	UsageTrackingConfig  *ResourceModelServingAiGatewayUsageTrackingConfig  `json:"usage_tracking_config,omitempty"`
}

type ResourceModelServingConfigAutoCaptureConfig struct {
	CatalogName     string `json:"catalog_name,omitempty"`
	Enabled         bool   `json:"enabled,omitempty"`
	SchemaName      string `json:"schema_name,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModelAi21LabsConfig struct {
	Ai21LabsApiKey          string `json:"ai21labs_api_key,omitempty"`
	Ai21LabsApiKeyPlaintext string `json:"ai21labs_api_key_plaintext,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModelAmazonBedrockConfig struct {
	AwsAccessKeyId              string `json:"aws_access_key_id,omitempty"`
	AwsAccessKeyIdPlaintext     string `json:"aws_access_key_id_plaintext,omitempty"`
	AwsRegion                   string `json:"aws_region"`
	AwsSecretAccessKey          string `json:"aws_secret_access_key,omitempty"`
	AwsSecretAccessKeyPlaintext string `json:"aws_secret_access_key_plaintext,omitempty"`
	BedrockProvider             string `json:"bedrock_provider"`
	InstanceProfileArn          string `json:"instance_profile_arn,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModelAnthropicConfig struct {
	AnthropicApiKey          string `json:"anthropic_api_key,omitempty"`
	AnthropicApiKeyPlaintext string `json:"anthropic_api_key_plaintext,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModelCohereConfig struct {
	CohereApiBase         string `json:"cohere_api_base,omitempty"`
	CohereApiKey          string `json:"cohere_api_key,omitempty"`
	CohereApiKeyPlaintext string `json:"cohere_api_key_plaintext,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModelCustomProviderConfigApiKeyAuth struct {
	Key            string `json:"key"`
	Value          string `json:"value,omitempty"`
	ValuePlaintext string `json:"value_plaintext,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModelCustomProviderConfigBearerTokenAuth struct {
	Token          string `json:"token,omitempty"`
	TokenPlaintext string `json:"token_plaintext,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModelCustomProviderConfig struct {
	CustomProviderUrl string                                                                                    `json:"custom_provider_url"`
	ApiKeyAuth        *ResourceModelServingConfigServedEntitiesExternalModelCustomProviderConfigApiKeyAuth      `json:"api_key_auth,omitempty"`
	BearerTokenAuth   *ResourceModelServingConfigServedEntitiesExternalModelCustomProviderConfigBearerTokenAuth `json:"bearer_token_auth,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModelDatabricksModelServingConfig struct {
	DatabricksApiToken          string `json:"databricks_api_token,omitempty"`
	DatabricksApiTokenPlaintext string `json:"databricks_api_token_plaintext,omitempty"`
	DatabricksWorkspaceUrl      string `json:"databricks_workspace_url"`
}

type ResourceModelServingConfigServedEntitiesExternalModelGoogleCloudVertexAiConfig struct {
	PrivateKey          string `json:"private_key,omitempty"`
	PrivateKeyPlaintext string `json:"private_key_plaintext,omitempty"`
	ProjectId           string `json:"project_id"`
	Region              string `json:"region"`
}

type ResourceModelServingConfigServedEntitiesExternalModelOpenaiConfig struct {
	MicrosoftEntraClientId              string `json:"microsoft_entra_client_id,omitempty"`
	MicrosoftEntraClientSecret          string `json:"microsoft_entra_client_secret,omitempty"`
	MicrosoftEntraClientSecretPlaintext string `json:"microsoft_entra_client_secret_plaintext,omitempty"`
	MicrosoftEntraTenantId              string `json:"microsoft_entra_tenant_id,omitempty"`
	OpenaiApiBase                       string `json:"openai_api_base,omitempty"`
	OpenaiApiKey                        string `json:"openai_api_key,omitempty"`
	OpenaiApiKeyPlaintext               string `json:"openai_api_key_plaintext,omitempty"`
	OpenaiApiType                       string `json:"openai_api_type,omitempty"`
	OpenaiApiVersion                    string `json:"openai_api_version,omitempty"`
	OpenaiDeploymentName                string `json:"openai_deployment_name,omitempty"`
	OpenaiOrganization                  string `json:"openai_organization,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModelPalmConfig struct {
	PalmApiKey          string `json:"palm_api_key,omitempty"`
	PalmApiKeyPlaintext string `json:"palm_api_key_plaintext,omitempty"`
}

type ResourceModelServingConfigServedEntitiesExternalModel struct {
	Name                         string                                                                             `json:"name"`
	Provider                     string                                                                             `json:"provider"`
	Task                         string                                                                             `json:"task"`
	Ai21LabsConfig               *ResourceModelServingConfigServedEntitiesExternalModelAi21LabsConfig               `json:"ai21labs_config,omitempty"`
	AmazonBedrockConfig          *ResourceModelServingConfigServedEntitiesExternalModelAmazonBedrockConfig          `json:"amazon_bedrock_config,omitempty"`
	AnthropicConfig              *ResourceModelServingConfigServedEntitiesExternalModelAnthropicConfig              `json:"anthropic_config,omitempty"`
	CohereConfig                 *ResourceModelServingConfigServedEntitiesExternalModelCohereConfig                 `json:"cohere_config,omitempty"`
	CustomProviderConfig         *ResourceModelServingConfigServedEntitiesExternalModelCustomProviderConfig         `json:"custom_provider_config,omitempty"`
	DatabricksModelServingConfig *ResourceModelServingConfigServedEntitiesExternalModelDatabricksModelServingConfig `json:"databricks_model_serving_config,omitempty"`
	GoogleCloudVertexAiConfig    *ResourceModelServingConfigServedEntitiesExternalModelGoogleCloudVertexAiConfig    `json:"google_cloud_vertex_ai_config,omitempty"`
	OpenaiConfig                 *ResourceModelServingConfigServedEntitiesExternalModelOpenaiConfig                 `json:"openai_config,omitempty"`
	PalmConfig                   *ResourceModelServingConfigServedEntitiesExternalModelPalmConfig                   `json:"palm_config,omitempty"`
}

type ResourceModelServingConfigServedEntities struct {
	EntityName                string                                                 `json:"entity_name,omitempty"`
	EntityVersion             string                                                 `json:"entity_version,omitempty"`
	EnvironmentVars           map[string]string                                      `json:"environment_vars,omitempty"`
	InstanceProfileArn        string                                                 `json:"instance_profile_arn,omitempty"`
	MaxProvisionedConcurrency int                                                    `json:"max_provisioned_concurrency,omitempty"`
	MaxProvisionedThroughput  int                                                    `json:"max_provisioned_throughput,omitempty"`
	MinProvisionedConcurrency int                                                    `json:"min_provisioned_concurrency,omitempty"`
	MinProvisionedThroughput  int                                                    `json:"min_provisioned_throughput,omitempty"`
	Name                      string                                                 `json:"name,omitempty"`
	ProvisionedModelUnits     int                                                    `json:"provisioned_model_units,omitempty"`
	ScaleToZeroEnabled        bool                                                   `json:"scale_to_zero_enabled,omitempty"`
	WorkloadSize              string                                                 `json:"workload_size,omitempty"`
	WorkloadType              string                                                 `json:"workload_type,omitempty"`
	ExternalModel             *ResourceModelServingConfigServedEntitiesExternalModel `json:"external_model,omitempty"`
}

type ResourceModelServingConfigServedModels struct {
	EnvironmentVars           map[string]string `json:"environment_vars,omitempty"`
	InstanceProfileArn        string            `json:"instance_profile_arn,omitempty"`
	MaxProvisionedConcurrency int               `json:"max_provisioned_concurrency,omitempty"`
	MaxProvisionedThroughput  int               `json:"max_provisioned_throughput,omitempty"`
	MinProvisionedConcurrency int               `json:"min_provisioned_concurrency,omitempty"`
	MinProvisionedThroughput  int               `json:"min_provisioned_throughput,omitempty"`
	ModelName                 string            `json:"model_name"`
	ModelVersion              string            `json:"model_version"`
	Name                      string            `json:"name,omitempty"`
	ProvisionedModelUnits     int               `json:"provisioned_model_units,omitempty"`
	ScaleToZeroEnabled        bool              `json:"scale_to_zero_enabled,omitempty"`
	WorkloadSize              string            `json:"workload_size,omitempty"`
	WorkloadType              string            `json:"workload_type,omitempty"`
}

type ResourceModelServingConfigTrafficConfigRoutes struct {
	ServedEntityName  string `json:"served_entity_name,omitempty"`
	ServedModelName   string `json:"served_model_name,omitempty"`
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
	BudgetPolicyId    string                           `json:"budget_policy_id,omitempty"`
	Description       string                           `json:"description,omitempty"`
	Id                string                           `json:"id,omitempty"`
	Name              string                           `json:"name"`
	RouteOptimized    bool                             `json:"route_optimized,omitempty"`
	ServingEndpointId string                           `json:"serving_endpoint_id,omitempty"`
	AiGateway         *ResourceModelServingAiGateway   `json:"ai_gateway,omitempty"`
	Config            *ResourceModelServingConfig      `json:"config,omitempty"`
	RateLimits        []ResourceModelServingRateLimits `json:"rate_limits,omitempty"`
	Tags              []ResourceModelServingTags       `json:"tags,omitempty"`
}
