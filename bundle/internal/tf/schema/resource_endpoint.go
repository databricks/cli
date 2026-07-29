// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceEndpointAwsVpcEndpointInfo struct {
	AwsAccountId         string `json:"aws_account_id,omitempty"`
	AwsEndpointServiceId string `json:"aws_endpoint_service_id,omitempty"`
	AwsVpcEndpointId     string `json:"aws_vpc_endpoint_id"`
}

type ResourceEndpointAzurePrivateEndpointInfo struct {
	PrivateEndpointName         string `json:"private_endpoint_name"`
	PrivateEndpointResourceGuid string `json:"private_endpoint_resource_guid"`
	PrivateEndpointResourceId   string `json:"private_endpoint_resource_id,omitempty"`
	PrivateLinkServiceId        string `json:"private_link_service_id,omitempty"`
}

type ResourceEndpointGcpPscEndpointInfo struct {
	EndpointRegion      string `json:"endpoint_region"`
	ProjectId           string `json:"project_id"`
	PscConnectionId     string `json:"psc_connection_id,omitempty"`
	PscEndpoint         string `json:"psc_endpoint"`
	ServiceAttachmentId string `json:"service_attachment_id,omitempty"`
}

type ResourceEndpoint struct {
	AccountId                string                                    `json:"account_id,omitempty"`
	AwsVpcEndpointInfo       *ResourceEndpointAwsVpcEndpointInfo       `json:"aws_vpc_endpoint_info,omitempty"`
	AzurePrivateEndpointInfo *ResourceEndpointAzurePrivateEndpointInfo `json:"azure_private_endpoint_info,omitempty"`
	CreateTime               string                                    `json:"create_time,omitempty"`
	DisplayName              string                                    `json:"display_name"`
	EndpointId               string                                    `json:"endpoint_id,omitempty"`
	GcpPscEndpointInfo       *ResourceEndpointGcpPscEndpointInfo       `json:"gcp_psc_endpoint_info,omitempty"`
	Name                     string                                    `json:"name,omitempty"`
	Parent                   string                                    `json:"parent"`
	Region                   string                                    `json:"region"`
	State                    string                                    `json:"state,omitempty"`
	UseCase                  string                                    `json:"use_case,omitempty"`
}
