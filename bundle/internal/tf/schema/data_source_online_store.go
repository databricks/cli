// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceOnlineStoreProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceOnlineStore struct {
	Capacity         string                               `json:"capacity,omitempty"`
	CreationTime     string                               `json:"creation_time,omitempty"`
	Creator          string                               `json:"creator,omitempty"`
	Name             string                               `json:"name"`
	ProviderConfig   *DataSourceOnlineStoreProviderConfig `json:"provider_config,omitempty"`
	ReadReplicaCount int                                  `json:"read_replica_count,omitempty"`
	State            string                               `json:"state,omitempty"`
	UsagePolicyId    string                               `json:"usage_policy_id,omitempty"`
}
