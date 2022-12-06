// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePermissionAssignment struct {
	Id          string   `json:"id,omitempty"`
	Permissions []string `json:"permissions"`
	PrincipalId int      `json:"principal_id"`
}
