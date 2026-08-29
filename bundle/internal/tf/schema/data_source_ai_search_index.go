// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAiSearchIndexDeltaSyncIndexSpecEmbeddingSourceColumns struct {
	EmbeddingModelEndpoint    string `json:"embedding_model_endpoint,omitempty"`
	ModelEndpointNameForQuery string `json:"model_endpoint_name_for_query,omitempty"`
	Name                      string `json:"name,omitempty"`
}

type DataSourceAiSearchIndexDeltaSyncIndexSpecEmbeddingVectorColumns struct {
	EmbeddingDimension int    `json:"embedding_dimension,omitempty"`
	Name               string `json:"name,omitempty"`
}

type DataSourceAiSearchIndexDeltaSyncIndexSpec struct {
	ColumnsToSync           []string                                                          `json:"columns_to_sync,omitempty"`
	EmbeddingSourceColumns  []DataSourceAiSearchIndexDeltaSyncIndexSpecEmbeddingSourceColumns `json:"embedding_source_columns,omitempty"`
	EmbeddingVectorColumns  []DataSourceAiSearchIndexDeltaSyncIndexSpecEmbeddingVectorColumns `json:"embedding_vector_columns,omitempty"`
	EmbeddingWritebackTable string                                                            `json:"embedding_writeback_table,omitempty"`
	PipelineId              string                                                            `json:"pipeline_id,omitempty"`
	PipelineType            string                                                            `json:"pipeline_type"`
	SourceTable             string                                                            `json:"source_table,omitempty"`
}

type DataSourceAiSearchIndexDirectAccessIndexSpecEmbeddingSourceColumns struct {
	EmbeddingModelEndpoint    string `json:"embedding_model_endpoint,omitempty"`
	ModelEndpointNameForQuery string `json:"model_endpoint_name_for_query,omitempty"`
	Name                      string `json:"name,omitempty"`
}

type DataSourceAiSearchIndexDirectAccessIndexSpecEmbeddingVectorColumns struct {
	EmbeddingDimension int    `json:"embedding_dimension,omitempty"`
	Name               string `json:"name,omitempty"`
}

type DataSourceAiSearchIndexDirectAccessIndexSpec struct {
	EmbeddingSourceColumns []DataSourceAiSearchIndexDirectAccessIndexSpecEmbeddingSourceColumns `json:"embedding_source_columns,omitempty"`
	EmbeddingVectorColumns []DataSourceAiSearchIndexDirectAccessIndexSpecEmbeddingVectorColumns `json:"embedding_vector_columns,omitempty"`
	SchemaJson             string                                                               `json:"schema_json,omitempty"`
}

type DataSourceAiSearchIndexProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiSearchIndexStatus struct {
	IndexUrl        string `json:"index_url,omitempty"`
	IndexedRowCount int    `json:"indexed_row_count,omitempty"`
	Message         string `json:"message,omitempty"`
	Ready           bool   `json:"ready,omitempty"`
}

type DataSourceAiSearchIndex struct {
	Creator               string                                        `json:"creator,omitempty"`
	DeltaSyncIndexSpec    *DataSourceAiSearchIndexDeltaSyncIndexSpec    `json:"delta_sync_index_spec,omitempty"`
	DirectAccessIndexSpec *DataSourceAiSearchIndexDirectAccessIndexSpec `json:"direct_access_index_spec,omitempty"`
	Endpoint              string                                        `json:"endpoint,omitempty"`
	IndexSubtype          string                                        `json:"index_subtype,omitempty"`
	IndexType             string                                        `json:"index_type,omitempty"`
	Name                  string                                        `json:"name"`
	PrimaryKey            string                                        `json:"primary_key,omitempty"`
	ProviderConfig        *DataSourceAiSearchIndexProviderConfig        `json:"provider_config,omitempty"`
	Status                *DataSourceAiSearchIndexStatus                `json:"status,omitempty"`
}
