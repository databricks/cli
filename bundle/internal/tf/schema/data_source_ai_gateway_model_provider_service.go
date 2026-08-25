// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectAwsAccessKeySecretAccessKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectAwsAccessKey struct {
	AccessKeyId     string                                                                                       `json:"access_key_id,omitempty"`
	SecretAccessKey *DataSourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectAwsAccessKeySecretAccessKey `json:"secret_access_key,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectServiceCredential struct {
	Name string `json:"name"`
}

type DataSourceAiGatewayModelProviderServiceConfigAmazonBedrockDirect struct {
	AwsAccessKey      *DataSourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectAwsAccessKey      `json:"aws_access_key,omitempty"`
	Region            string                                                                             `json:"region,omitempty"`
	ServiceCredential *DataSourceAiGatewayModelProviderServiceConfigAmazonBedrockDirectServiceCredential `json:"service_credential,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAmazonBedrock struct {
	Direct *DataSourceAiGatewayModelProviderServiceConfigAmazonBedrockDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAnthropicDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAnthropicDirect struct {
	ApiKey *DataSourceAiGatewayModelProviderServiceConfigAnthropicDirectApiKey `json:"api_key,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAnthropicRelayed struct {
	PlanType string `json:"plan_type,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAnthropic struct {
	Direct  *DataSourceAiGatewayModelProviderServiceConfigAnthropicDirect  `json:"direct,omitempty"`
	Relayed *DataSourceAiGatewayModelProviderServiceConfigAnthropicRelayed `json:"relayed,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectEntraServicePrincipalClientSecret struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectEntraServicePrincipal struct {
	ClientId     string                                                                                           `json:"client_id,omitempty"`
	ClientSecret *DataSourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectEntraServicePrincipalClientSecret `json:"client_secret,omitempty"`
	TenantId     string                                                                                           `json:"tenant_id,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectServiceCredential struct {
	Name string `json:"name"`
}

