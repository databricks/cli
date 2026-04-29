// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSupervisorAgentToolApp struct {
	Name string `json:"name"`
}

type ResourceSupervisorAgentToolGenieSpace struct {
	Id string `json:"id"`
}

type ResourceSupervisorAgentToolKnowledgeAssistant struct {
	KnowledgeAssistantId string `json:"knowledge_assistant_id"`
	ServingEndpointName  string `json:"serving_endpoint_name,omitempty"`
}

type ResourceSupervisorAgentToolProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceSupervisorAgentToolUcConnection struct {
	Name string `json:"name"`
}

type ResourceSupervisorAgentToolUcFunction struct {
	Name string `json:"name"`
}

type ResourceSupervisorAgentToolVolume struct {
	Name string `json:"name"`
}

type ResourceSupervisorAgentTool struct {
	App                *ResourceSupervisorAgentToolApp                `json:"app,omitempty"`
	Description        string                                         `json:"description"`
	GenieSpace         *ResourceSupervisorAgentToolGenieSpace         `json:"genie_space,omitempty"`
	Id                 string                                         `json:"id,omitempty"`
	KnowledgeAssistant *ResourceSupervisorAgentToolKnowledgeAssistant `json:"knowledge_assistant,omitempty"`
	Name               string                                         `json:"name,omitempty"`
	Parent             string                                         `json:"parent"`
	ProviderConfig     *ResourceSupervisorAgentToolProviderConfig     `json:"provider_config,omitempty"`
	ToolId             string                                         `json:"tool_id"`
	ToolType           string                                         `json:"tool_type"`
	UcConnection       *ResourceSupervisorAgentToolUcConnection       `json:"uc_connection,omitempty"`
	UcFunction         *ResourceSupervisorAgentToolUcFunction         `json:"uc_function,omitempty"`
	Volume             *ResourceSupervisorAgentToolVolume             `json:"volume,omitempty"`
}
