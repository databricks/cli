// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMwsNetworksErrorMessages struct {
	ErrorMessage string `json:"error_message,omitempty"`
	ErrorType    string `json:"error_type,omitempty"`
}

type ResourceMwsNetworksVpcEndpoints struct {
	DataplaneRelay []string `json:"dataplane_relay"`
	RestApi        []string `json:"rest_api"`
}

type ResourceMwsNetworks struct {
	AccountId        string                             `json:"account_id"`
	CreationTime     int                                `json:"creation_time,omitempty"`
	Id               string                             `json:"id,omitempty"`
	NetworkId        string                             `json:"network_id,omitempty"`
	NetworkName      string                             `json:"network_name"`
	SecurityGroupIds []string                           `json:"security_group_ids"`
	SubnetIds        []string                           `json:"subnet_ids"`
	VpcId            string                             `json:"vpc_id"`
	VpcStatus        string                             `json:"vpc_status,omitempty"`
	WorkspaceId      int                                `json:"workspace_id,omitempty"`
	ErrorMessages    []ResourceMwsNetworksErrorMessages `json:"error_messages,omitempty"`
	VpcEndpoints     *ResourceMwsNetworksVpcEndpoints   `json:"vpc_endpoints,omitempty"`
}
