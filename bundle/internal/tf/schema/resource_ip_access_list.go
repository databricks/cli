// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceIpAccessList struct {
	Enabled     bool     `json:"enabled,omitempty"`
	Id          string   `json:"id,omitempty"`
	IpAddresses []string `json:"ip_addresses"`
	Label       string   `json:"label"`
	ListType    string   `json:"list_type"`
}
