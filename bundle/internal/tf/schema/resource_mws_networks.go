// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMwsNetworksErrorMessages struct {
	ErrorMessage string `json:"error_message,omitempty"`
	ErrorType    string `json:"error_type,omitempty"`
}

type ResourceMwsNetworksGcpNetworkInfo struct {
	NetworkProjectId   string `json:"network_project_id"`
	PodIpRangeName     string `json:"pod_ip_range_name,omitempty"`
	ServiceIpRangeName string `json:"service_ip_range_name,omitempty"`
	SubnetId           string `json:"subnet_id"`
	SubnetRegion       string `json:"subnet_region"`
	VpcId              string `json:"vpc_id"`
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
	SecurityGroupIds []string                           `json:"security_group_ids,omitempty"`
	SubnetIds        []string                           `json:"subnet_ids,omitempty"`
	VpcId            string                             `json:"vpc_id,omitempty"`
	VpcStatus        string                             `json:"vpc_status,omitempty"`
	WorkspaceId      int                                `json:"workspace_id,omitempty"`
	ErrorMessages    []ResourceMwsNetworksErrorMessages `json:"error_messages,omitempty"`
	GcpNetworkInfo   *ResourceMwsNetworksGcpNetworkInfo `json:"gcp_network_info,omitempty"`
	VpcEndpoints     *ResourceMwsNetworksVpcEndpoints   `json:"vpc_endpoints,omitempty"`
}
