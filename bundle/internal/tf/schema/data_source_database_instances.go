// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDatabaseInstancesDatabaseInstancesChildInstanceRefs struct {
	BranchTime   string `json:"branch_time,omitempty"`
	EffectiveLsn string `json:"effective_lsn,omitempty"`
	Lsn          string `json:"lsn,omitempty"`
	Name         string `json:"name,omitempty"`
	Uid          string `json:"uid,omitempty"`
}

type DataSourceDatabaseInstancesDatabaseInstancesParentInstanceRef struct {
	BranchTime   string `json:"branch_time,omitempty"`
	EffectiveLsn string `json:"effective_lsn,omitempty"`
	Lsn          string `json:"lsn,omitempty"`
	Name         string `json:"name,omitempty"`
	Uid          string `json:"uid,omitempty"`
}

type DataSourceDatabaseInstancesDatabaseInstances struct {
	Capacity                           string                                                          `json:"capacity,omitempty"`
	ChildInstanceRefs                  []DataSourceDatabaseInstancesDatabaseInstancesChildInstanceRefs `json:"child_instance_refs,omitempty"`
	CreationTime                       string                                                          `json:"creation_time,omitempty"`
	Creator                            string                                                          `json:"creator,omitempty"`
	EffectiveEnableReadableSecondaries bool                                                            `json:"effective_enable_readable_secondaries,omitempty"`
	EffectiveNodeCount                 int                                                             `json:"effective_node_count,omitempty"`
	EffectiveRetentionWindowInDays     int                                                             `json:"effective_retention_window_in_days,omitempty"`
	EffectiveStopped                   bool                                                            `json:"effective_stopped,omitempty"`
	EnableReadableSecondaries          bool                                                            `json:"enable_readable_secondaries,omitempty"`
	Name                               string                                                          `json:"name"`
	NodeCount                          int                                                             `json:"node_count,omitempty"`
	ParentInstanceRef                  *DataSourceDatabaseInstancesDatabaseInstancesParentInstanceRef  `json:"parent_instance_ref,omitempty"`
	PgVersion                          string                                                          `json:"pg_version,omitempty"`
	ReadOnlyDns                        string                                                          `json:"read_only_dns,omitempty"`
	ReadWriteDns                       string                                                          `json:"read_write_dns,omitempty"`
	RetentionWindowInDays              int                                                             `json:"retention_window_in_days,omitempty"`
	State                              string                                                          `json:"state,omitempty"`
	Stopped                            bool                                                            `json:"stopped,omitempty"`
	Uid                                string                                                          `json:"uid,omitempty"`
}

type DataSourceDatabaseInstances struct {
	DatabaseInstances []DataSourceDatabaseInstancesDatabaseInstances `json:"database_instances,omitempty"`
}
