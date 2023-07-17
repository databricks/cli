// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceUser struct {
	Active                  bool   `json:"active,omitempty"`
	AllowClusterCreate      bool   `json:"allow_cluster_create,omitempty"`
	AllowInstancePoolCreate bool   `json:"allow_instance_pool_create,omitempty"`
	DatabricksSqlAccess     bool   `json:"databricks_sql_access,omitempty"`
	DisableAsUserDeletion   bool   `json:"disable_as_user_deletion,omitempty"`
	DisplayName             string `json:"display_name,omitempty"`
	ExternalId              string `json:"external_id,omitempty"`
	Force                   bool   `json:"force,omitempty"`
	ForceDeleteHomeDir      bool   `json:"force_delete_home_dir,omitempty"`
	ForceDeleteRepos        bool   `json:"force_delete_repos,omitempty"`
	Home                    string `json:"home,omitempty"`
	Id                      string `json:"id,omitempty"`
	Repos                   string `json:"repos,omitempty"`
	UserName                string `json:"user_name"`
	WorkspaceAccess         bool   `json:"workspace_access,omitempty"`
}
