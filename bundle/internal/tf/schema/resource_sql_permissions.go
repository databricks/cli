// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlPermissionsPrivilegeAssignments struct {
	Principal  string   `json:"principal"`
	Privileges []string `json:"privileges"`
}

type ResourceSqlPermissions struct {
	AnonymousFunction    bool                                         `json:"anonymous_function,omitempty"`
	AnyFile              bool                                         `json:"any_file,omitempty"`
	Catalog              bool                                         `json:"catalog,omitempty"`
	ClusterId            string                                       `json:"cluster_id,omitempty"`
	Database             string                                       `json:"database,omitempty"`
	Id                   string                                       `json:"id,omitempty"`
	Table                string                                       `json:"table,omitempty"`
	View                 string                                       `json:"view,omitempty"`
	PrivilegeAssignments []ResourceSqlPermissionsPrivilegeAssignments `json:"privilege_assignments,omitempty"`
}
