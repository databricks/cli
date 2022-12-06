// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceToken struct {
	Comment         string `json:"comment,omitempty"`
	CreationTime    int    `json:"creation_time,omitempty"`
	ExpiryTime      int    `json:"expiry_time,omitempty"`
	Id              string `json:"id,omitempty"`
	LifetimeSeconds int    `json:"lifetime_seconds,omitempty"`
	TokenId         string `json:"token_id,omitempty"`
	TokenValue      string `json:"token_value,omitempty"`
}
