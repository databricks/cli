// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAiGatewayMcpServiceConfigRateLimits struct {
	Key             string `json:"key"`
	Principal       string `json:"principal,omitempty"`
	RenewalPeriod   string `json:"renewal_period"`
	RequestTagKey   string `json:"request_tag_key,omitempty"`
	RequestTagValue string `json:"request_tag_value,omitempty"`
	Requests        int    `json:"requests,omitempty"`
	Tokens          int    `json:"tokens,omitempty"`
}

type DataSourceAiGatewayMcpServiceConfigSourceConnection struct {
	IsDeleted bool   `json:"is_deleted,omitempty"`
	Name      string `json:"name"`
}

type DataSourceAiGatewayMcpServiceConfig struct {
	IncludeToolSelectors []string                                             `json:"include_tool_selectors,omitempty"`
	RateLimits           []DataSourceAiGatewayMcpServiceConfigRateLimits      `json:"rate_limits,omitempty"`
	SourceConnection     *DataSourceAiGatewayMcpServiceConfigSourceConnection `json:"source_connection,omitempty"`
}

type DataSourceAiGatewayMcpServiceProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiGatewayMcpService struct {
	Comment        string                                       `json:"comment,omitempty"`
	Config         *DataSourceAiGatewayMcpServiceConfig         `json:"config,omitempty"`
	CreateTime     string                                       `json:"create_time,omitempty"`
	CreatedBy      string                                       `json:"created_by,omitempty"`
	EffectiveOwner string                                       `json:"effective_owner,omitempty"`
	Etag           string                                       `json:"etag,omitempty"`
	MetastoreId    string                                       `json:"metastore_id,omitempty"`
	Name           string                                       `json:"name"`
	Owner          string                                       `json:"owner,omitempty"`
	ProviderConfig *DataSourceAiGatewayMcpServiceProviderConfig `json:"provider_config,omitempty"`
	UpdateTime     string                                       `json:"update_time,omitempty"`
	UpdatedBy      string                                       `json:"updated_by,omitempty"`
}
