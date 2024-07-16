// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlEndpointChannel struct {
	DbsqlVersion string `json:"dbsql_version,omitempty"`
	Name         string `json:"name,omitempty"`
}

type ResourceSqlEndpointTagsCustomTags struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResourceSqlEndpointTags struct {
	CustomTags []ResourceSqlEndpointTagsCustomTags `json:"custom_tags,omitempty"`
}

type ResourceSqlEndpoint struct {
	AutoStopMins            int                         `json:"auto_stop_mins,omitempty"`
	ClusterSize             string                      `json:"cluster_size"`
	CreatorName             string                      `json:"creator_name,omitempty"`
	DataSourceId            string                      `json:"data_source_id,omitempty"`
	EnablePhoton            bool                        `json:"enable_photon,omitempty"`
	EnableServerlessCompute bool                        `json:"enable_serverless_compute,omitempty"`
	Health                  []any                       `json:"health,omitempty"`
	Id                      string                      `json:"id,omitempty"`
	InstanceProfileArn      string                      `json:"instance_profile_arn,omitempty"`
	JdbcUrl                 string                      `json:"jdbc_url,omitempty"`
	MaxNumClusters          int                         `json:"max_num_clusters,omitempty"`
	MinNumClusters          int                         `json:"min_num_clusters,omitempty"`
	Name                    string                      `json:"name"`
	NumActiveSessions       int                         `json:"num_active_sessions,omitempty"`
	NumClusters             int                         `json:"num_clusters,omitempty"`
	OdbcParams              []any                       `json:"odbc_params,omitempty"`
	SpotInstancePolicy      string                      `json:"spot_instance_policy,omitempty"`
	State                   string                      `json:"state,omitempty"`
	WarehouseType           string                      `json:"warehouse_type,omitempty"`
	Channel                 *ResourceSqlEndpointChannel `json:"channel,omitempty"`
	Tags                    *ResourceSqlEndpointTags    `json:"tags,omitempty"`
}
