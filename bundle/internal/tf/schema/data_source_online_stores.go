// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceOnlineStoresOnlineStores struct {
	Capacity     string `json:"capacity,omitempty"`
	CreationTime string `json:"creation_time,omitempty"`
	Creator      string `json:"creator,omitempty"`
	Name         string `json:"name"`
	State        string `json:"state,omitempty"`
}

type DataSourceOnlineStores struct {
	OnlineStores []DataSourceOnlineStoresOnlineStores `json:"online_stores,omitempty"`
}
