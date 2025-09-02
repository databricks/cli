// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMwsNetworkConnectivityConfigEgressConfigDefaultRulesAwsStableIpRule struct {
	CidrBlocks []string `json:"cidr_blocks,omitempty"`
}

type DataSourceMwsNetworkConnectivityConfigEgressConfigDefaultRulesAzureServiceEndpointRule struct {
	Subnets        []string `json:"subnets,omitempty"`
	TargetRegion   string   `json:"target_region,omitempty"`
	TargetServices []string `json:"target_services,omitempty"`
}

type DataSourceMwsNetworkConnectivityConfigEgressConfigDefaultRules struct {
	AwsStableIpRule          *DataSourceMwsNetworkConnectivityConfigEgressConfigDefaultRulesAwsStableIpRule          `json:"aws_stable_ip_rule,omitempty"`
	AzureServiceEndpointRule *DataSourceMwsNetworkConnectivityConfigEgressConfigDefaultRulesAzureServiceEndpointRule `json:"azure_service_endpoint_rule,omitempty"`
}

type DataSourceMwsNetworkConnectivityConfigEgressConfigTargetRulesAwsPrivateEndpointRules struct {
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

type DataSourceMwsNetworkConnectivityConfigEgressConfigTargetRulesAzurePrivateEndpointRules struct {
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

type DataSourceMwsNetworkConnectivityConfigEgressConfigTargetRules struct {
	AwsPrivateEndpointRules   []DataSourceMwsNetworkConnectivityConfigEgressConfigTargetRulesAwsPrivateEndpointRules   `json:"aws_private_endpoint_rules,omitempty"`
	AzurePrivateEndpointRules []DataSourceMwsNetworkConnectivityConfigEgressConfigTargetRulesAzurePrivateEndpointRules `json:"azure_private_endpoint_rules,omitempty"`
}

type DataSourceMwsNetworkConnectivityConfigEgressConfig struct {
	DefaultRules *DataSourceMwsNetworkConnectivityConfigEgressConfigDefaultRules `json:"default_rules,omitempty"`
	TargetRules  *DataSourceMwsNetworkConnectivityConfigEgressConfigTargetRules  `json:"target_rules,omitempty"`
}

type DataSourceMwsNetworkConnectivityConfig struct {
	AccountId                   string                                              `json:"account_id,omitempty"`
	CreationTime                int                                                 `json:"creation_time,omitempty"`
	Id                          string                                              `json:"id,omitempty"`
	Name                        string                                              `json:"name"`
	NetworkConnectivityConfigId string                                              `json:"network_connectivity_config_id,omitempty"`
	Region                      string                                              `json:"region,omitempty"`
	UpdatedTime                 int                                                 `json:"updated_time,omitempty"`
	EgressConfig                *DataSourceMwsNetworkConnectivityConfigEgressConfig `json:"egress_config,omitempty"`
}
