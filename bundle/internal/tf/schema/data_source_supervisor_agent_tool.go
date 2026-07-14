// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSupervisorAgentToolApp struct {
	Name string `json:"name"`
}

type DataSourceSupervisorAgentToolGenieSpace struct {
	Id string `json:"id"`
}

type DataSourceSupervisorAgentToolKnowledgeAssistant struct {
	KnowledgeAssistantId string `json:"knowledge_assistant_id"`
	ServingEndpointName  string `json:"serving_endpoint_name,omitempty"`
}

type DataSourceSupervisorAgentToolProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceSupervisorAgentToolUcConnection struct {
	Name string `json:"name"`
}

type DataSourceSupervisorAgentToolUcFunction struct {
	Name string `json:"name"`
}

type DataSourceSupervisorAgentToolVolume struct {
	Name string `json:"name"`
}

type DataSourceSupervisorAgentTool struct {
	App                *DataSourceSupervisorAgentToolApp                `json:"app,omitempty"`
	Description        string                                           `json:"description,omitempty"`
	GenieSpace         *DataSourceSupervisorAgentToolGenieSpace         `json:"genie_space,omitempty"`
	Id                 string                                           `json:"id,omitempty"`
	KnowledgeAssistant *DataSourceSupervisorAgentToolKnowledgeAssistant `json:"knowledge_assistant,omitempty"`
	Name               string                                           `json:"name"`
	ProviderConfig     *DataSourceSupervisorAgentToolProviderConfig     `json:"provider_config,omitempty"`
	ToolId             string                                           `json:"tool_id,omitempty"`
	ToolType           string                                           `json:"tool_type,omitempty"`
	UcConnection       *DataSourceSupervisorAgentToolUcConnection       `json:"uc_connection,omitempty"`
	UcFunction         *DataSourceSupervisorAgentToolUcFunction         `json:"uc_function,omitempty"`
	Volume             *DataSourceSupervisorAgentToolVolume             `json:"volume,omitempty"`
}
