// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAiGatewayModelServiceConfigInferenceTable struct {
	Disabled        bool   `json:"disabled,omitempty"`
	IsDeleted       bool   `json:"is_deleted,omitempty"`
	Parent          string `json:"parent"`
	Table           string `json:"table,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type ResourceAiGatewayModelServiceConfigRateLimits struct {
	Key             string `json:"key"`
	Principal       string `json:"principal,omitempty"`
	RenewalPeriod   string `json:"renewal_period"`
	RequestTagKey   string `json:"request_tag_key,omitempty"`
	RequestTagValue string `json:"request_tag_value,omitempty"`
	Requests        int    `json:"requests,omitempty"`
	Tokens          int    `json:"tokens,omitempty"`
}

type ResourceAiGatewayModelServiceConfigRoutingDestinationsExternalModelConfigTarget struct {
	Model          string   `json:"model"`
	NativeApiTypes []string `json:"native_api_types,omitempty"`
}

type ResourceAiGatewayModelServiceConfigRoutingDestinationsExternalModelConfig struct {
	ModelProviderService string                                                                           `json:"model_provider_service"`
	Target               *ResourceAiGatewayModelServiceConfigRoutingDestinationsExternalModelConfigTarget `json:"target,omitempty"`
}

type ResourceAiGatewayModelServiceConfigRoutingDestinationsPayPerTokenConfig struct {
	Model string `json:"model"`
}

type ResourceAiGatewayModelServiceConfigRoutingDestinationsProvisionedThroughputConfig struct {
	Model                string `json:"model,omitempty"`
	ModelServingEndpoint string `json:"model_serving_endpoint"`
}

type ResourceAiGatewayModelServiceConfigRoutingDestinations struct {
	DestinationType             string                                                                             `json:"destination_type"`
	ExternalModelConfig         *ResourceAiGatewayModelServiceConfigRoutingDestinationsExternalModelConfig         `json:"external_model_config,omitempty"`
	IsDeleted                   bool                                                                               `json:"is_deleted,omitempty"`
	Name                        string                                                                             `json:"name"`
	PayPerTokenConfig           *ResourceAiGatewayModelServiceConfigRoutingDestinationsPayPerTokenConfig           `json:"pay_per_token_config,omitempty"`
	ProvisionedThroughputConfig *ResourceAiGatewayModelServiceConfigRoutingDestinationsProvisionedThroughputConfig `json:"provisioned_throughput_config,omitempty"`
	TrafficPercentage           int                                                                                `json:"traffic_percentage,omitempty"`
}

type ResourceAiGatewayModelServiceConfigRoutingFallbackDestinationsExternalModelConfigTarget struct {
	Model          string   `json:"model"`
	NativeApiTypes []string `json:"native_api_types,omitempty"`
}

type ResourceAiGatewayModelServiceConfigRoutingFallbackDestinationsExternalModelConfig struct {
	ModelProviderService string                                                                                   `json:"model_provider_service"`
	Target               *ResourceAiGatewayModelServiceConfigRoutingFallbackDestinationsExternalModelConfigTarget `json:"target,omitempty"`
}

type ResourceAiGatewayModelServiceConfigRoutingFallbackDestinationsPayPerTokenConfig struct {
	Model string `json:"model"`
}

type ResourceAiGatewayModelServiceConfigRoutingFallbackDestinationsProvisionedThroughputConfig struct {
	Model                string `json:"model,omitempty"`
	ModelServingEndpoint string `json:"model_serving_endpoint"`
}

type ResourceAiGatewayModelServiceConfigRoutingFallbackDestinations struct {
	DestinationType             string                                                                                     `json:"destination_type"`
	ExternalModelConfig         *ResourceAiGatewayModelServiceConfigRoutingFallbackDestinationsExternalModelConfig         `json:"external_model_config,omitempty"`
	IsDeleted                   bool                                                                                       `json:"is_deleted,omitempty"`
	Name                        string                                                                                     `json:"name"`
	PayPerTokenConfig           *ResourceAiGatewayModelServiceConfigRoutingFallbackDestinationsPayPerTokenConfig           `json:"pay_per_token_config,omitempty"`
	ProvisionedThroughputConfig *ResourceAiGatewayModelServiceConfigRoutingFallbackDestinationsProvisionedThroughputConfig `json:"provisioned_throughput_config,omitempty"`
	TrafficPercentage           int                                                                                        `json:"traffic_percentage,omitempty"`
}

type ResourceAiGatewayModelServiceConfigRoutingFallback struct {
	Destinations []ResourceAiGatewayModelServiceConfigRoutingFallbackDestinations `json:"destinations,omitempty"`
}

type ResourceAiGatewayModelServiceConfigRoutingTrafficSplitting struct{}

type ResourceAiGatewayModelServiceConfigRouting struct {
	Destinations      []ResourceAiGatewayModelServiceConfigRoutingDestinations    `json:"destinations,omitempty"`
	Fallback          *ResourceAiGatewayModelServiceConfigRoutingFallback         `json:"fallback,omitempty"`
	FirstTokenTimeout string                                                      `json:"first_token_timeout,omitempty"`
	TrafficSplitting  *ResourceAiGatewayModelServiceConfigRoutingTrafficSplitting `json:"traffic_splitting,omitempty"`
}

type ResourceAiGatewayModelServiceConfig struct {
	InferenceTable *ResourceAiGatewayModelServiceConfigInferenceTable `json:"inference_table,omitempty"`
	RateLimits     []ResourceAiGatewayModelServiceConfigRateLimits    `json:"rate_limits,omitempty"`
	Routing        *ResourceAiGatewayModelServiceConfigRouting        `json:"routing,omitempty"`
}

type ResourceAiGatewayModelServiceProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceAiGatewayModelService struct {
	BrowseOnly        bool                                         `json:"browse_only,omitempty"`
	Comment           string                                       `json:"comment,omitempty"`
	Config            *ResourceAiGatewayModelServiceConfig         `json:"config,omitempty"`
	CreateTime        string                                       `json:"create_time,omitempty"`
	CreatedBy         string                                       `json:"created_by,omitempty"`
	EffectiveOwner    string                                       `json:"effective_owner,omitempty"`
	Etag              string                                       `json:"etag,omitempty"`
	MetastoreId       string                                       `json:"metastore_id,omitempty"`
	ModelServiceId    string                                       `json:"model_service_id"`
	Name              string                                       `json:"name,omitempty"`
	Owner             string                                       `json:"owner,omitempty"`
	Parent            string                                       `json:"parent"`
	ProviderConfig    *ResourceAiGatewayModelServiceProviderConfig `json:"provider_config,omitempty"`
	SupportedApiTypes []string                                     `json:"supported_api_types,omitempty"`
	UpdateTime        string                                       `json:"update_time,omitempty"`
	UpdatedBy         string                                       `json:"updated_by,omitempty"`
}
