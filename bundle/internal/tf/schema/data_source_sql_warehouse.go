// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSqlWarehouseChannel struct {
	DbsqlVersion string `json:"dbsql_version,omitempty"`
	Name         string `json:"name,omitempty"`
}

type DataSourceSqlWarehouseHealthFailureReason struct {
	Code       string            `json:"code,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Type       string            `json:"type,omitempty"`
}

type DataSourceSqlWarehouseHealth struct {
	Details       string                                     `json:"details,omitempty"`
	Message       string                                     `json:"message,omitempty"`
	Status        string                                     `json:"status,omitempty"`
	Summary       string                                     `json:"summary,omitempty"`
	FailureReason *DataSourceSqlWarehouseHealthFailureReason `json:"failure_reason,omitempty"`
}

type DataSourceSqlWarehouseOdbcParams struct {
	Hostname string `json:"hostname,omitempty"`
	Path     string `json:"path,omitempty"`
	Port     int    `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

type DataSourceSqlWarehouseTagsCustomTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type DataSourceSqlWarehouseTags struct {
	CustomTags []DataSourceSqlWarehouseTagsCustomTags `json:"custom_tags,omitempty"`
}

type DataSourceSqlWarehouse struct {
	AutoStopMins            int                               `json:"auto_stop_mins,omitempty"`
	ClusterSize             string                            `json:"cluster_size,omitempty"`
	CreatorName             string                            `json:"creator_name,omitempty"`
	DataSourceId            string                            `json:"data_source_id,omitempty"`
	EnablePhoton            bool                              `json:"enable_photon,omitempty"`
	EnableServerlessCompute bool                              `json:"enable_serverless_compute,omitempty"`
	Id                      string                            `json:"id,omitempty"`
	InstanceProfileArn      string                            `json:"instance_profile_arn,omitempty"`
	JdbcUrl                 string                            `json:"jdbc_url,omitempty"`
	MaxNumClusters          int                               `json:"max_num_clusters,omitempty"`
	MinNumClusters          int                               `json:"min_num_clusters,omitempty"`
	Name                    string                            `json:"name,omitempty"`
	NumActiveSessions       int                               `json:"num_active_sessions,omitempty"`
	NumClusters             int                               `json:"num_clusters,omitempty"`
	SpotInstancePolicy      string                            `json:"spot_instance_policy,omitempty"`
	State                   string                            `json:"state,omitempty"`
	WarehouseType           string                            `json:"warehouse_type,omitempty"`
	Channel                 *DataSourceSqlWarehouseChannel    `json:"channel,omitempty"`
	Health                  *DataSourceSqlWarehouseHealth     `json:"health,omitempty"`
	OdbcParams              *DataSourceSqlWarehouseOdbcParams `json:"odbc_params,omitempty"`
	Tags                    *DataSourceSqlWarehouseTags       `json:"tags,omitempty"`
}
