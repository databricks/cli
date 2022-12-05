// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type ResourceMwsVpcEndpoint struct {
	AccountId            string `json:"account_id,omitempty"`
	AwsAccountId         string `json:"aws_account_id,omitempty"`
	AwsEndpointServiceId string `json:"aws_endpoint_service_id,omitempty"`
	AwsVpcEndpointId     string `json:"aws_vpc_endpoint_id"`
	Id                   string `json:"id,omitempty"`
	Region               string `json:"region"`
	State                string `json:"state,omitempty"`
	UseCase              string `json:"use_case,omitempty"`
	VpcEndpointId        string `json:"vpc_endpoint_id,omitempty"`
	VpcEndpointName      string `json:"vpc_endpoint_name"`
}
