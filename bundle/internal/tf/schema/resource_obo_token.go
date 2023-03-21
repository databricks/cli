// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceOboToken struct {
	ApplicationId   string `json:"application_id"`
	Comment         string `json:"comment,omitempty"`
	Id              string `json:"id,omitempty"`
	LifetimeSeconds int    `json:"lifetime_seconds,omitempty"`
	TokenValue      string `json:"token_value,omitempty"`
}
