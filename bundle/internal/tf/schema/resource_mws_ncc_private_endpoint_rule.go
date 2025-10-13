// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMwsNccPrivateEndpointRule struct {
	AccountId                   string   `json:"account_id,omitempty"`
	ConnectionState             string   `json:"connection_state,omitempty"`
	CreationTime                int      `json:"creation_time,omitempty"`
	Deactivated                 bool     `json:"deactivated,omitempty"`
	DeactivatedAt               int      `json:"deactivated_at,omitempty"`
	DomainNames                 []string `json:"domain_names,omitempty"`
	Enabled                     bool     `json:"enabled,omitempty"`
	EndpointName                string   `json:"endpoint_name,omitempty"`
	EndpointService             string   `json:"endpoint_service,omitempty"`
	GroupId                     string   `json:"group_id,omitempty"`
	Id                          string   `json:"id,omitempty"`
	NetworkConnectivityConfigId string   `json:"network_connectivity_config_id"`
	ResourceId                  string   `json:"resource_id,omitempty"`
	ResourceNames               []string `json:"resource_names,omitempty"`
	RuleId                      string   `json:"rule_id,omitempty"`
	UpdatedTime                 int      `json:"updated_time,omitempty"`
	VpcEndpointId               string   `json:"vpc_endpoint_id,omitempty"`
}
