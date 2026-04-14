// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceKnowledgeAssistantKnowledgeSourceFileTable struct {
	FileCol   string `json:"file_col"`
	TableName string `json:"table_name"`
}

type ResourceKnowledgeAssistantKnowledgeSourceFiles struct {
	Path string `json:"path"`
}

type ResourceKnowledgeAssistantKnowledgeSourceIndex struct {
	DocUriCol string `json:"doc_uri_col"`
	IndexName string `json:"index_name"`
	TextCol   string `json:"text_col"`
}

type ResourceKnowledgeAssistantKnowledgeSourceProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceKnowledgeAssistantKnowledgeSource struct {
	CreateTime          string                                                   `json:"create_time,omitempty"`
	Description         string                                                   `json:"description"`
	DisplayName         string                                                   `json:"display_name"`
	FileTable           *ResourceKnowledgeAssistantKnowledgeSourceFileTable      `json:"file_table,omitempty"`
	Files               *ResourceKnowledgeAssistantKnowledgeSourceFiles          `json:"files,omitempty"`
	Id                  string                                                   `json:"id,omitempty"`
	Index               *ResourceKnowledgeAssistantKnowledgeSourceIndex          `json:"index,omitempty"`
	KnowledgeCutoffTime string                                                   `json:"knowledge_cutoff_time,omitempty"`
	Name                string                                                   `json:"name,omitempty"`
	Parent              string                                                   `json:"parent"`
	ProviderConfig      *ResourceKnowledgeAssistantKnowledgeSourceProviderConfig `json:"provider_config,omitempty"`
	SourceType          string                                                   `json:"source_type"`
	State               string                                                   `json:"state,omitempty"`
}
