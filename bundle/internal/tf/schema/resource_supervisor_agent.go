// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSupervisorAgentProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceSupervisorAgent struct {
	CreateTime        string                                 `json:"create_time,omitempty"`
	Creator           string                                 `json:"creator,omitempty"`
	Description       string                                 `json:"description"`
	DisplayName       string                                 `json:"display_name"`
	EndpointName      string                                 `json:"endpoint_name,omitempty"`
	ExperimentId      string                                 `json:"experiment_id,omitempty"`
	Id                string                                 `json:"id,omitempty"`
	Instructions      string                                 `json:"instructions,omitempty"`
	Name              string                                 `json:"name,omitempty"`
	ProviderConfig    *ResourceSupervisorAgentProviderConfig `json:"provider_config,omitempty"`
	SupervisorAgentId string                                 `json:"supervisor_agent_id,omitempty"`
}
