// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type ResourceSecretAcl struct {
	Id         string `json:"id,omitempty"`
	Permission string `json:"permission"`
	Principal  string `json:"principal"`
	Scope      string `json:"scope"`
}
