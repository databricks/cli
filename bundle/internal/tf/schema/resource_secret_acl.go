// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSecretAcl struct {
	Id         string `json:"id,omitempty"`
	Permission string `json:"permission"`
	Principal  string `json:"principal"`
	Scope      string `json:"scope"`
}
