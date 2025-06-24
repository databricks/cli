// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceGroup struct {
	AclPrincipalId          string   `json:"acl_principal_id,omitempty"`
	AllowClusterCreate      bool     `json:"allow_cluster_create,omitempty"`
	AllowInstancePoolCreate bool     `json:"allow_instance_pool_create,omitempty"`
	ChildGroups             []string `json:"child_groups,omitempty"`
	DatabricksSqlAccess     bool     `json:"databricks_sql_access,omitempty"`
	DisplayName             string   `json:"display_name"`
	ExternalId              string   `json:"external_id,omitempty"`
	Groups                  []string `json:"groups,omitempty"`
	Id                      string   `json:"id,omitempty"`
	InstanceProfiles        []string `json:"instance_profiles,omitempty"`
	Members                 []string `json:"members,omitempty"`
	Recursive               bool     `json:"recursive,omitempty"`
	ServicePrincipals       []string `json:"service_principals,omitempty"`
	Users                   []string `json:"users,omitempty"`
	WorkspaceAccess         bool     `json:"workspace_access,omitempty"`
	WorkspaceConsume        bool     `json:"workspace_consume,omitempty"`
}
