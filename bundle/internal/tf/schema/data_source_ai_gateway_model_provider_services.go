// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAmazonBedrockDirectAwsSecretAccessKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAmazonBedrockDirectServiceCredential struct {
	Name string `json:"name"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAmazonBedrockDirect struct {
	AwsAccessKeyId     string                                                                                                    `json:"aws_access_key_id,omitempty"`
	AwsSecretAccessKey *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAmazonBedrockDirectAwsSecretAccessKey `json:"aws_secret_access_key,omitempty"`
	Region             string                                                                                                    `json:"region,omitempty"`
	ServiceCredential  *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAmazonBedrockDirectServiceCredential  `json:"service_credential,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAmazonBedrock struct {
	Direct *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAmazonBedrockDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAnthropicDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAnthropicDirect struct {
	ApiKey *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAnthropicDirectApiKey `json:"api_key,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAnthropicRelayed struct {
	PlanType string `json:"plan_type,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAnthropic struct {
	Direct  *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAnthropicDirect  `json:"direct,omitempty"`
	Relayed *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAnthropicRelayed `json:"relayed,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAzureOpenaiDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAzureOpenaiDirectClientSecret struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAzureOpenaiDirectServiceCredential struct {
	Name string `json:"name"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAzureOpenaiDirect struct {
	ApiKey            *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAzureOpenaiDirectApiKey            `json:"api_key,omitempty"`
	BaseUrl           string                                                                                                 `json:"base_url,omitempty"`
	ClientId          string                                                                                                 `json:"client_id,omitempty"`
	ClientSecret      *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAzureOpenaiDirectClientSecret      `json:"client_secret,omitempty"`
	ServiceCredential *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAzureOpenaiDirectServiceCredential `json:"service_credential,omitempty"`
	TenantId          string                                                                                                 `json:"tenant_id,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAzureOpenai struct {
	Direct *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAzureOpenaiDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigCustomDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigCustomDirect struct {
	ApiKey  *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigCustomDirectApiKey `json:"api_key,omitempty"`
	BaseUrl string                                                                                 `json:"base_url,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigCustom struct {
	Direct *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigCustomDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigGeminiEnterpriseDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigGeminiEnterpriseDirect struct {
	ApiKey    *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigGeminiEnterpriseDirectApiKey `json:"api_key,omitempty"`
	ProjectId string                                                                                           `json:"project_id,omitempty"`
	Region    string                                                                                           `json:"region,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigGeminiEnterprise struct {
	Direct *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigGeminiEnterpriseDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigInferenceTable struct {
	Disabled        bool   `json:"disabled,omitempty"`
	IsDeleted       bool   `json:"is_deleted,omitempty"`
	Parent          string `json:"parent"`
	Table           string `json:"table,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigMicrosoftFoundryDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigMicrosoftFoundryDirectClientSecret struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigMicrosoftFoundryDirectServiceCredential struct {
	Name string `json:"name"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigMicrosoftFoundryDirect struct {
	ApiKey            *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigMicrosoftFoundryDirectApiKey            `json:"api_key,omitempty"`
	BaseUrl           string                                                                                                      `json:"base_url,omitempty"`
	ClientId          string                                                                                                      `json:"client_id,omitempty"`
	ClientSecret      *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigMicrosoftFoundryDirectClientSecret      `json:"client_secret,omitempty"`
	ServiceCredential *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigMicrosoftFoundryDirectServiceCredential `json:"service_credential,omitempty"`
	TenantId          string                                                                                                      `json:"tenant_id,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigMicrosoftFoundry struct {
	Direct *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigMicrosoftFoundryDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigOpenaiDirectApiKey struct {
	Plaintext string `json:"plaintext,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigOpenaiDirect struct {
	ApiKey       *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigOpenaiDirectApiKey `json:"api_key,omitempty"`
	BaseUrl      string                                                                                 `json:"base_url,omitempty"`
	Organization string                                                                                 `json:"organization,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigOpenai struct {
	Direct *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigOpenaiDirect `json:"direct,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigRateLimits struct {
	Key             string `json:"key"`
	Principal       string `json:"principal,omitempty"`
	RenewalPeriod   string `json:"renewal_period"`
	RequestTagKey   string `json:"request_tag_key,omitempty"`
	RequestTagValue string `json:"request_tag_value,omitempty"`
	Requests        int    `json:"requests,omitempty"`
	Tokens          int    `json:"tokens,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigTargets struct {
	Model          string   `json:"model"`
	NativeApiTypes []string `json:"native_api_types,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesConfig struct {
	AllowAllTargets        bool                                                                                 `json:"allow_all_targets,omitempty"`
	AmazonBedrock          *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAmazonBedrock    `json:"amazon_bedrock,omitempty"`
	Anthropic              *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAnthropic        `json:"anthropic,omitempty"`
	AzureOpenai            *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigAzureOpenai      `json:"azure_openai,omitempty"`
	Custom                 *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigCustom           `json:"custom,omitempty"`
	ForwardHeaders         bool                                                                                 `json:"forward_headers,omitempty"`
	ForwardQueryParameters bool                                                                                 `json:"forward_query_parameters,omitempty"`
	ForwardUnmanagedPaths  bool                                                                                 `json:"forward_unmanaged_paths,omitempty"`
	GeminiEnterprise       *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigGeminiEnterprise `json:"gemini_enterprise,omitempty"`
	InferenceTable         *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigInferenceTable   `json:"inference_table,omitempty"`
	MicrosoftFoundry       *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigMicrosoftFoundry `json:"microsoft_foundry,omitempty"`
	Openai                 *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigOpenai           `json:"openai,omitempty"`
	ProviderType           string                                                                               `json:"provider_type,omitempty"`
	RateLimits             []DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigRateLimits      `json:"rate_limits,omitempty"`
	Targets                []DataSourceAiGatewayModelProviderServicesModelProviderServicesConfigTargets         `json:"targets,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServicesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesModelProviderServices struct {
	BrowseOnly     bool                                                                         `json:"browse_only,omitempty"`
	Comment        string                                                                       `json:"comment,omitempty"`
	Config         *DataSourceAiGatewayModelProviderServicesModelProviderServicesConfig         `json:"config,omitempty"`
	CreateTime     string                                                                       `json:"create_time,omitempty"`
	CreatedBy      string                                                                       `json:"created_by,omitempty"`
	EffectiveOwner string                                                                       `json:"effective_owner,omitempty"`
	Etag           string                                                                       `json:"etag,omitempty"`
	MetastoreId    string                                                                       `json:"metastore_id,omitempty"`
	Name           string                                                                       `json:"name"`
	Owner          string                                                                       `json:"owner,omitempty"`
	ProviderConfig *DataSourceAiGatewayModelProviderServicesModelProviderServicesProviderConfig `json:"provider_config,omitempty"`
	UpdateTime     string                                                                       `json:"update_time,omitempty"`
	UpdatedBy      string                                                                       `json:"updated_by,omitempty"`
}

type DataSourceAiGatewayModelProviderServicesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiGatewayModelProviderServices struct {
	IncludeBrowse         bool                                                            `json:"include_browse,omitempty"`
	ModelProviderServices []DataSourceAiGatewayModelProviderServicesModelProviderServices `json:"model_provider_services,omitempty"`
	PageSize              int                                                             `json:"page_size,omitempty"`
	Parent                string                                                          `json:"parent,omitempty"`
	ProviderConfig        *DataSourceAiGatewayModelProviderServicesProviderConfig         `json:"provider_config,omitempty"`
	View                  string                                                          `json:"view,omitempty"`
}
