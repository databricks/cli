// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceServicePrincipalSecret struct {
	CreateTime         string `json:"create_time,omitempty"`
	ExpireTime         string `json:"expire_time,omitempty"`
	Id                 string `json:"id,omitempty"`
	Lifetime           string `json:"lifetime,omitempty"`
	Secret             string `json:"secret,omitempty"`
	SecretHash         string `json:"secret_hash,omitempty"`
	ServicePrincipalId string `json:"service_principal_id"`
	Status             string `json:"status,omitempty"`
	TimeRotating       string `json:"time_rotating,omitempty"`
	UpdateTime         string `json:"update_time,omitempty"`
}
