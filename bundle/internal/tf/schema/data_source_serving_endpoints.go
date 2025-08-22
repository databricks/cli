// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceServingEndpointsEndpointsAiGatewayFallbackConfig struct {
	Enabled bool `json:"enabled"`
}

type DataSourceServingEndpointsEndpointsAiGatewayGuardrailsInputPii struct {
	Behavior string `json:"behavior,omitempty"`
}

type DataSourceServingEndpointsEndpointsAiGatewayGuardrailsInput struct {
	InvalidKeywords []string                                                         `json:"invalid_keywords,omitempty"`
	Pii             []DataSourceServingEndpointsEndpointsAiGatewayGuardrailsInputPii `json:"pii,omitempty"`
	Safety          bool                                                             `json:"safety,omitempty"`
	ValidTopics     []string                                                         `json:"valid_topics,omitempty"`
}

type DataSourceServingEndpointsEndpointsAiGatewayGuardrailsOutputPii struct {
	Behavior string `json:"behavior,omitempty"`
}

type DataSourceServingEndpointsEndpointsAiGatewayGuardrailsOutput struct {
	InvalidKeywords []string                                                          `json:"invalid_keywords,omitempty"`
	Pii             []DataSourceServingEndpointsEndpointsAiGatewayGuardrailsOutputPii `json:"pii,omitempty"`
	Safety          bool                                                              `json:"safety,omitempty"`
	ValidTopics     []string                                                          `json:"valid_topics,omitempty"`
}

type DataSourceServingEndpointsEndpointsAiGatewayGuardrails struct {
	Input  []DataSourceServingEndpointsEndpointsAiGatewayGuardrailsInput  `json:"input,omitempty"`
	Output []DataSourceServingEndpointsEndpointsAiGatewayGuardrailsOutput `json:"output,omitempty"`
}

