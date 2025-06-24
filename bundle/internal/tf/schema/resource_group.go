// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceGroup struct {
	AclPrincipalId          string `json:"acl_principal_id,omitempty"`
	AllowClusterCreate      bool   `json:"allow_cluster_create,omitempty"`
	AllowInstancePoolCreate bool   `json:"allow_instance_pool_create,omitempty"`
	DatabricksSqlAccess     bool   `json:"databricks_sql_access,omitempty"`
	DisplayName             string `json:"display_name"`
	ExternalId              string `json:"external_id,omitempty"`
	Force                   bool   `json:"force,omitempty"`
	Id                      string `json:"id,omitempty"`
	Url                     string `json:"url,omitempty"`
	WorkspaceAccess         bool   `json:"workspace_access,omitempty"`
	WorkspaceConsume        bool   `json:"workspace_consume,omitempty"`
}
