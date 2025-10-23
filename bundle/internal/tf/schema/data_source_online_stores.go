// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceOnlineStoresOnlineStores struct {
	Capacity         string `json:"capacity"`
	CreationTime     string `json:"creation_time,omitempty"`
	Creator          string `json:"creator,omitempty"`
	Name             string `json:"name"`
	ReadReplicaCount int    `json:"read_replica_count,omitempty"`
	State            string `json:"state,omitempty"`
}

type DataSourceOnlineStores struct {
	OnlineStores []DataSourceOnlineStoresOnlineStores `json:"online_stores,omitempty"`
}