type DataSourceServingEndpointsEndpointsAiGatewayInferenceTableConfig struct {
	CatalogName     string `json:"catalog_name,omitempty"`
	Enabled         bool   `json:"enabled,omitempty"`
	SchemaName      string `json:"schema_name,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type DataSourceServingEndpointsEndpointsAiGatewayRateLimits struct {
	Calls         int    `json:"calls,omitempty"`
	Key           string `json:"key,omitempty"`
	Principal     string `json:"principal,omitempty"`
	RenewalPeriod string `json:"renewal_period"`
}

type DataSourceServingEndpointsEndpointsAiGatewayUsageTrackingConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

type DataSourceServingEndpointsEndpointsAiGateway struct {
	FallbackConfig       []DataSourceServingEndpointsEndpointsAiGatewayFallbackConfig       `json:"fallback_config,omitempty"`
	Guardrails           []DataSourceServingEndpointsEndpointsAiGatewayGuardrails           `json:"guardrails,omitempty"`
	InferenceTableConfig []DataSourceServingEndpointsEndpointsAiGatewayInferenceTableConfig `json:"inference_table_config,omitempty"`
	RateLimits           []DataSourceServingEndpointsEndpointsAiGatewayRateLimits           `json:"rate_limits,omitempty"`
	UsageTrackingConfig  []DataSourceServingEndpointsEndpointsAiGatewayUsageTrackingConfig  `json:"usage_tracking_config,omitempty"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelAi21LabsConfig struct {
	Ai21LabsApiKey          string `json:"ai21labs_api_key,omitempty"`
	Ai21LabsApiKeyPlaintext string `json:"ai21labs_api_key_plaintext,omitempty"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelAmazonBedrockConfig struct {
	AwsAccessKeyId              string `json:"aws_access_key_id,omitempty"`
	AwsAccessKeyIdPlaintext     string `json:"aws_access_key_id_plaintext,omitempty"`
	AwsRegion                   string `json:"aws_region"`
	AwsSecretAccessKey          string `json:"aws_secret_access_key,omitempty"`
	AwsSecretAccessKeyPlaintext string `json:"aws_secret_access_key_plaintext,omitempty"`
	BedrockProvider             string `json:"bedrock_provider"`
	InstanceProfileArn          string `json:"instance_profile_arn,omitempty"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelAnthropicConfig struct {
	AnthropicApiKey          string `json:"anthropic_api_key,omitempty"`
	AnthropicApiKeyPlaintext string `json:"anthropic_api_key_plaintext,omitempty"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelCohereConfig struct {
	CohereApiBase         string `json:"cohere_api_base,omitempty"`
	CohereApiKey          string `json:"cohere_api_key,omitempty"`
	CohereApiKeyPlaintext string `json:"cohere_api_key_plaintext,omitempty"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelCustomProviderConfigApiKeyAuth struct {
	Key            string `json:"key"`
	Value          string `json:"value,omitempty"`
	ValuePlaintext string `json:"value_plaintext,omitempty"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelCustomProviderConfigBearerTokenAuth struct {
	Token          string `json:"token,omitempty"`
	TokenPlaintext string `json:"token_plaintext,omitempty"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelCustomProviderConfig struct {
	ApiKeyAuth        []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelCustomProviderConfigApiKeyAuth      `json:"api_key_auth,omitempty"`
	BearerTokenAuth   []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelCustomProviderConfigBearerTokenAuth `json:"bearer_token_auth,omitempty"`
	CustomProviderUrl string                                                                                                    `json:"custom_provider_url"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelDatabricksModelServingConfig struct {
	DatabricksApiToken          string `json:"databricks_api_token,omitempty"`
	DatabricksApiTokenPlaintext string `json:"databricks_api_token_plaintext,omitempty"`
	DatabricksWorkspaceUrl      string `json:"databricks_workspace_url"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelGoogleCloudVertexAiConfig struct {
	PrivateKey          string `json:"private_key,omitempty"`
	PrivateKeyPlaintext string `json:"private_key_plaintext,omitempty"`
	ProjectId           string `json:"project_id"`
	Region              string `json:"region"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelOpenaiConfig struct {
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

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelPalmConfig struct {
	PalmApiKey          string `json:"palm_api_key,omitempty"`
	PalmApiKeyPlaintext string `json:"palm_api_key_plaintext,omitempty"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModel struct {
	Ai21LabsConfig               []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelAi21LabsConfig               `json:"ai21labs_config,omitempty"`
	AmazonBedrockConfig          []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelAmazonBedrockConfig          `json:"amazon_bedrock_config,omitempty"`
	AnthropicConfig              []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelAnthropicConfig              `json:"anthropic_config,omitempty"`
	CohereConfig                 []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelCohereConfig                 `json:"cohere_config,omitempty"`
	CustomProviderConfig         []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelCustomProviderConfig         `json:"custom_provider_config,omitempty"`
	DatabricksModelServingConfig []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelDatabricksModelServingConfig `json:"databricks_model_serving_config,omitempty"`
	GoogleCloudVertexAiConfig    []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelGoogleCloudVertexAiConfig    `json:"google_cloud_vertex_ai_config,omitempty"`
	Name                         string                                                                                             `json:"name"`
	OpenaiConfig                 []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelOpenaiConfig                 `json:"openai_config,omitempty"`
	PalmConfig                   []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModelPalmConfig                   `json:"palm_config,omitempty"`
	Provider                     string                                                                                             `json:"provider"`
	Task                         string                                                                                             `json:"task"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntitiesFoundationModel struct {
	Description string `json:"description,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Docs        string `json:"docs,omitempty"`
	Name        string `json:"name,omitempty"`
}

type DataSourceServingEndpointsEndpointsConfigServedEntities struct {
	EntityName      string                                                                   `json:"entity_name,omitempty"`
	EntityVersion   string                                                                   `json:"entity_version,omitempty"`
	ExternalModel   []DataSourceServingEndpointsEndpointsConfigServedEntitiesExternalModel   `json:"external_model,omitempty"`
	FoundationModel []DataSourceServingEndpointsEndpointsConfigServedEntitiesFoundationModel `json:"foundation_model,omitempty"`
	Name            string                                                                   `json:"name,omitempty"`
}

type DataSourceServingEndpointsEndpointsConfigServedModels struct {
	ModelName    string `json:"model_name,omitempty"`
	ModelVersion string `json:"model_version,omitempty"`
	Name         string `json:"name,omitempty"`
}

type DataSourceServingEndpointsEndpointsConfig struct {
	ServedEntities []DataSourceServingEndpointsEndpointsConfigServedEntities `json:"served_entities,omitempty"`
	ServedModels   []DataSourceServingEndpointsEndpointsConfigServedModels   `json:"served_models,omitempty"`
}

type DataSourceServingEndpointsEndpointsState struct {
	ConfigUpdate string `json:"config_update,omitempty"`
	Ready        string `json:"ready,omitempty"`
}

type DataSourceServingEndpointsEndpointsTags struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type DataSourceServingEndpointsEndpoints struct {
	AiGateway            []DataSourceServingEndpointsEndpointsAiGateway `json:"ai_gateway,omitempty"`
	BudgetPolicyId       string                                         `json:"budget_policy_id,omitempty"`
	Config               []DataSourceServingEndpointsEndpointsConfig    `json:"config,omitempty"`
	CreationTimestamp    int                                            `json:"creation_timestamp,omitempty"`
	Creator              string                                         `json:"creator,omitempty"`
	Description          string                                         `json:"description,omitempty"`
	Id                   string                                         `json:"id,omitempty"`
	LastUpdatedTimestamp int                                            `json:"last_updated_timestamp,omitempty"`
	Name                 string                                         `json:"name,omitempty"`
	State                []DataSourceServingEndpointsEndpointsState     `json:"state,omitempty"`
	Tags                 []DataSourceServingEndpointsEndpointsTags      `json:"tags,omitempty"`
	Task                 string                                         `json:"task,omitempty"`
}

type DataSourceServingEndpoints struct {
	Endpoints []DataSourceServingEndpointsEndpoints `json:"endpoints,omitempty"`
}
