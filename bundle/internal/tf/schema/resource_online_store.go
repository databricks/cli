// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceOnlineStoreProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceOnlineStore struct {
	Capacity         string                             `json:"capacity"`
	CreationTime     string                             `json:"creation_time,omitempty"`
	Creator          string                             `json:"creator,omitempty"`
	Name             string                             `json:"name"`
	ProviderConfig   *ResourceOnlineStoreProviderConfig `json:"provider_config,omitempty"`
	ReadReplicaCount int                                `json:"read_replica_count,omitempty"`
	State            string                             `json:"state,omitempty"`
	UsagePolicyId    string                             `json:"usage_policy_id,omitempty"`
}
