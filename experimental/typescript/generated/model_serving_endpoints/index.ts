/**
 * ModelServingEndpoint resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface ModelServingEndpointParams {
  /**
   * The AI Gateway configuration for the serving endpoint. NOTE: External model, provisioned throughput, and pay-per-token endpoints are fully supported; agent endpoints currently only support inference tables.
   */
  ai_gateway?: VariableOr<AiGatewayConfig>;
  /**
   * The budget policy to be applied to the serving endpoint.
   */
  budget_policy_id?: VariableOr<string>;
  /**
   * The core config of the serving endpoint.
   */
  config?: VariableOr<EndpointCoreConfigInput>;
  description?: VariableOr<string>;
  /**
   * Email notification settings.
   */
  email_notifications?: VariableOr<EmailNotifications>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * The name of the serving endpoint. This field is required and must be unique across a Databricks workspace.
   * An endpoint name can consist of alphanumeric characters, dashes, and underscores.
   */
  name: VariableOr<string>;
  permissions?: VariableOr<ModelServingEndpointPermission[]>;
  /**
   * Rate limits to be applied to the serving endpoint. NOTE: this field is deprecated, please use AI Gateway to manage rate limits.
   * @deprecated
   */
  rate_limits?: VariableOr<RateLimit[]>;
  /**
   * Enable route optimization for the serving endpoint.
   */
  route_optimized?: VariableOr<boolean>;
  /**
   * Tags to be attached to the serving endpoint and automatically propagated to billing logs.
   */
  tags?: VariableOr<EndpointTag[]>;
}

export class ModelServingEndpoint extends Resource<ModelServingEndpointParams> {
  constructor(name: string, params: ModelServingEndpointParams) {
    super(name, params, "model_serving_endpoints");
  }
}

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

