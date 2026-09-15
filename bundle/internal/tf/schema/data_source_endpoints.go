// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceEndpointsItemsAwsVpcEndpointInfo struct {
	AwsAccountId         string `json:"aws_account_id,omitempty"`
	AwsEndpointServiceId string `json:"aws_endpoint_service_id,omitempty"`
	AwsVpcEndpointId     string `json:"aws_vpc_endpoint_id"`
}

type DataSourceEndpointsItemsAzurePrivateEndpointInfo struct {
	PrivateEndpointName         string `json:"private_endpoint_name"`
	PrivateEndpointResourceGuid string `json:"private_endpoint_resource_guid"`
	PrivateEndpointResourceId   string `json:"private_endpoint_resource_id,omitempty"`
	PrivateLinkServiceId        string `json:"private_link_service_id,omitempty"`
}

type DataSourceEndpointsItemsGcpPscEndpointInfo struct {
	EndpointRegion      string `json:"endpoint_region"`
	ProjectId           string `json:"project_id"`
	PscConnectionId     string `json:"psc_connection_id,omitempty"`
	PscEndpoint         string `json:"psc_endpoint"`
	ServiceAttachmentId string `json:"service_attachment_id,omitempty"`
}

type DataSourceEndpointsItems struct {
	AccountId                string                                            `json:"account_id,omitempty"`
	AwsVpcEndpointInfo       *DataSourceEndpointsItemsAwsVpcEndpointInfo       `json:"aws_vpc_endpoint_info,omitempty"`
	AzurePrivateEndpointInfo *DataSourceEndpointsItemsAzurePrivateEndpointInfo `json:"azure_private_endpoint_info,omitempty"`
	CreateTime               string                                            `json:"create_time,omitempty"`
	DisplayName              string                                            `json:"display_name,omitempty"`
	EndpointId               string                                            `json:"endpoint_id,omitempty"`
	GcpPscEndpointInfo       *DataSourceEndpointsItemsGcpPscEndpointInfo       `json:"gcp_psc_endpoint_info,omitempty"`
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
