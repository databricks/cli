// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSupervisorAgentToolsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceSupervisorAgentToolsToolsApp struct {
	Name string `json:"name"`
}

type DataSourceSupervisorAgentToolsToolsGenieSpace struct {
	Id string `json:"id"`
}

type DataSourceSupervisorAgentToolsToolsKnowledgeAssistant struct {
	KnowledgeAssistantId string `json:"knowledge_assistant_id"`
	ServingEndpointName  string `json:"serving_endpoint_name,omitempty"`
}

type DataSourceSupervisorAgentToolsToolsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceSupervisorAgentToolsToolsUcConnection struct {
	Name string `json:"name"`
}

type DataSourceSupervisorAgentToolsToolsUcFunction struct {
	Name string `json:"name"`
}

type DataSourceSupervisorAgentToolsToolsVolume struct {
	Name string `json:"name"`
}

type DataSourceSupervisorAgentToolsTools struct {
	App                *DataSourceSupervisorAgentToolsToolsApp                `json:"app,omitempty"`
	Description        string                                                 `json:"description,omitempty"`
	GenieSpace         *DataSourceSupervisorAgentToolsToolsGenieSpace         `json:"genie_space,omitempty"`
	Id                 string                                                 `json:"id,omitempty"`
	KnowledgeAssistant *DataSourceSupervisorAgentToolsToolsKnowledgeAssistant `json:"knowledge_assistant,omitempty"`
	Name               string                                                 `json:"name"`
	ProviderConfig     *DataSourceSupervisorAgentToolsToolsProviderConfig     `json:"provider_config,omitempty"`
	ToolId             string                                                 `json:"tool_id,omitempty"`
	ToolType           string                                                 `json:"tool_type,omitempty"`
	UcConnection       *DataSourceSupervisorAgentToolsToolsUcConnection       `json:"uc_connection,omitempty"`
	UcFunction         *DataSourceSupervisorAgentToolsToolsUcFunction         `json:"uc_function,omitempty"`
	Volume             *DataSourceSupervisorAgentToolsToolsVolume             `json:"volume,omitempty"`
}

type DataSourceSupervisorAgentTools struct {
	PageSize       int                                           `json:"page_size,omitempty"`
	Parent         string                                        `json:"parent"`
	ProviderConfig *DataSourceSupervisorAgentToolsProviderConfig `json:"provider_config,omitempty"`
	Tools          []DataSourceSupervisorAgentToolsTools         `json:"tools,omitempty"`
}
