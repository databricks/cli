// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAiSearchEndpointsEndpointsCustomTags struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type DataSourceAiSearchEndpointsEndpointsEndpointStatus struct {
	Message string `json:"message,omitempty"`
	State   string `json:"state,omitempty"`
}

type DataSourceAiSearchEndpointsEndpointsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiSearchEndpointsEndpointsScalingInfo struct {
	RequestedTargetQps int    `json:"requested_target_qps,omitempty"`
	State              string `json:"state,omitempty"`
}

type DataSourceAiSearchEndpointsEndpointsThroughputInfo struct {
	ChangeRequestMessage                    string `json:"change_request_message,omitempty"`
	ChangeRequestState                      string `json:"change_request_state,omitempty"`
	CurrentConcurrency                      int    `json:"current_concurrency,omitempty"`
	CurrentConcurrencyUtilizationPercentage int    `json:"current_concurrency_utilization_percentage,omitempty"`
	CurrentNumReplicas                      int    `json:"current_num_replicas,omitempty"`
	MaximumConcurrencyAllowed               int    `json:"maximum_concurrency_allowed,omitempty"`
	MinimalConcurrencyAllowed               int    `json:"minimal_concurrency_allowed,omitempty"`
	RequestedConcurrency                    int    `json:"requested_concurrency,omitempty"`
	RequestedNumReplicas                    int    `json:"requested_num_replicas,omitempty"`
}

type DataSourceAiSearchEndpointsEndpoints struct {
	BudgetPolicyId          string                                              `json:"budget_policy_id,omitempty"`
	CreateTime              string                                              `json:"create_time,omitempty"`
	Creator                 string                                              `json:"creator,omitempty"`
	CustomTags              []DataSourceAiSearchEndpointsEndpointsCustomTags    `json:"custom_tags,omitempty"`
	EffectiveBudgetPolicyId string                                              `json:"effective_budget_policy_id,omitempty"`
	EndpointStatus          *DataSourceAiSearchEndpointsEndpointsEndpointStatus `json:"endpoint_status,omitempty"`
	EndpointType            string                                              `json:"endpoint_type,omitempty"`
	Id                      string                                              `json:"id,omitempty"`
	IndexCount              int                                                 `json:"index_count,omitempty"`
	LastUpdatedUser         string                                              `json:"last_updated_user,omitempty"`
	Name                    string                                              `json:"name"`
	ProviderConfig          *DataSourceAiSearchEndpointsEndpointsProviderConfig `json:"provider_config,omitempty"`
	ReplicaCount            int                                                 `json:"replica_count,omitempty"`
	ScalingInfo             *DataSourceAiSearchEndpointsEndpointsScalingInfo    `json:"scaling_info,omitempty"`
	TargetQps               int                                                 `json:"target_qps,omitempty"`
	ThroughputInfo          *DataSourceAiSearchEndpointsEndpointsThroughputInfo `json:"throughput_info,omitempty"`
	UpdateTime              string                                              `json:"update_time,omitempty"`
	UsagePolicyId           string                                              `json:"usage_policy_id,omitempty"`
}

type DataSourceAiSearchEndpointsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiSearchEndpoints struct {
	Endpoints      []DataSourceAiSearchEndpointsEndpoints     `json:"endpoints,omitempty"`
	PageSize       int                                        `json:"page_size,omitempty"`
	Parent         string                                     `json:"parent"`
	ProviderConfig *DataSourceAiSearchEndpointsProviderConfig `json:"provider_config,omitempty"`
}