export interface ModelServingEndpointPermission {
  group_name?: VariableOr<string>;
  level: VariableOr<ModelServingEndpointPermissionLevel>;
  service_principal_name?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export type ModelServingEndpointPermissionLevel =
  | "CAN_MANAGE"
  | "CAN_QUERY"
  | "CAN_VIEW";

export interface Ai21LabsConfig {
  /**
   * The Databricks secret key reference for an AI21 Labs API key. If you
   * prefer to paste your API key directly, see `ai21labs_api_key_plaintext`.
   * You must provide an API key using one of the following fields:
   * `ai21labs_api_key` or `ai21labs_api_key_plaintext`.
   */
  ai21labs_api_key?: VariableOr<string>;
  /**
   * An AI21 Labs API key provided as a plaintext string. If you prefer to
   * reference your key using Databricks Secrets, see `ai21labs_api_key`. You
   * must provide an API key using one of the following fields:
   * `ai21labs_api_key` or `ai21labs_api_key_plaintext`.
   */
  ai21labs_api_key_plaintext?: VariableOr<string>;
}

export interface AiGatewayConfig {
  /**
   * Configuration for traffic fallback which auto fallbacks to other served entities if the request to a served
   * entity fails with certain error codes, to increase availability.
   */
  fallback_config?: VariableOr<FallbackConfig>;
  /**
   * Configuration for AI Guardrails to prevent unwanted data and unsafe data in requests and responses.
   */
  guardrails?: VariableOr<AiGatewayGuardrails>;
  /**
   * Configuration for payload logging using inference tables.
   * Use these tables to monitor and audit data being sent to and received from model APIs and to improve model quality.
   */
  inference_table_config?: VariableOr<AiGatewayInferenceTableConfig>;
  /**
   * Configuration for rate limits which can be set to limit endpoint traffic.
   */
  rate_limits?: VariableOr<AiGatewayRateLimit[]>;
  /**
   * Configuration to enable usage tracking using system tables.
   * These tables allow you to monitor operational usage on endpoints and their associated costs.
   */
  usage_tracking_config?: VariableOr<AiGatewayUsageTrackingConfig>;
}

export interface AiGatewayGuardrailParameters {
  /**
   * List of invalid keywords.
   * AI guardrail uses keyword or string matching to decide if the keyword exists in the request or response content.
   * @deprecated
   */
  invalid_keywords?: VariableOr<string[]>;
  /**
   * Configuration for guardrail PII filter.
   */
  pii?: VariableOr<AiGatewayGuardrailPiiBehavior>;
  /**
   * Indicates whether the safety filter is enabled.
   */
  safety?: VariableOr<boolean>;
  /**
   * The list of allowed topics.
   * Given a chat request, this guardrail flags the request if its topic is not in the allowed topics.
   * @deprecated
   */
  valid_topics?: VariableOr<string[]>;
}

export interface AiGatewayGuardrailPiiBehavior {
  /**
   * Configuration for input guardrail filters.
   */
  behavior?: VariableOr<AiGatewayGuardrailPiiBehaviorBehavior>;
}

export type AiGatewayGuardrailPiiBehaviorBehavior =
  | "NONE"
  | "BLOCK"
  | "MASK";

export interface AiGatewayGuardrails {
  /**
   * Configuration for input guardrail filters.
   */
  input?: VariableOr<AiGatewayGuardrailParameters>;
  /**
   * Configuration for output guardrail filters.
   */
  output?: VariableOr<AiGatewayGuardrailParameters>;
}

export interface AiGatewayInferenceTableConfig {
  /**
   * The name of the catalog in Unity Catalog. Required when enabling inference tables.
   * NOTE: On update, you have to disable inference table first in order to change the catalog name.
   */
  catalog_name?: VariableOr<string>;
  /**
   * Indicates whether the inference table is enabled.
   */
  enabled?: VariableOr<boolean>;
  /**
   * The name of the schema in Unity Catalog. Required when enabling inference tables.
   * NOTE: On update, you have to disable inference table first in order to change the schema name.
   */
  schema_name?: VariableOr<string>;
  /**
   * The prefix of the table in Unity Catalog.
   * NOTE: On update, you have to disable inference table first in order to change the prefix name.
   */
  table_name_prefix?: VariableOr<string>;
}

export interface AiGatewayRateLimit {
  /**
   * Used to specify how many calls are allowed for a key within the renewal_period.
   */
  calls?: VariableOr<number>;
  /**
   * Key field for a rate limit. Currently, 'user', 'user_group, 'service_principal', and 'endpoint' are supported,
   * with 'endpoint' being the default if not specified.
   */
  key?: VariableOr<AiGatewayRateLimitKey>;
  /**
   * Principal field for a user, user group, or service principal to apply rate limiting to. Accepts a user email, group name, or service principal application ID.
   */
  principal?: VariableOr<string>;
  /**
   * Renewal period field for a rate limit. Currently, only 'minute' is supported.
   */
  renewal_period: VariableOr<AiGatewayRateLimitRenewalPeriod>;
  /**
   * Used to specify how many tokens are allowed for a key within the renewal_period.
   */
  tokens?: VariableOr<number>;
}

export type AiGatewayRateLimitKey =
  | "user"
  | "endpoint"
  | "user_group"
  | "service_principal";

export type AiGatewayRateLimitRenewalPeriod =
  | "minute";

export interface AiGatewayUsageTrackingConfig {
  /**
   * Whether to enable usage tracking.
   */
  enabled?: VariableOr<boolean>;
}

export interface AmazonBedrockConfig {
  /**
   * The Databricks secret key reference for an AWS access key ID with
   * permissions to interact with Bedrock services. If you prefer to paste
   * your API key directly, see `aws_access_key_id_plaintext`. You must provide an API
   * key using one of the following fields: `aws_access_key_id` or
   * `aws_access_key_id_plaintext`.
   */
  aws_access_key_id?: VariableOr<string>;
  /**
   * An AWS access key ID with permissions to interact with Bedrock services
   * provided as a plaintext string. If you prefer to reference your key using
   * Databricks Secrets, see `aws_access_key_id`. You must provide an API key
   * using one of the following fields: `aws_access_key_id` or
   * `aws_access_key_id_plaintext`.
   */
  aws_access_key_id_plaintext?: VariableOr<string>;
  /**
   * The AWS region to use. Bedrock has to be enabled there.
   */
  aws_region: VariableOr<string>;
  /**
   * The Databricks secret key reference for an AWS secret access key paired
   * with the access key ID, with permissions to interact with Bedrock
   * services. If you prefer to paste your API key directly, see
   * `aws_secret_access_key_plaintext`. You must provide an API key using one
   * of the following fields: `aws_secret_access_key` or
   * `aws_secret_access_key_plaintext`.
   */
  aws_secret_access_key?: VariableOr<string>;
  /**
   * An AWS secret access key paired with the access key ID, with permissions
   * to interact with Bedrock services provided as a plaintext string. If you
   * prefer to reference your key using Databricks Secrets, see
   * `aws_secret_access_key`. You must provide an API key using one of the
   * following fields: `aws_secret_access_key` or
   * `aws_secret_access_key_plaintext`.
   */
  aws_secret_access_key_plaintext?: VariableOr<string>;
  /**
   * The underlying provider in Amazon Bedrock. Supported values (case
   * insensitive) include: Anthropic, Cohere, AI21Labs, Amazon.
   */
  bedrock_provider: VariableOr<AmazonBedrockConfigBedrockProvider>;
  /**
   * ARN of the instance profile that the external model will use to access AWS resources.
   * You must authenticate using an instance profile or access keys.
   * If you prefer to authenticate using access keys, see `aws_access_key_id`,
   * `aws_access_key_id_plaintext`, `aws_secret_access_key` and `aws_secret_access_key_plaintext`.
   */
  instance_profile_arn?: VariableOr<string>;
}

export type AmazonBedrockConfigBedrockProvider =
  | "anthropic"
  | "cohere"
  | "ai21labs"
  | "amazon";

export interface AnthropicConfig {
  /**
   * The Databricks secret key reference for an Anthropic API key. If you
   * prefer to paste your API key directly, see `anthropic_api_key_plaintext`.
   * You must provide an API key using one of the following fields:
   * `anthropic_api_key` or `anthropic_api_key_plaintext`.
   */
  anthropic_api_key?: VariableOr<string>;
  /**
   * The Anthropic API key provided as a plaintext string. If you prefer to
   * reference your key using Databricks Secrets, see `anthropic_api_key`. You
   * must provide an API key using one of the following fields:
   * `anthropic_api_key` or `anthropic_api_key_plaintext`.
   */
  anthropic_api_key_plaintext?: VariableOr<string>;
}

export interface ApiKeyAuth {
  /**
   * The name of the API key parameter used for authentication.
   */
  key: VariableOr<string>;
  /**
   * The Databricks secret key reference for an API Key.
   * If you prefer to paste your token directly, see `value_plaintext`.
   */
  value?: VariableOr<string>;
  /**
   * The API Key provided as a plaintext string. If you prefer to reference your
   * token using Databricks Secrets, see `value`.
   */
  value_plaintext?: VariableOr<string>;
}

export interface AutoCaptureConfigInput {
  /**
   * The name of the catalog in Unity Catalog. NOTE: On update, you cannot change the catalog name if the inference table is already enabled.
   */
  catalog_name?: VariableOr<string>;
  /**
   * Indicates whether the inference table is enabled.
   */
  enabled?: VariableOr<boolean>;
  /**
   * The name of the schema in Unity Catalog. NOTE: On update, you cannot change the schema name if the inference table is already enabled.
   */
  schema_name?: VariableOr<string>;
  /**
   * The prefix of the table in Unity Catalog. NOTE: On update, you cannot change the prefix name if the inference table is already enabled.
   */
  table_name_prefix?: VariableOr<string>;
}

export interface BearerTokenAuth {
  /**
   * The Databricks secret key reference for a token.
   * If you prefer to paste your token directly, see `token_plaintext`.
   */
  token?: VariableOr<string>;
  /**
   * The token provided as a plaintext string. If you prefer to reference your
   * token using Databricks Secrets, see `token`.
   */
  token_plaintext?: VariableOr<string>;
}

export interface CohereConfig {
  /**
   * This is an optional field to provide a customized base URL for the Cohere
   * API. If left unspecified, the standard Cohere base URL is used.
   */
  cohere_api_base?: VariableOr<string>;
  /**
   * The Databricks secret key reference for a Cohere API key. If you prefer
   * to paste your API key directly, see `cohere_api_key_plaintext`. You must
   * provide an API key using one of the following fields: `cohere_api_key` or
   * `cohere_api_key_plaintext`.
   */
  cohere_api_key?: VariableOr<string>;
  /**
   * The Cohere API key provided as a plaintext string. If you prefer to
   * reference your key using Databricks Secrets, see `cohere_api_key`. You
   * must provide an API key using one of the following fields:
   * `cohere_api_key` or `cohere_api_key_plaintext`.
   */
  cohere_api_key_plaintext?: VariableOr<string>;
}

/**
 * Configs needed to create a custom provider model route.
 */
export interface CustomProviderConfig {
  /**
   * This is a field to provide API key authentication for the custom provider API.
   * You can only specify one authentication method.
   */
  api_key_auth?: VariableOr<ApiKeyAuth>;
  /**
   * This is a field to provide bearer token authentication for the custom provider API.
   * You can only specify one authentication method.
   */
  bearer_token_auth?: VariableOr<BearerTokenAuth>;
  /**
   * This is a field to provide the URL of the custom provider API.
   */
  custom_provider_url: VariableOr<string>;
}

export interface DatabricksModelServingConfig {
  /**
   * The Databricks secret key reference for a Databricks API token that
   * corresponds to a user or service principal with Can Query access to the
   * model serving endpoint pointed to by this external model. If you prefer
   * to paste your API key directly, see `databricks_api_token_plaintext`. You
   * must provide an API key using one of the following fields:
   * `databricks_api_token` or `databricks_api_token_plaintext`.
   */
  databricks_api_token?: VariableOr<string>;
  /**
   * The Databricks API token that corresponds to a user or service principal
   * with Can Query access to the model serving endpoint pointed to by this
   * external model provided as a plaintext string. If you prefer to reference
   * your key using Databricks Secrets, see `databricks_api_token`. You must
   * provide an API key using one of the following fields:
   * `databricks_api_token` or `databricks_api_token_plaintext`.
   */
  databricks_api_token_plaintext?: VariableOr<string>;
  /**
   * The URL of the Databricks workspace containing the model serving endpoint
   * pointed to by this external model.
   */
  databricks_workspace_url: VariableOr<string>;
}

export interface EmailNotifications {
  /**
   * A list of email addresses to be notified when an endpoint fails to update its configuration or state.
   */
  on_update_failure?: VariableOr<string[]>;
  /**
   * A list of email addresses to be notified when an endpoint successfully updates its configuration or state.
   */
  on_update_success?: VariableOr<string[]>;
}

export interface EndpointCoreConfigInput {
  /**
   * Configuration for Inference Tables which automatically logs requests and responses to Unity Catalog.
   * Note: this field is deprecated for creating new provisioned throughput endpoints,
   * or updating existing provisioned throughput endpoints that never have inference table configured;
   * in these cases please use AI Gateway to manage inference tables.
   */
  auto_capture_config?: VariableOr<AutoCaptureConfigInput>;
  /**
   * The list of served entities under the serving endpoint config.
   */
  served_entities?: VariableOr<ServedEntityInput[]>;
  /**
   * (Deprecated, use served_entities instead) The list of served models under the serving endpoint config.
   */
  served_models?: VariableOr<ServedModelInput[]>;
  /**
   * The traffic configuration associated with the serving endpoint config.
   */
  traffic_config?: VariableOr<TrafficConfig>;
}

export interface EndpointTag {
  /**
   * Key field for a serving endpoint tag.
   */
  key: VariableOr<string>;
  /**
   * Optional value field for a serving endpoint tag.
   */
  value?: VariableOr<string>;
}

export interface ExternalModel {
  /**
   * AI21Labs Config. Only required if the provider is 'ai21labs'.
   */
  ai21labs_config?: VariableOr<Ai21LabsConfig>;
  /**
   * Amazon Bedrock Config. Only required if the provider is 'amazon-bedrock'.
   */
  amazon_bedrock_config?: VariableOr<AmazonBedrockConfig>;
  /**
   * Anthropic Config. Only required if the provider is 'anthropic'.
   */
  anthropic_config?: VariableOr<AnthropicConfig>;
  /**
   * Cohere Config. Only required if the provider is 'cohere'.
   */
  cohere_config?: VariableOr<CohereConfig>;
  /**
   * Custom Provider Config. Only required if the provider is 'custom'.
   */
  custom_provider_config?: VariableOr<CustomProviderConfig>;
  /**
   * Databricks Model Serving Config. Only required if the provider is 'databricks-model-serving'.
   */
  databricks_model_serving_config?: VariableOr<DatabricksModelServingConfig>;
  /**
   * Google Cloud Vertex AI Config. Only required if the provider is 'google-cloud-vertex-ai'.
   */
  google_cloud_vertex_ai_config?: VariableOr<GoogleCloudVertexAiConfig>;
  /**
   * The name of the external model.
   */
  name: VariableOr<string>;
  /**
   * OpenAI Config. Only required if the provider is 'openai'.
   */
  openai_config?: VariableOr<OpenAiConfig>;
  /**
   * PaLM Config. Only required if the provider is 'palm'.
   */
  palm_config?: VariableOr<PaLmConfig>;
  /**
   * The name of the provider for the external model. Currently, the supported providers are 'ai21labs', 'anthropic', 'amazon-bedrock', 'cohere', 'databricks-model-serving', 'google-cloud-vertex-ai', 'openai', 'palm', and 'custom'.
   */
  provider: VariableOr<ExternalModelProvider>;
  /**
   * The task type of the external model.
   */
  task: VariableOr<string>;
}

export type ExternalModelProvider =
  | "ai21labs"
  | "anthropic"
  | "amazon-bedrock"
  | "cohere"
  | "databricks-model-serving"
  | "google-cloud-vertex-ai"
  | "openai"
  | "palm"
  | "custom";

export interface FallbackConfig {
  /**
   * Whether to enable traffic fallback. When a served entity in the serving endpoint returns specific error
   * codes (e.g. 500), the request will automatically be round-robin attempted with other served entities in the same
   * endpoint, following the order of served entity list, until a successful response is returned.
   * If all attempts fail, return the last response with the error code.
   */
  enabled: VariableOr<boolean>;
}

export interface GoogleCloudVertexAiConfig {
  /**
   * The Databricks secret key reference for a private key for the service
   * account which has access to the Google Cloud Vertex AI Service. See [Best
   * practices for managing service account keys]. If you prefer to paste your
   * API key directly, see `private_key_plaintext`. You must provide an API
   * key using one of the following fields: `private_key` or
   * `private_key_plaintext`
   * 
   * [Best practices for managing service account keys]: https://cloud.google.com/iam/docs/best-practices-for-managing-service-account-keys
   */
  private_key?: VariableOr<string>;
  /**
   * The private key for the service account which has access to the Google
   * Cloud Vertex AI Service provided as a plaintext secret. See [Best
   * practices for managing service account keys]. If you prefer to reference
   * your key using Databricks Secrets, see `private_key`. You must provide an
   * API key using one of the following fields: `private_key` or
   * `private_key_plaintext`.
   * 
   * [Best practices for managing service account keys]: https://cloud.google.com/iam/docs/best-practices-for-managing-service-account-keys
   */
  private_key_plaintext?: VariableOr<string>;
  /**
   * This is the Google Cloud project id that the service account is
   * associated with.
   */
  project_id: VariableOr<string>;
  /**
   * This is the region for the Google Cloud Vertex AI Service. See [supported
   * regions] for more details. Some models are only available in specific
   * regions.
   * 
   * [supported regions]: https://cloud.google.com/vertex-ai/docs/general/locations
   */
  region: VariableOr<string>;
}

/**
 * Configs needed to create an OpenAI model route.
 */
export interface OpenAiConfig {
  /**
   * This field is only required for Azure AD OpenAI and is the Microsoft
   * Entra Client ID.
   */
  microsoft_entra_client_id?: VariableOr<string>;
  /**
   * The Databricks secret key reference for a client secret used for
   * Microsoft Entra ID authentication. If you prefer to paste your client
   * secret directly, see `microsoft_entra_client_secret_plaintext`. You must
   * provide an API key using one of the following fields:
   * `microsoft_entra_client_secret` or
   * `microsoft_entra_client_secret_plaintext`.
   */
  microsoft_entra_client_secret?: VariableOr<string>;
  /**
   * The client secret used for Microsoft Entra ID authentication provided as
   * a plaintext string. If you prefer to reference your key using Databricks
   * Secrets, see `microsoft_entra_client_secret`. You must provide an API key
   * using one of the following fields: `microsoft_entra_client_secret` or
   * `microsoft_entra_client_secret_plaintext`.
   */
  microsoft_entra_client_secret_plaintext?: VariableOr<string>;
  /**
   * This field is only required for Azure AD OpenAI and is the Microsoft
   * Entra Tenant ID.
   */
  microsoft_entra_tenant_id?: VariableOr<string>;
  /**
   * This is a field to provide a customized base URl for the OpenAI API. For
   * Azure OpenAI, this field is required, and is the base URL for the Azure
   * OpenAI API service provided by Azure. For other OpenAI API types, this
   * field is optional, and if left unspecified, the standard OpenAI base URL
   * is used.
   */
  openai_api_base?: VariableOr<string>;
  /**
   * The Databricks secret key reference for an OpenAI API key using the
   * OpenAI or Azure service. If you prefer to paste your API key directly,
   * see `openai_api_key_plaintext`. You must provide an API key using one of
   * the following fields: `openai_api_key` or `openai_api_key_plaintext`.
   */
  openai_api_key?: VariableOr<string>;
  /**
   * The OpenAI API key using the OpenAI or Azure service provided as a
   * plaintext string. If you prefer to reference your key using Databricks
   * Secrets, see `openai_api_key`. You must provide an API key using one of
   * the following fields: `openai_api_key` or `openai_api_key_plaintext`.
   */
  openai_api_key_plaintext?: VariableOr<string>;
  /**
   * This is an optional field to specify the type of OpenAI API to use. For
   * Azure OpenAI, this field is required, and adjust this parameter to
   * represent the preferred security access validation protocol. For access
   * token validation, use azure. For authentication using Azure Active
   * Directory (Azure AD) use, azuread.
   */
  openai_api_type?: VariableOr<string>;
  /**
   * This is an optional field to specify the OpenAI API version. For Azure
   * OpenAI, this field is required, and is the version of the Azure OpenAI
   * service to utilize, specified by a date.
   */
  openai_api_version?: VariableOr<string>;
  /**
   * This field is only required for Azure OpenAI and is the name of the
   * deployment resource for the Azure OpenAI service.
   */
  openai_deployment_name?: VariableOr<string>;
  /**
   * This is an optional field to specify the organization in OpenAI or Azure
   * OpenAI.
   */
  openai_organization?: VariableOr<string>;
}

export interface PaLmConfig {
  /**
   * The Databricks secret key reference for a PaLM API key. If you prefer to
   * paste your API key directly, see `palm_api_key_plaintext`. You must
   * provide an API key using one of the following fields: `palm_api_key` or
   * `palm_api_key_plaintext`.
   */
  palm_api_key?: VariableOr<string>;
  /**
   * The PaLM API key provided as a plaintext string. If you prefer to
   * reference your key using Databricks Secrets, see `palm_api_key`. You must
   * provide an API key using one of the following fields: `palm_api_key` or
   * `palm_api_key_plaintext`.
   */
  palm_api_key_plaintext?: VariableOr<string>;
}

export interface RateLimit {
  /**
   * Used to specify how many calls are allowed for a key within the renewal_period.
   */
  calls: VariableOr<number>;
  /**
   * Key field for a serving endpoint rate limit. Currently, only 'user' and 'endpoint' are supported, with 'endpoint' being the default if not specified.
   */
  key?: VariableOr<RateLimitKey>;
  /**
   * Renewal period field for a serving endpoint rate limit. Currently, only 'minute' is supported.
   */
  renewal_period: VariableOr<RateLimitRenewalPeriod>;
}

export type RateLimitKey =
  | "user"
  | "endpoint";

export type RateLimitRenewalPeriod =
  | "minute";

export interface Route {
  served_entity_name?: VariableOr<string>;
  /**
   * The name of the served model this route configures traffic for.
   */
  served_model_name?: VariableOr<string>;
  /**
   * The percentage of endpoint traffic to send to this route. It must be an integer between 0 and 100 inclusive.
   */
  traffic_percentage: VariableOr<number>;
}

export interface ServedEntityInput {
  /**
   * The name of the entity to be served. The entity may be a model in the Databricks Model Registry, a model in the Unity Catalog (UC), or a function of type FEATURE_SPEC in the UC. If it is a UC object, the full name of the object should be given in the form of **catalog_name.schema_name.model_name**.
   */
  entity_name?: VariableOr<string>;
  entity_version?: VariableOr<string>;
  /**
   * An object containing a set of optional, user-specified environment variable key-value pairs used for serving this entity. Note: this is an experimental feature and subject to change. Example entity environment variables that refer to Databricks secrets: `{"OPENAI_API_KEY": "{{secrets/my_scope/my_key}}", "DATABRICKS_TOKEN": "{{secrets/my_scope2/my_key2}}"}`
   */
  environment_vars?: VariableOr<Record<string, string>>;
  /**
   * The external model to be served. NOTE: Only one of external_model and (entity_name, entity_version, workload_size, workload_type, and scale_to_zero_enabled) can be specified with the latter set being used for custom model serving for a Databricks registered model. For an existing endpoint with external_model, it cannot be updated to an endpoint without external_model. If the endpoint is created without external_model, users cannot update it to add external_model later. The task type of all external models within an endpoint must be the same.
   */
  external_model?: VariableOr<ExternalModel>;
  /**
   * ARN of the instance profile that the served entity uses to access AWS resources.
   */
  instance_profile_arn?: VariableOr<string>;
  /**
   * The maximum provisioned concurrency that the endpoint can scale up to. Do not use if workload_size is specified.
   */
  max_provisioned_concurrency?: VariableOr<number>;
  /**
   * The maximum tokens per second that the endpoint can scale up to.
   */
  max_provisioned_throughput?: VariableOr<number>;
  /**
   * The minimum provisioned concurrency that the endpoint can scale down to. Do not use if workload_size is specified.
   */
  min_provisioned_concurrency?: VariableOr<number>;
  /**
   * The minimum tokens per second that the endpoint can scale down to.
   */
  min_provisioned_throughput?: VariableOr<number>;
  /**
   * The name of a served entity. It must be unique across an endpoint. A served entity name can consist of alphanumeric characters, dashes, and underscores. If not specified for an external model, this field defaults to external_model.name, with '.' and ':' replaced with '-', and if not specified for other entities, it defaults to entity_name-entity_version.
   */
  name?: VariableOr<string>;
  /**
   * The number of model units provisioned.
   */
  provisioned_model_units?: VariableOr<number>;
  /**
   * Whether the compute resources for the served entity should scale down to zero.
   */
  scale_to_zero_enabled?: VariableOr<boolean>;
  /**
   * The workload size of the served entity. The workload size corresponds to a range of provisioned concurrency that the compute autoscales between. A single unit of provisioned concurrency can process one request at a time. Valid workload sizes are "Small" (4 - 4 provisioned concurrency), "Medium" (8 - 16 provisioned concurrency), and "Large" (16 - 64 provisioned concurrency). Additional custom workload sizes can also be used when available in the workspace. If scale-to-zero is enabled, the lower bound of the provisioned concurrency for each workload size is 0. Do not use if min_provisioned_concurrency and max_provisioned_concurrency are specified.
   */
  workload_size?: VariableOr<string>;
  /**
   * The workload type of the served entity. The workload type selects which type of compute to use in the endpoint. The default value for this parameter is "CPU". For deep learning workloads, GPU acceleration is available by selecting workload types like GPU_SMALL and others. See the available [GPU types](https://docs.databricks.com/en/machine-learning/model-serving/create-manage-serving-endpoints.html#gpu-workload-types).
   */
  workload_type?: VariableOr<ServingModelWorkloadType>;
}

export interface ServedModelInput {
  /**
   * An object containing a set of optional, user-specified environment variable key-value pairs used for serving this entity. Note: this is an experimental feature and subject to change. Example entity environment variables that refer to Databricks secrets: `{"OPENAI_API_KEY": "{{secrets/my_scope/my_key}}", "DATABRICKS_TOKEN": "{{secrets/my_scope2/my_key2}}"}`
   */
  environment_vars?: VariableOr<Record<string, string>>;
  /**
   * ARN of the instance profile that the served entity uses to access AWS resources.
   */
  instance_profile_arn?: VariableOr<string>;
  /**
   * The maximum provisioned concurrency that the endpoint can scale up to. Do not use if workload_size is specified.
   */
  max_provisioned_concurrency?: VariableOr<number>;
  /**
   * The maximum tokens per second that the endpoint can scale up to.
   */
  max_provisioned_throughput?: VariableOr<number>;
  /**
   * The minimum provisioned concurrency that the endpoint can scale down to. Do not use if workload_size is specified.
   */
  min_provisioned_concurrency?: VariableOr<number>;
  /**
   * The minimum tokens per second that the endpoint can scale down to.
   */
  min_provisioned_throughput?: VariableOr<number>;
  model_name: VariableOr<string>;
  model_version: VariableOr<string>;
  /**
   * The name of a served entity. It must be unique across an endpoint. A served entity name can consist of alphanumeric characters, dashes, and underscores. If not specified for an external model, this field defaults to external_model.name, with '.' and ':' replaced with '-', and if not specified for other entities, it defaults to entity_name-entity_version.
   */
  name?: VariableOr<string>;
  /**
   * The number of model units provisioned.
   */
  provisioned_model_units?: VariableOr<number>;
  /**
   * Whether the compute resources for the served entity should scale down to zero.
   */
  scale_to_zero_enabled: VariableOr<boolean>;
  /**
   * The workload size of the served entity. The workload size corresponds to a range of provisioned concurrency that the compute autoscales between. A single unit of provisioned concurrency can process one request at a time. Valid workload sizes are "Small" (4 - 4 provisioned concurrency), "Medium" (8 - 16 provisioned concurrency), and "Large" (16 - 64 provisioned concurrency). Additional custom workload sizes can also be used when available in the workspace. If scale-to-zero is enabled, the lower bound of the provisioned concurrency for each workload size is 0. Do not use if min_provisioned_concurrency and max_provisioned_concurrency are specified.
   */
  workload_size?: VariableOr<string>;
  /**
   * The workload type of the served entity. The workload type selects which type of compute to use in the endpoint. The default value for this parameter is "CPU". For deep learning workloads, GPU acceleration is available by selecting workload types like GPU_SMALL and others. See the available [GPU types](https://docs.databricks.com/en/machine-learning/model-serving/create-manage-serving-endpoints.html#gpu-workload-types).
   */
  workload_type?: VariableOr<ServedModelInputWorkloadType>;
}

/**
 * Please keep this in sync with with workload types in InferenceEndpointEntities.scala
 */
export type ServedModelInputWorkloadType =
  | "CPU"
  | "GPU_MEDIUM"
  | "GPU_SMALL"
  | "GPU_LARGE"
  | "MULTIGPU_MEDIUM";

/**
 * Please keep this in sync with with workload types in InferenceEndpointEntities.scala
 */
export type ServingModelWorkloadType =
  | "CPU"
  | "GPU_MEDIUM"
  | "GPU_SMALL"
  | "GPU_LARGE"
  | "MULTIGPU_MEDIUM";

export interface TrafficConfig {
  /**
   * The list of routes that define traffic to each served entity.
   */
  routes?: VariableOr<Route[]>;
}
