// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDatabaseInstance struct {
	Capacity     string `json:"capacity,omitempty"`
	CreationTime string `json:"creation_time,omitempty"`
	Creator      string `json:"creator,omitempty"`
	Name         string `json:"name"`
	PgVersion    string `json:"pg_version,omitempty"`
	ReadWriteDns string `json:"read_write_dns,omitempty"`
	State        string `json:"state,omitempty"`
	Stopped      bool   `json:"stopped,omitempty"`
	Uid          string `json:"uid,omitempty"`
}
