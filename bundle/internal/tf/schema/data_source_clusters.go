// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceClustersFilterBy struct {
	ClusterSources []string `json:"cluster_sources,omitempty"`
	ClusterStates  []string `json:"cluster_states,omitempty"`
	IsPinned       bool     `json:"is_pinned,omitempty"`
	PolicyId       string   `json:"policy_id,omitempty"`
}

type DataSourceClusters struct {
	ClusterNameContains string                      `json:"cluster_name_contains,omitempty"`
	Id                  string                      `json:"id,omitempty"`
	Ids                 []string                    `json:"ids,omitempty"`
	FilterBy            *DataSourceClustersFilterBy `json:"filter_by,omitempty"`
}
