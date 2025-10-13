// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMwsNetworkConnectivityConfigEgressConfigDefaultRulesAwsStableIpRule struct {
	CidrBlocks []string `json:"cidr_blocks,omitempty"`
}

type ResourceMwsNetworkConnectivityConfigEgressConfigDefaultRulesAzureServiceEndpointRule struct {
	Subnets        []string `json:"subnets,omitempty"`
	TargetRegion   string   `json:"target_region,omitempty"`
	TargetServices []string `json:"target_services,omitempty"`
}

type ResourceMwsNetworkConnectivityConfigEgressConfigDefaultRules struct {
	AwsStableIpRule          *ResourceMwsNetworkConnectivityConfigEgressConfigDefaultRulesAwsStableIpRule          `json:"aws_stable_ip_rule,omitempty"`
	AzureServiceEndpointRule *ResourceMwsNetworkConnectivityConfigEgressConfigDefaultRulesAzureServiceEndpointRule `json:"azure_service_endpoint_rule,omitempty"`
}

type ResourceMwsNetworkConnectivityConfigEgressConfigTargetRulesAwsPrivateEndpointRules struct {
	AccountId                   string   `json:"account_id,omitempty"`
	ConnectionState             string   `json:"connection_state,omitempty"`
	CreationTime                int      `json:"creation_time,omitempty"`
	Deactivated                 bool     `json:"deactivated,omitempty"`
	DeactivatedAt               int      `json:"deactivated_at,omitempty"`
	DomainNames                 []string `json:"domain_names,omitempty"`
	Enabled                     bool     `json:"enabled,omitempty"`
	EndpointService             string   `json:"endpoint_service,omitempty"`
	NetworkConnectivityConfigId string   `json:"network_connectivity_config_id,omitempty"`
	ResourceNames               []string `json:"resource_names,omitempty"`
	RuleId                      string   `json:"rule_id,omitempty"`
	UpdatedTime                 int      `json:"updated_time,omitempty"`
	VpcEndpointId               string   `json:"vpc_endpoint_id,omitempty"`
}

type ResourceMwsNetworkConnectivityConfigEgressConfigTargetRulesAzurePrivateEndpointRules struct {
	ConnectionState             string   `json:"connection_state,omitempty"`
	CreationTime                int      `json:"creation_time,omitempty"`
	Deactivated                 bool     `json:"deactivated,omitempty"`
	DeactivatedAt               int      `json:"deactivated_at,omitempty"`
	DomainNames                 []string `json:"domain_names,omitempty"`
	EndpointName                string   `json:"endpoint_name,omitempty"`
	GroupId                     string   `json:"group_id,omitempty"`
	NetworkConnectivityConfigId string   `json:"network_connectivity_config_id,omitempty"`
	ResourceId                  string   `json:"resource_id,omitempty"`
	RuleId                      string   `json:"rule_id,omitempty"`
	UpdatedTime                 int      `json:"updated_time,omitempty"`
}

type ResourceMwsNetworkConnectivityConfigEgressConfigTargetRules struct {
	AwsPrivateEndpointRules   []ResourceMwsNetworkConnectivityConfigEgressConfigTargetRulesAwsPrivateEndpointRules   `json:"aws_private_endpoint_rules,omitempty"`
	AzurePrivateEndpointRules []ResourceMwsNetworkConnectivityConfigEgressConfigTargetRulesAzurePrivateEndpointRules `json:"azure_private_endpoint_rules,omitempty"`
}

type ResourceMwsNetworkConnectivityConfigEgressConfig struct {
	DefaultRules *ResourceMwsNetworkConnectivityConfigEgressConfigDefaultRules `json:"default_rules,omitempty"`
	TargetRules  *ResourceMwsNetworkConnectivityConfigEgressConfigTargetRules  `json:"target_rules,omitempty"`
}

type ResourceMwsNetworkConnectivityConfig struct {
	AccountId                   string                                            `json:"account_id,omitempty"`
	CreationTime                int                                               `json:"creation_time,omitempty"`
	Id                          string                                            `json:"id,omitempty"`
	Name                        string                                            `json:"name"`
	NetworkConnectivityConfigId string                                            `json:"network_connectivity_config_id,omitempty"`
	Region                      string                                            `json:"region"`
	UpdatedTime                 int                                               `json:"updated_time,omitempty"`
	EgressConfig                *ResourceMwsNetworkConnectivityConfigEgressConfig `json:"egress_config,omitempty"`
}
