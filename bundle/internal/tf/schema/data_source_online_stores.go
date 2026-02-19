// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceOnlineStoresOnlineStoresProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceOnlineStoresOnlineStores struct {
	Capacity         string                                            `json:"capacity,omitempty"`
	CreationTime     string                                            `json:"creation_time,omitempty"`
	Creator          string                                            `json:"creator,omitempty"`
	Name             string                                            `json:"name"`
	ProviderConfig   *DataSourceOnlineStoresOnlineStoresProviderConfig `json:"provider_config,omitempty"`
	ReadReplicaCount int                                               `json:"read_replica_count,omitempty"`
	State            string                                            `json:"state,omitempty"`
	UsagePolicyId    string                                            `json:"usage_policy_id,omitempty"`
}

type DataSourceOnlineStoresProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceOnlineStores struct {
	OnlineStores   []DataSourceOnlineStoresOnlineStores  `json:"online_stores,omitempty"`
	PageSize       int                                   `json:"page_size,omitempty"`
	ProviderConfig *DataSourceOnlineStoresProviderConfig `json:"provider_config,omitempty"`
}
