// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectAwsAccessKeySecretAccessKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectAwsAccessKey struct {
	AccessKeyId     string                                                                                     `json:"access_key_id,omitempty"`
	SecretAccessKey *ResourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectAwsAccessKeySecretAccessKey `json:"secret_access_key,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectServiceCredential struct {
	Name string `json:"name"`
}

type ResourceAiGatewayModelProviderServiceConfigAmazonBedrockDirect struct {
	AwsAccessKey      *ResourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectAwsAccessKey      `json:"aws_access_key,omitempty"`
	Region            string                                                                           `json:"region,omitempty"`
	ServiceCredential *ResourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectServiceCredential `json:"service_credential,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAmazonBedrock struct {
	Direct *ResourceAiGatewayModelProviderServiceConfigAmazonBedrockDirect `json:"direct,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAnthropicDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAnthropicDirect struct {
	ApiKey *ResourceAiGatewayModelProviderServiceConfigAnthropicDirectApiKey `json:"api_key,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAnthropicRelayed struct {
	PlanType string `json:"plan_type,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAnthropic struct {
	Direct  *ResourceAiGatewayModelProviderServiceConfigAnthropicDirect  `json:"direct,omitempty"`
	Relayed *ResourceAiGatewayModelProviderServiceConfigAnthropicRelayed `json:"relayed,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectEntraServicePrincipalClientSecret struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectEntraServicePrincipal struct {
	ClientId     string                                                                                         `json:"client_id,omitempty"`
	ClientSecret *ResourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectEntraServicePrincipalClientSecret `json:"client_secret,omitempty"`
	TenantId     string                                                                                         `json:"tenant_id,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectServiceCredential struct {
	Name string `json:"name"`
}

type ResourceAiGatewayModelProviderServiceConfigAzureOpenaiDirect struct {
	ApiKey                *ResourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectApiKey                `json:"api_key,omitempty"`
	BaseUrl               string                                                                             `json:"base_url,omitempty"`
	EntraServicePrincipal *ResourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectEntraServicePrincipal `json:"entra_service_principal,omitempty"`
	ServiceCredential     *ResourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectServiceCredential     `json:"service_credential,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigAzureOpenai struct {
	Direct *ResourceAiGatewayModelProviderServiceConfigAzureOpenaiDirect `json:"direct,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigCustomDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigCustomDirect struct {
	ApiKey  *ResourceAiGatewayModelProviderServiceConfigCustomDirectApiKey `json:"api_key,omitempty"`
	BaseUrl string                                                         `json:"base_url,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigCustom struct {
	Direct *ResourceAiGatewayModelProviderServiceConfigCustomDirect `json:"direct,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigGeminiEnterpriseDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigGeminiEnterpriseDirect struct {
	ApiKey    *ResourceAiGatewayModelProviderServiceConfigGeminiEnterpriseDirectApiKey `json:"api_key,omitempty"`
	ProjectId string                                                                   `json:"project_id,omitempty"`
	Region    string                                                                   `json:"region,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigGeminiEnterprise struct {
	Direct *ResourceAiGatewayModelProviderServiceConfigGeminiEnterpriseDirect `json:"direct,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigInferenceTable struct {
	Disabled        bool   `json:"disabled,omitempty"`
	IsDeleted       bool   `json:"is_deleted,omitempty"`
	Parent          string `json:"parent"`
	Table           string `json:"table,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectEntraServicePrincipalClientSecret struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectEntraServicePrincipal struct {
	ClientId     string                                                                                              `json:"client_id,omitempty"`
	ClientSecret *ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectEntraServicePrincipalClientSecret `json:"client_secret,omitempty"`
	TenantId     string                                                                                              `json:"tenant_id,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectServiceCredential struct {
	Name string `json:"name"`
}

type ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirect struct {
	ApiKey                *ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectApiKey                `json:"api_key,omitempty"`
	BaseUrl               string                                                                                  `json:"base_url,omitempty"`
	EntraServicePrincipal *ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectEntraServicePrincipal `json:"entra_service_principal,omitempty"`
	ServiceCredential     *ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectServiceCredential     `json:"service_credential,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundry struct {
	Direct *ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirect `json:"direct,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigOpenaiDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigOpenaiDirect struct {
	ApiKey       *ResourceAiGatewayModelProviderServiceConfigOpenaiDirectApiKey `json:"api_key,omitempty"`
	BaseUrl      string                                                         `json:"base_url,omitempty"`
	Organization string                                                         `json:"organization,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigOpenai struct {
	Direct *ResourceAiGatewayModelProviderServiceConfigOpenaiDirect `json:"direct,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigRateLimits struct {
	Key             string `json:"key"`
	Principal       string `json:"principal,omitempty"`
	RenewalPeriod   string `json:"renewal_period"`
	RequestTagKey   string `json:"request_tag_key,omitempty"`
	RequestTagValue string `json:"request_tag_value,omitempty"`
	Requests        int    `json:"requests,omitempty"`
	Tokens          int    `json:"tokens,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfigTargets struct {
	Model          string   `json:"model"`
	NativeApiTypes []string `json:"native_api_types,omitempty"`
}

