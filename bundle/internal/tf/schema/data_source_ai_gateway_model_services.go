// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAiGatewayModelServicesModelServicesConfigInferenceTable struct {
	Disabled        bool   `json:"disabled,omitempty"`
	IsDeleted       bool   `json:"is_deleted,omitempty"`
	Parent          string `json:"parent"`
	Table           string `json:"table,omitempty"`
	TableNamePrefix string `json:"table_name_prefix,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRateLimits struct {
	Key             string `json:"key"`
	Principal       string `json:"principal,omitempty"`
	RenewalPeriod   string `json:"renewal_period"`
	RequestTagKey   string `json:"request_tag_key,omitempty"`
	RequestTagValue string `json:"request_tag_value,omitempty"`
	Requests        int    `json:"requests,omitempty"`
	Tokens          int    `json:"tokens,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingDestinationsExternalModelConfigTarget struct {
	Model          string   `json:"model"`
	NativeApiTypes []string `json:"native_api_types,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingDestinationsExternalModelConfig struct {
	ModelProviderService string                                                                                           `json:"model_provider_service"`
	Target               *DataSourceAiGatewayModelServicesModelServicesConfigRoutingDestinationsExternalModelConfigTarget `json:"target,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingDestinationsPayPerTokenConfig struct {
	Model string `json:"model"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingDestinationsProvisionedThroughputConfig struct {
	Model                string `json:"model,omitempty"`
	ModelServingEndpoint string `json:"model_serving_endpoint"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingDestinations struct {
	DestinationType             string                                                                                             `json:"destination_type"`
	ExternalModelConfig         *DataSourceAiGatewayModelServicesModelServicesConfigRoutingDestinationsExternalModelConfig         `json:"external_model_config,omitempty"`
	IsDeleted                   bool                                                                                               `json:"is_deleted,omitempty"`
	Name                        string                                                                                             `json:"name"`
	PayPerTokenConfig           *DataSourceAiGatewayModelServicesModelServicesConfigRoutingDestinationsPayPerTokenConfig           `json:"pay_per_token_config,omitempty"`
	ProvisionedThroughputConfig *DataSourceAiGatewayModelServicesModelServicesConfigRoutingDestinationsProvisionedThroughputConfig `json:"provisioned_throughput_config,omitempty"`
	TrafficPercentage           int                                                                                                `json:"traffic_percentage,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallbackDestinationsExternalModelConfigTarget struct {
	Model          string   `json:"model"`
	NativeApiTypes []string `json:"native_api_types,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallbackDestinationsExternalModelConfig struct {
	ModelProviderService string                                                                                                   `json:"model_provider_service"`
	Target               *DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallbackDestinationsExternalModelConfigTarget `json:"target,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallbackDestinationsPayPerTokenConfig struct {
	Model string `json:"model"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallbackDestinationsProvisionedThroughputConfig struct {
	Model                string `json:"model,omitempty"`
	ModelServingEndpoint string `json:"model_serving_endpoint"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallbackDestinations struct {
	DestinationType             string                                                                                                     `json:"destination_type"`
	ExternalModelConfig         *DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallbackDestinationsExternalModelConfig         `json:"external_model_config,omitempty"`
	IsDeleted                   bool                                                                                                       `json:"is_deleted,omitempty"`
	Name                        string                                                                                                     `json:"name"`
	PayPerTokenConfig           *DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallbackDestinationsPayPerTokenConfig           `json:"pay_per_token_config,omitempty"`
	ProvisionedThroughputConfig *DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallbackDestinationsProvisionedThroughputConfig `json:"provisioned_throughput_config,omitempty"`
	TrafficPercentage           int                                                                                                        `json:"traffic_percentage,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallback struct {
	Destinations []DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallbackDestinations `json:"destinations,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServicesConfigRoutingTrafficSplitting struct{}

type DataSourceAiGatewayModelServicesModelServicesConfigRouting struct {
	Destinations      []DataSourceAiGatewayModelServicesModelServicesConfigRoutingDestinations    `json:"destinations,omitempty"`
	Fallback          *DataSourceAiGatewayModelServicesModelServicesConfigRoutingFallback         `json:"fallback,omitempty"`
	FirstTokenTimeout string                                                                      `json:"first_token_timeout,omitempty"`
	TrafficSplitting  *DataSourceAiGatewayModelServicesModelServicesConfigRoutingTrafficSplitting `json:"traffic_splitting,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServicesConfig struct {
	InferenceTable *DataSourceAiGatewayModelServicesModelServicesConfigInferenceTable `json:"inference_table,omitempty"`
	RateLimits     []DataSourceAiGatewayModelServicesModelServicesConfigRateLimits    `json:"rate_limits,omitempty"`
	Routing        *DataSourceAiGatewayModelServicesModelServicesConfigRouting        `json:"routing,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServicesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiGatewayModelServicesModelServices struct {
	BrowseOnly        bool                                                         `json:"browse_only,omitempty"`
	Comment           string                                                       `json:"comment,omitempty"`
	Config            *DataSourceAiGatewayModelServicesModelServicesConfig         `json:"config,omitempty"`
	CreateTime        string                                                       `json:"create_time,omitempty"`
	CreatedBy         string                                                       `json:"created_by,omitempty"`
	EffectiveOwner    string                                                       `json:"effective_owner,omitempty"`
	Etag              string                                                       `json:"etag,omitempty"`
	MetastoreId       string                                                       `json:"metastore_id,omitempty"`
	Name              string                                                       `json:"name"`
	Owner             string                                                       `json:"owner,omitempty"`
	ProviderConfig    *DataSourceAiGatewayModelServicesModelServicesProviderConfig `json:"provider_config,omitempty"`
	SupportedApiTypes []string                                                     `json:"supported_api_types,omitempty"`
	UpdateTime        string                                                       `json:"update_time,omitempty"`
	UpdatedBy         string                                                       `json:"updated_by,omitempty"`
}

type DataSourceAiGatewayModelServicesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiGatewayModelServices struct {
	IncludeBrowse  bool                                            `json:"include_browse,omitempty"`
	ModelServices  []DataSourceAiGatewayModelServicesModelServices `json:"model_services,omitempty"`
	PageSize       int                                             `json:"page_size,omitempty"`
	Parent         string                                          `json:"parent,omitempty"`
	ProviderConfig *DataSourceAiGatewayModelServicesProviderConfig `json:"provider_config,omitempty"`
	View           string                                          `json:"view,omitempty"`
}
