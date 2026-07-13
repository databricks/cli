// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAiSearchIndexesIndexesDeltaSyncIndexSpecEmbeddingSourceColumns struct {
	EmbeddingModelEndpoint    string `json:"embedding_model_endpoint,omitempty"`
	ModelEndpointNameForQuery string `json:"model_endpoint_name_for_query,omitempty"`
	Name                      string `json:"name,omitempty"`
}

type DataSourceAiSearchIndexesIndexesDeltaSyncIndexSpecEmbeddingVectorColumns struct {
	EmbeddingDimension int    `json:"embedding_dimension,omitempty"`
	Name               string `json:"name,omitempty"`
}

type DataSourceAiSearchIndexesIndexesDeltaSyncIndexSpec struct {
	ColumnsToSync           []string                                                                   `json:"columns_to_sync,omitempty"`
	EmbeddingSourceColumns  []DataSourceAiSearchIndexesIndexesDeltaSyncIndexSpecEmbeddingSourceColumns `json:"embedding_source_columns,omitempty"`
	EmbeddingVectorColumns  []DataSourceAiSearchIndexesIndexesDeltaSyncIndexSpecEmbeddingVectorColumns `json:"embedding_vector_columns,omitempty"`
	EmbeddingWritebackTable string                                                                     `json:"embedding_writeback_table,omitempty"`
	PipelineId              string                                                                     `json:"pipeline_id,omitempty"`
	PipelineType            string                                                                     `json:"pipeline_type"`
	SourceTable             string                                                                     `json:"source_table,omitempty"`
}

type DataSourceAiSearchIndexesIndexesDirectAccessIndexSpecEmbeddingSourceColumns struct {
	EmbeddingModelEndpoint    string `json:"embedding_model_endpoint,omitempty"`
	ModelEndpointNameForQuery string `json:"model_endpoint_name_for_query,omitempty"`
	Name                      string `json:"name,omitempty"`
}

type DataSourceAiSearchIndexesIndexesDirectAccessIndexSpecEmbeddingVectorColumns struct {
	EmbeddingDimension int    `json:"embedding_dimension,omitempty"`
	Name               string `json:"name,omitempty"`
}

type DataSourceAiSearchIndexesIndexesDirectAccessIndexSpec struct {
	EmbeddingSourceColumns []DataSourceAiSearchIndexesIndexesDirectAccessIndexSpecEmbeddingSourceColumns `json:"embedding_source_columns,omitempty"`
	EmbeddingVectorColumns []DataSourceAiSearchIndexesIndexesDirectAccessIndexSpecEmbeddingVectorColumns `json:"embedding_vector_columns,omitempty"`
	SchemaJson             string                                                                        `json:"schema_json,omitempty"`
}

type DataSourceAiSearchIndexesIndexesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiSearchIndexesIndexesStatus struct {
	IndexUrl        string `json:"index_url,omitempty"`
	IndexedRowCount int    `json:"indexed_row_count,omitempty"`
	Message         string `json:"message,omitempty"`
	Ready           bool   `json:"ready,omitempty"`
}

type DataSourceAiSearchIndexesIndexes struct {
	Creator               string                                                 `json:"creator,omitempty"`
	DeltaSyncIndexSpec    *DataSourceAiSearchIndexesIndexesDeltaSyncIndexSpec    `json:"delta_sync_index_spec,omitempty"`
	DirectAccessIndexSpec *DataSourceAiSearchIndexesIndexesDirectAccessIndexSpec `json:"direct_access_index_spec,omitempty"`
	Endpoint              string                                                 `json:"endpoint,omitempty"`
	IndexSubtype          string                                                 `json:"index_subtype,omitempty"`
	IndexType             string                                                 `json:"index_type,omitempty"`
	Name                  string                                                 `json:"name"`
	PrimaryKey            string                                                 `json:"primary_key,omitempty"`
	ProviderConfig        *DataSourceAiSearchIndexesIndexesProviderConfig        `json:"provider_config,omitempty"`
	Status                *DataSourceAiSearchIndexesIndexesStatus                `json:"status,omitempty"`
}

type DataSourceAiSearchIndexesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceAiSearchIndexes struct {
	Indexes        []DataSourceAiSearchIndexesIndexes       `json:"indexes,omitempty"`
	PageSize       int                                      `json:"page_size,omitempty"`
	Parent         string                                   `json:"parent"`
	ProviderConfig *DataSourceAiSearchIndexesProviderConfig `json:"provider_config,omitempty"`
}
