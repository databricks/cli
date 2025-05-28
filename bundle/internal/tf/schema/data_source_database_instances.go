// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDatabaseInstancesDatabaseInstances struct {
	AdminPassword string `json:"admin_password,omitempty"`
	AdminRolename string `json:"admin_rolename,omitempty"`
	Capacity      string `json:"capacity,omitempty"`
	CreationTime  string `json:"creation_time,omitempty"`
	Creator       string `json:"creator,omitempty"`
	Name          string `json:"name"`
	PgVersion     string `json:"pg_version,omitempty"`
	ReadWriteDns  string `json:"read_write_dns,omitempty"`
	State         string `json:"state,omitempty"`
	Stopped       bool   `json:"stopped,omitempty"`
	Uid           string `json:"uid,omitempty"`
}

type DataSourceDatabaseInstances struct {
	DatabaseInstances []DataSourceDatabaseInstancesDatabaseInstances `json:"database_instances,omitempty"`
}
