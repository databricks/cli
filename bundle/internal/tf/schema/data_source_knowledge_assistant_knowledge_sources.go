// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceKnowledgeAssistantKnowledgeSourcesKnowledgeSourcesFileTable struct {
	FileCol   string `json:"file_col"`
	TableName string `json:"table_name"`
}

type DataSourceKnowledgeAssistantKnowledgeSourcesKnowledgeSourcesFiles struct {
	Path string `json:"path"`
}

type DataSourceKnowledgeAssistantKnowledgeSourcesKnowledgeSourcesIndex struct {
	DocUriCol string `json:"doc_uri_col"`
	IndexName string `json:"index_name"`
	TextCol   string `json:"text_col"`
}

type DataSourceKnowledgeAssistantKnowledgeSourcesKnowledgeSourcesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceKnowledgeAssistantKnowledgeSourcesKnowledgeSources struct {
	CreateTime          string                                                                      `json:"create_time,omitempty"`
	Description         string                                                                      `json:"description,omitempty"`
	DisplayName         string                                                                      `json:"display_name,omitempty"`
	FileTable           *DataSourceKnowledgeAssistantKnowledgeSourcesKnowledgeSourcesFileTable      `json:"file_table,omitempty"`
	Files               *DataSourceKnowledgeAssistantKnowledgeSourcesKnowledgeSourcesFiles          `json:"files,omitempty"`
	Id                  string                                                                      `json:"id,omitempty"`
	Index               *DataSourceKnowledgeAssistantKnowledgeSourcesKnowledgeSourcesIndex          `json:"index,omitempty"`
	KnowledgeCutoffTime string                                                                      `json:"knowledge_cutoff_time,omitempty"`
	Name                string                                                                      `json:"name"`
	ProviderConfig      *DataSourceKnowledgeAssistantKnowledgeSourcesKnowledgeSourcesProviderConfig `json:"provider_config,omitempty"`
	SourceType          string                                                                      `json:"source_type,omitempty"`
	State               string                                                                      `json:"state,omitempty"`
}

type DataSourceKnowledgeAssistantKnowledgeSourcesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceKnowledgeAssistantKnowledgeSources struct {
	KnowledgeSources []DataSourceKnowledgeAssistantKnowledgeSourcesKnowledgeSources `json:"knowledge_sources,omitempty"`
	PageSize         int                                                            `json:"page_size,omitempty"`
	Parent           string                                                         `json:"parent"`
	ProviderConfig   *DataSourceKnowledgeAssistantKnowledgeSourcesProviderConfig    `json:"provider_config,omitempty"`
}
