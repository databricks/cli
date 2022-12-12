// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlEndpointChannel struct {
	Name string `json:"name,omitempty"`
}

type ResourceSqlEndpointOdbcParams struct {
	Host     string `json:"host,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Path     string `json:"path"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type ResourceSqlEndpointTagsCustomTags struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResourceSqlEndpointTags struct {
	CustomTags []ResourceSqlEndpointTagsCustomTags `json:"custom_tags,omitempty"`
}

type ResourceSqlEndpoint struct {
	AutoStopMins            int                            `json:"auto_stop_mins,omitempty"`
	ClusterSize             string                         `json:"cluster_size"`
	DataSourceId            string                         `json:"data_source_id,omitempty"`
	EnablePhoton            bool                           `json:"enable_photon,omitempty"`
	EnableServerlessCompute bool                           `json:"enable_serverless_compute,omitempty"`
	Id                      string                         `json:"id,omitempty"`
	InstanceProfileArn      string                         `json:"instance_profile_arn,omitempty"`
	JdbcUrl                 string                         `json:"jdbc_url,omitempty"`
	MaxNumClusters          int                            `json:"max_num_clusters,omitempty"`
	MinNumClusters          int                            `json:"min_num_clusters,omitempty"`
	Name                    string                         `json:"name"`
	NumClusters             int                            `json:"num_clusters,omitempty"`
	SpotInstancePolicy      string                         `json:"spot_instance_policy,omitempty"`
	State                   string                         `json:"state,omitempty"`
	WarehouseType           string                         `json:"warehouse_type,omitempty"`
	Channel                 *ResourceSqlEndpointChannel    `json:"channel,omitempty"`
	OdbcParams              *ResourceSqlEndpointOdbcParams `json:"odbc_params,omitempty"`
	Tags                    *ResourceSqlEndpointTags       `json:"tags,omitempty"`
}
