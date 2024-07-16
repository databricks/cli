// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMwsPrivateAccessSettings struct {
	AccountId                 string   `json:"account_id,omitempty"`
	AllowedVpcEndpointIds     []string `json:"allowed_vpc_endpoint_ids,omitempty"`
	Id                        string   `json:"id,omitempty"`
	PrivateAccessLevel        string   `json:"private_access_level,omitempty"`
	PrivateAccessSettingsId   string   `json:"private_access_settings_id,omitempty"`
	PrivateAccessSettingsName string   `json:"private_access_settings_name"`
	PublicAccessEnabled       bool     `json:"public_access_enabled,omitempty"`
	Region                    string   `json:"region"`
}
