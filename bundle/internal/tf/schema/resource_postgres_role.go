// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePostgresRoleProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourcePostgresRoleSpecAttributes struct {
	Bypassrls  bool `json:"bypassrls,omitempty"`
	Createdb   bool `json:"createdb,omitempty"`
	Createrole bool `json:"createrole,omitempty"`
}

type ResourcePostgresRoleSpec struct {
	Attributes      *ResourcePostgresRoleSpecAttributes `json:"attributes,omitempty"`
	AuthMethod      string                              `json:"auth_method,omitempty"`
	IdentityType    string                              `json:"identity_type,omitempty"`
	MembershipRoles []string                            `json:"membership_roles,omitempty"`
	PostgresRole    string                              `json:"postgres_role,omitempty"`
}

type ResourcePostgresRoleStatusAttributes struct {
	Bypassrls  bool `json:"bypassrls,omitempty"`
	Createdb   bool `json:"createdb,omitempty"`
	Createrole bool `json:"createrole,omitempty"`
}

type ResourcePostgresRoleStatus struct {
	Attributes      *ResourcePostgresRoleStatusAttributes `json:"attributes,omitempty"`
	AuthMethod      string                                `json:"auth_method,omitempty"`
	IdentityType    string                                `json:"identity_type,omitempty"`
	MembershipRoles []string                              `json:"membership_roles,omitempty"`
	PostgresRole    string                                `json:"postgres_role,omitempty"`
}

type ResourcePostgresRole struct {
	CreateTime     string                              `json:"create_time,omitempty"`
	Name           string                              `json:"name,omitempty"`
	Parent         string                              `json:"parent"`
	ProviderConfig *ResourcePostgresRoleProviderConfig `json:"provider_config,omitempty"`
	RoleId         string                              `json:"role_id,omitempty"`
	Spec           *ResourcePostgresRoleSpec           `json:"spec,omitempty"`
	Status         *ResourcePostgresRoleStatus         `json:"status,omitempty"`
	UpdateTime     string                              `json:"update_time,omitempty"`
}