type DataSourceAiGatewayModelProviderServiceConfigAzureOpenaiDirect struct {
	ApiKey                *DataSourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectApiKey                `json:"api_key,omitempty"`
	BaseUrl               string                                                                               `json:"base_url,omitempty"`
	EntraServicePrincipal *DataSourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectEntraServicePrincipal `json:"entra_service_principal,omitempty"`
	ServiceCredential     *DataSourceAiGatewayModelProviderServiceConfigAzureOpenaiDirectServiceCredential     `json:"service_credential,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigAzureOpenai struct {
	Direct *DataSourceAiGatewayModelProviderServiceConfigAzureOpenaiDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigCustomDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigCustomDirect struct {
	ApiKey  *DataSourceAiGatewayModelProviderServiceConfigCustomDirectApiKey `json:"api_key,omitempty"`
	BaseUrl string                                                           `json:"base_url,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigCustom struct {
	Direct *DataSourceAiGatewayModelProviderServiceConfigCustomDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigGeminiEnterpriseDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigGeminiEnterpriseDirect struct {
	ApiKey    *DataSourceAiGatewayModelProviderServiceConfigGeminiEnterpriseDirectApiKey `json:"api_key,omitempty"`
	ProjectId string                                                                     `json:"project_id,omitempty"`
	Region    string                                                                     `json:"region,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigGeminiEnterprise struct {
	Direct *DataSourceAiGatewayModelProviderServiceConfigGeminiEnterpriseDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigInferenceTable struct {
	Disabled        bool   `json:"disabled,omitempty"`
	IsDeleted       bool   `json:"is_deleted,omitempty"`
	Parent          string `json:"parent"`
	Table           string `json:"table,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectEntraServicePrincipalClientSecret struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectEntraServicePrincipal struct {
	ClientId     string                                                                                                `json:"client_id,omitempty"`
	ClientSecret *DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectEntraServicePrincipalClientSecret `json:"client_secret,omitempty"`
	TenantId     string                                                                                                `json:"tenant_id,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectServiceCredential struct {
	Name string `json:"name"`
}

type DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirect struct {
	ApiKey                *DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectApiKey                `json:"api_key,omitempty"`
	BaseUrl               string                                                                                    `json:"base_url,omitempty"`
	EntraServicePrincipal *DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectEntraServicePrincipal `json:"entra_service_principal,omitempty"`
	ServiceCredential     *DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirectServiceCredential     `json:"service_credential,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundry struct {
	Direct *DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundryDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigOpenaiDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigOpenaiDirect struct {
	ApiKey       *DataSourceAiGatewayModelProviderServiceConfigOpenaiDirectApiKey `json:"api_key,omitempty"`
	BaseUrl      string                                                           `json:"base_url,omitempty"`
	Organization string                                                           `json:"organization,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigOpenai struct {
	Direct *DataSourceAiGatewayModelProviderServiceConfigOpenaiDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigRateLimits struct {
	Key             string `json:"key"`
	Principal       string `json:"principal,omitempty"`
	RenewalPeriod   string `json:"renewal_period"`
	RequestTagKey   string `json:"request_tag_key,omitempty"`
	RequestTagValue string `json:"request_tag_value,omitempty"`
	Requests        int    `json:"requests,omitempty"`
	Tokens          int    `json:"tokens,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfigTargets struct {
	Model          string   `json:"model"`
	NativeApiTypes []string `json:"native_api_types,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceConfig struct {
	AllowAllTargets        bool                                                           `json:"allow_all_targets,omitempty"`
	AmazonBedrock          *DataSourceAiGatewayModelProviderServiceConfigAmazonBedrock    `json:"amazon_bedrock,omitempty"`
	Anthropic              *DataSourceAiGatewayModelProviderServiceConfigAnthropic        `json:"anthropic,omitempty"`
	AzureOpenai            *DataSourceAiGatewayModelProviderServiceConfigAzureOpenai      `json:"azure_openai,omitempty"`
	Custom                 *DataSourceAiGatewayModelProviderServiceConfigCustom           `json:"custom,omitempty"`
	ForwardHeaders         bool                                                           `json:"forward_headers,omitempty"`
	ForwardQueryParameters bool                                                           `json:"forward_query_parameters,omitempty"`
	ForwardUnmanagedPaths  bool                                                           `json:"forward_unmanaged_paths,omitempty"`
	GeminiEnterprise       *DataSourceAiGatewayModelProviderServiceConfigGeminiEnterprise `json:"gemini_enterprise,omitempty"`
	InferenceTable         *DataSourceAiGatewayModelProviderServiceConfigInferenceTable   `json:"inference_table,omitempty"`
	MicrosoftFoundry       *DataSourceAiGatewayModelProviderServiceConfigMicrosoftFoundry `json:"microsoft_foundry,omitempty"`
	Openai                 *DataSourceAiGatewayModelProviderServiceConfigOpenai           `json:"openai,omitempty"`
	ProviderType           string                                                         `json:"provider_type,omitempty"`
	RateLimits             []DataSourceAiGatewayModelProviderServiceConfigRateLimits      `json:"rate_limits,omitempty"`
	Targets                []DataSourceAiGatewayModelProviderServiceConfigTargets         `json:"targets,omitempty"`
}

type DataSourceAiGatewayModelProviderServiceProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiGatewayModelProviderService struct {
	Comment        string                                                 `json:"comment,omitempty"`
	Config         *DataSourceAiGatewayModelProviderServiceConfig         `json:"config,omitempty"`
	CreateTime     string                                                 `json:"create_time,omitempty"`
	CreatedBy      string                                                 `json:"created_by,omitempty"`
	EffectiveOwner string                                                 `json:"effective_owner,omitempty"`
	Etag           string                                                 `json:"etag,omitempty"`
	MetastoreId    string                                                 `json:"metastore_id,omitempty"`
	Name           string                                                 `json:"name"`
	Owner          string                                                 `json:"owner,omitempty"`
	ProviderConfig *DataSourceAiGatewayModelProviderServiceProviderConfig `json:"provider_config,omitempty"`
	UpdateTime     string                                                 `json:"update_time,omitempty"`
	UpdatedBy      string                                                 `json:"updated_by,omitempty"`
}
