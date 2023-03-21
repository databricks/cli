// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMwsVpcEndpointGcpVpcEndpointInfo struct {
	EndpointRegion      string `json:"endpoint_region"`
	ProjectId           string `json:"project_id"`
	PscConnectionId     string `json:"psc_connection_id,omitempty"`
	PscEndpointName     string `json:"psc_endpoint_name"`
	ServiceAttachmentId string `json:"service_attachment_id,omitempty"`
}

type ResourceMwsVpcEndpoint struct {
	AccountId            string                                    `json:"account_id,omitempty"`
	AwsAccountId         string                                    `json:"aws_account_id,omitempty"`
	AwsEndpointServiceId string                                    `json:"aws_endpoint_service_id,omitempty"`
	AwsVpcEndpointId     string                                    `json:"aws_vpc_endpoint_id,omitempty"`
	Id                   string                                    `json:"id,omitempty"`
	Region               string                                    `json:"region,omitempty"`
	State                string                                    `json:"state,omitempty"`
	UseCase              string                                    `json:"use_case,omitempty"`
	VpcEndpointId        string                                    `json:"vpc_endpoint_id,omitempty"`
	VpcEndpointName      string                                    `json:"vpc_endpoint_name"`
	GcpVpcEndpointInfo   *ResourceMwsVpcEndpointGcpVpcEndpointInfo `json:"gcp_vpc_endpoint_info,omitempty"`
}
