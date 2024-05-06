// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMwsNccPrivateEndpointRule struct {
	ConnectionState             string `json:"connection_state,omitempty"`
	CreationTime                int    `json:"creation_time,omitempty"`
	Deactivated                 bool   `json:"deactivated,omitempty"`
	DeactivatedAt               int    `json:"deactivated_at,omitempty"`
	EndpointName                string `json:"endpoint_name,omitempty"`
	GroupId                     string `json:"group_id"`
	Id                          string `json:"id,omitempty"`
	NetworkConnectivityConfigId string `json:"network_connectivity_config_id"`
	ResourceId                  string `json:"resource_id"`
	RuleId                      string `json:"rule_id,omitempty"`
	UpdatedTime                 int    `json:"updated_time,omitempty"`
}
