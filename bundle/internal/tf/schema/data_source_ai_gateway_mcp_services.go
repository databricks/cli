// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAiGatewayMcpServicesMcpServicesConfigRateLimits struct {
	Key             string `json:"key"`
	Principal       string `json:"principal,omitempty"`
	RenewalPeriod   string `json:"renewal_period"`
	RequestTagKey   string `json:"request_tag_key,omitempty"`
	RequestTagValue string `json:"request_tag_value,omitempty"`
	Requests        int    `json:"requests,omitempty"`
	Tokens          int    `json:"tokens,omitempty"`
}

type DataSourceAiGatewayMcpServicesMcpServicesConfigSourceConnection struct {
	IsDeleted bool   `json:"is_deleted,omitempty"`
	Name      string `json:"name"`
}

type DataSourceAiGatewayMcpServicesMcpServicesConfig struct {
	IncludeToolSelectors []string                                                         `json:"include_tool_selectors,omitempty"`
	RateLimits           []DataSourceAiGatewayMcpServicesMcpServicesConfigRateLimits      `json:"rate_limits,omitempty"`
	SourceConnection     *DataSourceAiGatewayMcpServicesMcpServicesConfigSourceConnection `json:"source_connection,omitempty"`
}

type DataSourceAiGatewayMcpServicesMcpServicesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiGatewayMcpServicesMcpServices struct {
	Comment        string                                                   `json:"comment,omitempty"`
	Config         *DataSourceAiGatewayMcpServicesMcpServicesConfig         `json:"config,omitempty"`
	CreateTime     string                                                   `json:"create_time,omitempty"`
	CreatedBy      string                                                   `json:"created_by,omitempty"`
	EffectiveOwner string                                                   `json:"effective_owner,omitempty"`
	Etag           string                                                   `json:"etag,omitempty"`
	MetastoreId    string                                                   `json:"metastore_id,omitempty"`
	Name           string                                                   `json:"name"`
	Owner          string                                                   `json:"owner,omitempty"`
	ProviderConfig *DataSourceAiGatewayMcpServicesMcpServicesProviderConfig `json:"provider_config,omitempty"`
	UpdateTime     string                                                   `json:"update_time,omitempty"`
	UpdatedBy      string                                                   `json:"updated_by,omitempty"`
}

type DataSourceAiGatewayMcpServicesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiGatewayMcpServices struct {
	McpServices    []DataSourceAiGatewayMcpServicesMcpServices   `json:"mcp_services,omitempty"`
	PageSize       int                                           `json:"page_size,omitempty"`
	Parent         string                                        `json:"parent,omitempty"`
	ProviderConfig *DataSourceAiGatewayMcpServicesProviderConfig `json:"provider_config,omitempty"`
	View           string                                        `json:"view,omitempty"`
}
