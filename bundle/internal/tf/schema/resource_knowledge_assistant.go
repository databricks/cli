// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceKnowledgeAssistantProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceKnowledgeAssistant struct {
	CreateTime     string                                    `json:"create_time,omitempty"`
	Creator        string                                    `json:"creator,omitempty"`
	Description    string                                    `json:"description"`
	DisplayName    string                                    `json:"display_name"`
	EndpointName   string                                    `json:"endpoint_name,omitempty"`
	ErrorInfo      string                                    `json:"error_info,omitempty"`
	ExperimentId   string                                    `json:"experiment_id,omitempty"`
	Id             string                                    `json:"id,omitempty"`
	Instructions   string                                    `json:"instructions,omitempty"`
	Name           string                                    `json:"name,omitempty"`
	ProviderConfig *ResourceKnowledgeAssistantProviderConfig `json:"provider_config,omitempty"`
	State          string                                    `json:"state,omitempty"`
}
