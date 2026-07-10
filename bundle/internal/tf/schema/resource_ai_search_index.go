// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAiSearchIndexDeltaSyncIndexSpecEmbeddingSourceColumns struct {
	EmbeddingModelEndpoint    string `json:"embedding_model_endpoint,omitempty"`
	ModelEndpointNameForQuery string `json:"model_endpoint_name_for_query,omitempty"`
	Name                      string `json:"name,omitempty"`
}

type ResourceAiSearchIndexDeltaSyncIndexSpecEmbeddingVectorColumns struct {
	EmbeddingDimension int    `json:"embedding_dimension,omitempty"`
	Name               string `json:"name,omitempty"`
}

type ResourceAiSearchIndexDeltaSyncIndexSpec struct {
	ColumnsToSync           []string                                                        `json:"columns_to_sync,omitempty"`
	EmbeddingSourceColumns  []ResourceAiSearchIndexDeltaSyncIndexSpecEmbeddingSourceColumns `json:"embedding_source_columns,omitempty"`
	EmbeddingVectorColumns  []ResourceAiSearchIndexDeltaSyncIndexSpecEmbeddingVectorColumns `json:"embedding_vector_columns,omitempty"`
	EmbeddingWritebackTable string                                                          `json:"embedding_writeback_table,omitempty"`
	PipelineId              string                                                          `json:"pipeline_id,omitempty"`
	PipelineType            string                                                          `json:"pipeline_type"`
	SourceTable             string                                                          `json:"source_table,omitempty"`
}

type ResourceAiSearchIndexDirectAccessIndexSpecEmbeddingSourceColumns struct {
	EmbeddingModelEndpoint    string `json:"embedding_model_endpoint,omitempty"`
	ModelEndpointNameForQuery string `json:"model_endpoint_name_for_query,omitempty"`
	Name                      string `json:"name,omitempty"`
}

type ResourceAiSearchIndexDirectAccessIndexSpecEmbeddingVectorColumns struct {
	EmbeddingDimension int    `json:"embedding_dimension,omitempty"`
	Name               string `json:"name,omitempty"`
}

type ResourceAiSearchIndexDirectAccessIndexSpec struct {
	EmbeddingSourceColumns []ResourceAiSearchIndexDirectAccessIndexSpecEmbeddingSourceColumns `json:"embedding_source_columns,omitempty"`
	EmbeddingVectorColumns []ResourceAiSearchIndexDirectAccessIndexSpecEmbeddingVectorColumns `json:"embedding_vector_columns,omitempty"`
	SchemaJson             string                                                             `json:"schema_json,omitempty"`
}

type ResourceAiSearchIndexProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceAiSearchIndexStatus struct {
	IndexUrl        string `json:"index_url,omitempty"`
	IndexedRowCount int    `json:"indexed_row_count,omitempty"`
	Message         string `json:"message,omitempty"`
	Ready           bool   `json:"ready,omitempty"`
}

type ResourceAiSearchIndex struct {
	Creator               string                                      `json:"creator,omitempty"`
	DeltaSyncIndexSpec    *ResourceAiSearchIndexDeltaSyncIndexSpec    `json:"delta_sync_index_spec,omitempty"`
	DirectAccessIndexSpec *ResourceAiSearchIndexDirectAccessIndexSpec `json:"direct_access_index_spec,omitempty"`
	Endpoint              string                                      `json:"endpoint,omitempty"`
	IndexId               string                                      `json:"index_id,omitempty"`
	IndexSubtype          string                                      `json:"index_subtype,omitempty"`
	IndexType             string                                      `json:"index_type"`
	Name                  string                                      `json:"name,omitempty"`
	Parent                string                                      `json:"parent"`
	PrimaryKey            string                                      `json:"primary_key"`
	ProviderConfig        *ResourceAiSearchIndexProviderConfig        `json:"provider_config,omitempty"`
	Status                *ResourceAiSearchIndexStatus                `json:"status,omitempty"`
}
