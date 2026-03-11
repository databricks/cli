// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceIpAccessListProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceIpAccessList struct {
	Enabled        bool                                `json:"enabled,omitempty"`
	Id             string                              `json:"id,omitempty"`
	IpAddresses    []string                            `json:"ip_addresses"`
	Label          string                              `json:"label"`
	ListType       string                              `json:"list_type"`
	ProviderConfig *ResourceIpAccessListProviderConfig `json:"provider_config,omitempty"`
}