type ResourceAiGatewayModelProviderServiceConfig struct {
	AllowAllTargets        bool                                                         `json:"allow_all_targets,omitempty"`
	AmazonBedrock          *ResourceAiGatewayModelProviderServiceConfigAmazonBedrock    `json:"amazon_bedrock,omitempty"`
	Anthropic              *ResourceAiGatewayModelProviderServiceConfigAnthropic        `json:"anthropic,omitempty"`
	AzureOpenai            *ResourceAiGatewayModelProviderServiceConfigAzureOpenai      `json:"azure_openai,omitempty"`
	Custom                 *ResourceAiGatewayModelProviderServiceConfigCustom           `json:"custom,omitempty"`
	ForwardHeaders         bool                                                         `json:"forward_headers,omitempty"`
	ForwardQueryParameters bool                                                         `json:"forward_query_parameters,omitempty"`
	ForwardUnmanagedPaths  bool                                                         `json:"forward_unmanaged_paths,omitempty"`
	GeminiEnterprise       *ResourceAiGatewayModelProviderServiceConfigGeminiEnterprise `json:"gemini_enterprise,omitempty"`
	InferenceTable         *ResourceAiGatewayModelProviderServiceConfigInferenceTable   `json:"inference_table,omitempty"`
	MicrosoftFoundry       *ResourceAiGatewayModelProviderServiceConfigMicrosoftFoundry `json:"microsoft_foundry,omitempty"`
	Openai                 *ResourceAiGatewayModelProviderServiceConfigOpenai           `json:"openai,omitempty"`
	ProviderType           string                                                       `json:"provider_type,omitempty"`
	RateLimits             []ResourceAiGatewayModelProviderServiceConfigRateLimits      `json:"rate_limits,omitempty"`
	Targets                []ResourceAiGatewayModelProviderServiceConfigTargets         `json:"targets,omitempty"`
}

type ResourceAiGatewayModelProviderServiceProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceAiGatewayModelProviderService struct {
	Comment                string                                               `json:"comment,omitempty"`
	Config                 *ResourceAiGatewayModelProviderServiceConfig         `json:"config,omitempty"`
	CreateTime             string                                               `json:"create_time,omitempty"`
	CreatedBy              string                                               `json:"created_by,omitempty"`
	EffectiveOwner         string                                               `json:"effective_owner,omitempty"`
	Etag                   string                                               `json:"etag,omitempty"`
	MetastoreId            string                                               `json:"metastore_id,omitempty"`
	ModelProviderServiceId string                                               `json:"model_provider_service_id"`
	Name                   string                                               `json:"name,omitempty"`
	Owner                  string                                               `json:"owner,omitempty"`
	Parent                 string                                               `json:"parent"`
	ProviderConfig         *ResourceAiGatewayModelProviderServiceProviderConfig `json:"provider_config,omitempty"`
	UpdateTime             string                                               `json:"update_time,omitempty"`
	UpdatedBy              string                                               `json:"updated_by,omitempty"`
}
