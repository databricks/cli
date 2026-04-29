// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSupervisorAgentProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceSupervisorAgent struct {
	CreateTime        string                                   `json:"create_time,omitempty"`
	Creator           string                                   `json:"creator,omitempty"`
	Description       string                                   `json:"description,omitempty"`
	DisplayName       string                                   `json:"display_name,omitempty"`
	EndpointName      string                                   `json:"endpoint_name,omitempty"`
	ExperimentId      string                                   `json:"experiment_id,omitempty"`
	Id                string                                   `json:"id,omitempty"`
	Instructions      string                                   `json:"instructions,omitempty"`
	Name              string                                   `json:"name"`
	ProviderConfig    *DataSourceSupervisorAgentProviderConfig `json:"provider_config,omitempty"`
	SupervisorAgentId string                                   `json:"supervisor_agent_id,omitempty"`
}
