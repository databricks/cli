// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceEndpointAzurePrivateEndpointInfo struct {
	PrivateEndpointName         string `json:"private_endpoint_name"`
	PrivateEndpointResourceGuid string `json:"private_endpoint_resource_guid"`
	PrivateEndpointResourceId   string `json:"private_endpoint_resource_id,omitempty"`
	PrivateLinkServiceId        string `json:"private_link_service_id,omitempty"`
}

type ResourceEndpoint struct {
	AccountId                string                                    `json:"account_id,omitempty"`
	AzurePrivateEndpointInfo *ResourceEndpointAzurePrivateEndpointInfo `json:"azure_private_endpoint_info,omitempty"`
	CreateTime               string                                    `json:"create_time,omitempty"`
	DisplayName              string                                    `json:"display_name"`
	EndpointId               string                                    `json:"endpoint_id,omitempty"`
	Name                     string                                    `json:"name,omitempty"`
	Parent                   string                                    `json:"parent"`
	Region                   string                                    `json:"region"`
	State                    string                                    `json:"state,omitempty"`
	UseCase                  string                                    `json:"use_case,omitempty"`
}
