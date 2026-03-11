// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema


type DataSourceKnowledgeAssistantsKnowledgeAssistantsProviderConfig struct {
    WorkspaceId string `json:"workspace_id"`
}

type DataSourceKnowledgeAssistantsKnowledgeAssistants struct {
    CreateTime string `json:"create_time,omitempty"`
    Creator string `json:"creator,omitempty"`
    Description string `json:"description,omitempty"`
    DisplayName string `json:"display_name,omitempty"`
    EndpointName string `json:"endpoint_name,omitempty"`
    ErrorInfo string `json:"error_info,omitempty"`
    ExperimentId string `json:"experiment_id,omitempty"`
    Id string `json:"id,omitempty"`
    Instructions string `json:"instructions,omitempty"`
    Name string `json:"name"`
    ProviderConfig *DataSourceKnowledgeAssistantsKnowledgeAssistantsProviderConfig `json:"provider_config,omitempty"`
    State string `json:"state,omitempty"`
}

type DataSourceKnowledgeAssistantsProviderConfig struct {
    WorkspaceId string `json:"workspace_id"`
}

type DataSourceKnowledgeAssistants struct {
    KnowledgeAssistants []DataSourceKnowledgeAssistantsKnowledgeAssistants `json:"knowledge_assistants,omitempty"`
    PageSize int `json:"page_size,omitempty"`
    ProviderConfig *DataSourceKnowledgeAssistantsProviderConfig `json:"provider_config,omitempty"`
}
