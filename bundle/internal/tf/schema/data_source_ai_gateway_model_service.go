// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAiGatewayModelServiceConfigInferenceTable struct {
	Disabled        bool   `json:"disabled,omitempty"`
	IsDeleted       bool   `json:"is_deleted,omitempty"`
	Parent          string `json:"parent"`
	Table           string `json:"table,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type DataSourceAiGatewayModelServiceConfigRateLimits struct {
	Key             string `json:"key"`
	Principal       string `json:"principal,omitempty"`
	RenewalPeriod   string `json:"renewal_period"`
	RequestTagKey   string `json:"request_tag_key,omitempty"`
	RequestTagValue string `json:"request_tag_value,omitempty"`
	Requests        int    `json:"requests,omitempty"`
	Tokens          int    `json:"tokens,omitempty"`
}

type DataSourceAiGatewayModelServiceConfigRoutingDestinationsExternalModelConfigTarget struct {
	Model          string   `json:"model"`
	NativeApiTypes []string `json:"native_api_types,omitempty"`
}

type DataSourceAiGatewayModelServiceConfigRoutingDestinationsExternalModelConfig struct {
	ModelProviderService string                                                                             `json:"model_provider_service"`
	Target               *DataSourceAiGatewayModelServiceConfigRoutingDestinationsExternalModelConfigTarget `json:"target,omitempty"`
}

type DataSourceAiGatewayModelServiceConfigRoutingDestinationsPayPerTokenConfig struct {
	Model string `json:"model"`
}

type DataSourceAiGatewayModelServiceConfigRoutingDestinationsProvisionedThroughputConfig struct {
	Model                string `json:"model,omitempty"`
	ModelServingEndpoint string `json:"model_serving_endpoint"`
}

type DataSourceAiGatewayModelServiceConfigRoutingDestinations struct {
	DestinationType             string                                                                               `json:"destination_type"`
	ExternalModelConfig         *DataSourceAiGatewayModelServiceConfigRoutingDestinationsExternalModelConfig         `json:"external_model_config,omitempty"`
	IsDeleted                   bool                                                                                 `json:"is_deleted,omitempty"`
	Name                        string                                                                               `json:"name"`
	PayPerTokenConfig           *DataSourceAiGatewayModelServiceConfigRoutingDestinationsPayPerTokenConfig           `json:"pay_per_token_config,omitempty"`
	ProvisionedThroughputConfig *DataSourceAiGatewayModelServiceConfigRoutingDestinationsProvisionedThroughputConfig `json:"provisioned_throughput_config,omitempty"`
	TrafficPercentage           int                                                                                  `json:"traffic_percentage,omitempty"`
}

type DataSourceAiGatewayModelServiceConfigRoutingFallbackDestinationsExternalModelConfigTarget struct {
	Model          string   `json:"model"`
	NativeApiTypes []string `json:"native_api_types,omitempty"`
}

type DataSourceAiGatewayModelServiceConfigRoutingFallbackDestinationsExternalModelConfig struct {
	ModelProviderService string                                                                                     `json:"model_provider_service"`
	Target               *DataSourceAiGatewayModelServiceConfigRoutingFallbackDestinationsExternalModelConfigTarget `json:"target,omitempty"`
}

type DataSourceAiGatewayModelServiceConfigRoutingFallbackDestinationsPayPerTokenConfig struct {
	Model string `json:"model"`
}

type DataSourceAiGatewayModelServiceConfigRoutingFallbackDestinationsProvisionedThroughputConfig struct {
	Model                string `json:"model,omitempty"`
	ModelServingEndpoint string `json:"model_serving_endpoint"`
}

type DataSourceAiGatewayModelServiceConfigRoutingFallbackDestinations struct {
	DestinationType             string                                                                                       `json:"destination_type"`
	ExternalModelConfig         *DataSourceAiGatewayModelServiceConfigRoutingFallbackDestinationsExternalModelConfig         `json:"external_model_config,omitempty"`
	IsDeleted                   bool                                                                                         `json:"is_deleted,omitempty"`
	Name                        string                                                                                       `json:"name"`
	PayPerTokenConfig           *DataSourceAiGatewayModelServiceConfigRoutingFallbackDestinationsPayPerTokenConfig           `json:"pay_per_token_config,omitempty"`
	ProvisionedThroughputConfig *DataSourceAiGatewayModelServiceConfigRoutingFallbackDestinationsProvisionedThroughputConfig `json:"provisioned_throughput_config,omitempty"`
	TrafficPercentage           int                                                                                          `json:"traffic_percentage,omitempty"`
}

type DataSourceAiGatewayModelServiceConfigRoutingFallback struct {
	Destinations []DataSourceAiGatewayModelServiceConfigRoutingFallbackDestinations `json:"destinations,omitempty"`
}

type DataSourceAiGatewayModelServiceConfigRoutingTrafficSplitting struct{}

type DataSourceAiGatewayModelServiceConfigRouting struct {
	Destinations      []DataSourceAiGatewayModelServiceConfigRoutingDestinations    `json:"destinations,omitempty"`
	Fallback          *DataSourceAiGatewayModelServiceConfigRoutingFallback         `json:"fallback,omitempty"`
	FirstTokenTimeout string                                                        `json:"first_token_timeout,omitempty"`
	TrafficSplitting  *DataSourceAiGatewayModelServiceConfigRoutingTrafficSplitting `json:"traffic_splitting,omitempty"`
}

type DataSourceAiGatewayModelServiceConfig struct {
	InferenceTable *DataSourceAiGatewayModelServiceConfigInferenceTable `json:"inference_table,omitempty"`
	RateLimits     []DataSourceAiGatewayModelServiceConfigRateLimits    `json:"rate_limits,omitempty"`
	Routing        *DataSourceAiGatewayModelServiceConfigRouting        `json:"routing,omitempty"`
}

type DataSourceAiGatewayModelServiceProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiGatewayModelService struct {
	BrowseOnly        bool                                           `json:"browse_only,omitempty"`
	Comment           string                                         `json:"comment,omitempty"`
	Config            *DataSourceAiGatewayModelServiceConfig         `json:"config,omitempty"`
	CreateTime        string                                         `json:"create_time,omitempty"`
	CreatedBy         string                                         `json:"created_by,omitempty"`
	EffectiveOwner    string                                         `json:"effective_owner,omitempty"`
	Etag              string                                         `json:"etag,omitempty"`
	MetastoreId       string                                         `json:"metastore_id,omitempty"`
	Name              string                                         `json:"name"`
	Owner             string                                         `json:"owner,omitempty"`
	ProviderConfig    *DataSourceAiGatewayModelServiceProviderConfig `json:"provider_config,omitempty"`
	SupportedApiTypes []string                                       `json:"supported_api_types,omitempty"`
	UpdateTime        string                                         `json:"update_time,omitempty"`
	UpdatedBy         string                                         `json:"updated_by,omitempty"`
}
