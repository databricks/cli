// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceOnlineStore struct {
	Capacity         string `json:"capacity"`
	CreationTime     string `json:"creation_time,omitempty"`
	Creator          string `json:"creator,omitempty"`
	Name             string `json:"name"`
	ReadReplicaCount int    `json:"read_replica_count,omitempty"`
	State            string `json:"state,omitempty"`
}
