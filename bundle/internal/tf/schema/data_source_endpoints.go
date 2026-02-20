// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceEndpointsItemsAzurePrivateEndpointInfo struct {
	PrivateEndpointName         string `json:"private_endpoint_name"`
	PrivateEndpointResourceGuid string `json:"private_endpoint_resource_guid"`
	PrivateEndpointResourceId   string `json:"private_endpoint_resource_id,omitempty"`
	PrivateLinkServiceId        string `json:"private_link_service_id,omitempty"`
}

type DataSourceEndpointsItems struct {
	AccountId                string                                            `json:"account_id,omitempty"`
	AzurePrivateEndpointInfo *DataSourceEndpointsItemsAzurePrivateEndpointInfo `json:"azure_private_endpoint_info,omitempty"`
	CreateTime               string                                            `json:"create_time,omitempty"`
	DisplayName              string                                            `json:"display_name,omitempty"`
	EndpointId               string                                            `json:"endpoint_id,omitempty"`
	Name                     string                                            `json:"name"`
	Region                   string                                            `json:"region,omitempty"`
	State                    string                                            `json:"state,omitempty"`
	UseCase                  string                                            `json:"use_case,omitempty"`
}

type DataSourceEndpoints struct {
	Items    []DataSourceEndpointsItems `json:"items,omitempty"`
	PageSize int                        `json:"page_size,omitempty"`
	Parent   string                     `json:"parent"`
}
