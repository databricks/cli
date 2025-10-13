// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceEntitlements struct {
	AllowClusterCreate      bool   `json:"allow_cluster_create,omitempty"`
	AllowInstancePoolCreate bool   `json:"allow_instance_pool_create,omitempty"`
	DatabricksSqlAccess     bool   `json:"databricks_sql_access,omitempty"`
	GroupId                 string `json:"group_id,omitempty"`
	Id                      string `json:"id,omitempty"`
	ServicePrincipalId      string `json:"service_principal_id,omitempty"`
	UserId                  string `json:"user_id,omitempty"`
	WorkspaceAccess         bool   `json:"workspace_access,omitempty"`
	WorkspaceConsume        bool   `json:"workspace_consume,omitempty"`
}
