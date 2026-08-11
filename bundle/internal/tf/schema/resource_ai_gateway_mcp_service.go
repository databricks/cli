// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAiGatewayMcpServiceConfigRateLimits struct {
	Key             string `json:"key"`
	Principal       string `json:"principal,omitempty"`
	RenewalPeriod   string `json:"renewal_period"`
	RequestTagKey   string `json:"request_tag_key,omitempty"`
	RequestTagValue string `json:"request_tag_value,omitempty"`
	Requests        int    `json:"requests,omitempty"`
	Tokens          int    `json:"tokens,omitempty"`
}

type ResourceAiGatewayMcpServiceConfigSourceConnection struct {
	IsDeleted bool   `json:"is_deleted,omitempty"`
	Name      string `json:"name"`
}

type ResourceAiGatewayMcpServiceConfig struct {
	IncludeToolSelectors []string                                           `json:"include_tool_selectors,omitempty"`
	RateLimits           []ResourceAiGatewayMcpServiceConfigRateLimits      `json:"rate_limits,omitempty"`
	SourceConnection     *ResourceAiGatewayMcpServiceConfigSourceConnection `json:"source_connection,omitempty"`
}

type ResourceAiGatewayMcpServiceProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceAiGatewayMcpService struct {
	BrowseOnly     bool                                       `json:"browse_only,omitempty"`
	Comment        string                                     `json:"comment,omitempty"`
	Config         *ResourceAiGatewayMcpServiceConfig         `json:"config,omitempty"`
	CreateTime     string                                     `json:"create_time,omitempty"`
	CreatedBy      string                                     `json:"created_by,omitempty"`
	EffectiveOwner string                                     `json:"effective_owner,omitempty"`
	Etag           string                                     `json:"etag,omitempty"`
	McpServiceId   string                                     `json:"mcp_service_id"`
	MetastoreId    string                                     `json:"metastore_id,omitempty"`
	Name           string                                     `json:"name,omitempty"`
	Owner          string                                     `json:"owner,omitempty"`
	Parent         string                                     `json:"parent"`
	ProviderConfig *ResourceAiGatewayMcpServiceProviderConfig `json:"provider_config,omitempty"`
	UpdateTime     string                                     `json:"update_time,omitempty"`
	UpdatedBy      string                                     `json:"updated_by,omitempty"`
}
