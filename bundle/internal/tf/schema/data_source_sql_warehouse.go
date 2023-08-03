// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSqlWarehouseChannel struct {
	Name string `json:"name,omitempty"`
}

type DataSourceSqlWarehouseOdbcParams struct {
	Host     string `json:"host,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Path     string `json:"path"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type DataSourceSqlWarehouseTagsCustomTags struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type DataSourceSqlWarehouseTags struct {
	CustomTags []DataSourceSqlWarehouseTagsCustomTags `json:"custom_tags,omitempty"`
}

type DataSourceSqlWarehouse struct {
	AutoStopMins            int                               `json:"auto_stop_mins,omitempty"`
	ClusterSize             string                            `json:"cluster_size,omitempty"`
	DataSourceId            string                            `json:"data_source_id,omitempty"`
	EnablePhoton            bool                              `json:"enable_photon,omitempty"`
	EnableServerlessCompute bool                              `json:"enable_serverless_compute,omitempty"`
	Id                      string                            `json:"id,omitempty"`
	InstanceProfileArn      string                            `json:"instance_profile_arn,omitempty"`
	JdbcUrl                 string                            `json:"jdbc_url,omitempty"`
	MaxNumClusters          int                               `json:"max_num_clusters,omitempty"`
	MinNumClusters          int                               `json:"min_num_clusters,omitempty"`
	Name                    string                            `json:"name,omitempty"`
	NumClusters             int                               `json:"num_clusters,omitempty"`
	SpotInstancePolicy      string                            `json:"spot_instance_policy,omitempty"`
	State                   string                            `json:"state,omitempty"`
	Channel                 *DataSourceSqlWarehouseChannel    `json:"channel,omitempty"`
	OdbcParams              *DataSourceSqlWarehouseOdbcParams `json:"odbc_params,omitempty"`
	Tags                    *DataSourceSqlWarehouseTags       `json:"tags,omitempty"`
}
