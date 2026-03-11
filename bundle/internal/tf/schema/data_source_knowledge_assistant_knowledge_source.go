// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema


type DataSourceKnowledgeAssistantKnowledgeSourceFileTable struct {
    FileCol string `json:"file_col"`
    TableName string `json:"table_name"`
}

type DataSourceKnowledgeAssistantKnowledgeSourceFiles struct {
    Path string `json:"path"`
}

type DataSourceKnowledgeAssistantKnowledgeSourceIndex struct {
    DocUriCol string `json:"doc_uri_col"`
    IndexName string `json:"index_name"`
    TextCol string `json:"text_col"`
}

type DataSourceKnowledgeAssistantKnowledgeSourceProviderConfig struct {
    WorkspaceId string `json:"workspace_id"`
}

type DataSourceKnowledgeAssistantKnowledgeSource struct {
    CreateTime string `json:"create_time,omitempty"`
    Description string `json:"description,omitempty"`
    DisplayName string `json:"display_name,omitempty"`
    FileTable *DataSourceKnowledgeAssistantKnowledgeSourceFileTable `json:"file_table,omitempty"`
    Files *DataSourceKnowledgeAssistantKnowledgeSourceFiles `json:"files,omitempty"`
    Id string `json:"id,omitempty"`
    Index *DataSourceKnowledgeAssistantKnowledgeSourceIndex `json:"index,omitempty"`
    KnowledgeCutoffTime string `json:"knowledge_cutoff_time,omitempty"`
    Name string `json:"name"`
    ProviderConfig *DataSourceKnowledgeAssistantKnowledgeSourceProviderConfig `json:"provider_config,omitempty"`
    SourceType string `json:"source_type,omitempty"`
    State string `json:"state,omitempty"`
}
