// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type ResourceServicePrincipal struct {
	Active                  bool   `json:"active,omitempty"`
	AllowClusterCreate      bool   `json:"allow_cluster_create,omitempty"`
	AllowInstancePoolCreate bool   `json:"allow_instance_pool_create,omitempty"`
	ApplicationId           string `json:"application_id,omitempty"`
	DatabricksSqlAccess     bool   `json:"databricks_sql_access,omitempty"`
	DisplayName             string `json:"display_name,omitempty"`
	ExternalId              string `json:"external_id,omitempty"`
	Force                   bool   `json:"force,omitempty"`
	Id                      string `json:"id,omitempty"`
	WorkspaceAccess         bool   `json:"workspace_access,omitempty"`
}
