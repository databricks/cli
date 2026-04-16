// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresRoleProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourcePostgresRoleSpecAttributes struct {
	Bypassrls  bool `json:"bypassrls,omitempty"`
	Createdb   bool `json:"createdb,omitempty"`
	Createrole bool `json:"createrole,omitempty"`
}

type DataSourcePostgresRoleSpec struct {
	Attributes      *DataSourcePostgresRoleSpecAttributes `json:"attributes,omitempty"`
	AuthMethod      string                                `json:"auth_method,omitempty"`
	IdentityType    string                                `json:"identity_type,omitempty"`
	MembershipRoles []string                              `json:"membership_roles,omitempty"`
	PostgresRole    string                                `json:"postgres_role,omitempty"`
}

type DataSourcePostgresRoleStatusAttributes struct {
	Bypassrls  bool `json:"bypassrls,omitempty"`
	Createdb   bool `json:"createdb,omitempty"`
	Createrole bool `json:"createrole,omitempty"`
}

type DataSourcePostgresRoleStatus struct {
	Attributes      *DataSourcePostgresRoleStatusAttributes `json:"attributes,omitempty"`
	AuthMethod      string                                  `json:"auth_method,omitempty"`
	IdentityType    string                                  `json:"identity_type,omitempty"`
	MembershipRoles []string                                `json:"membership_roles,omitempty"`
	PostgresRole    string                                  `json:"postgres_role,omitempty"`
}

type DataSourcePostgresRole struct {
	CreateTime     string                                `json:"create_time,omitempty"`
	Name           string                                `json:"name"`
	Parent         string                                `json:"parent,omitempty"`
	ProviderConfig *DataSourcePostgresRoleProviderConfig `json:"provider_config,omitempty"`
	Spec           *DataSourcePostgresRoleSpec           `json:"spec,omitempty"`
	Status         *DataSourcePostgresRoleStatus         `json:"status,omitempty"`
	UpdateTime     string                                `json:"update_time,omitempty"`
}
