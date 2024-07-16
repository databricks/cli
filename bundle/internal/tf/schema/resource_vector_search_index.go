// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceVectorSearchIndexDeltaSyncIndexSpecEmbeddingSourceColumns struct {
	EmbeddingModelEndpointName string `json:"embedding_model_endpoint_name,omitempty"`
	Name                       string `json:"name,omitempty"`
}

type ResourceVectorSearchIndexDeltaSyncIndexSpecEmbeddingVectorColumns struct {
	EmbeddingDimension int    `json:"embedding_dimension,omitempty"`
	Name               string `json:"name,omitempty"`
}

type ResourceVectorSearchIndexDeltaSyncIndexSpec struct {
	EmbeddingWritebackTable string                                                              `json:"embedding_writeback_table,omitempty"`
	PipelineId              string                                                              `json:"pipeline_id,omitempty"`
	PipelineType            string                                                              `json:"pipeline_type,omitempty"`
	SourceTable             string                                                              `json:"source_table,omitempty"`
	EmbeddingSourceColumns  []ResourceVectorSearchIndexDeltaSyncIndexSpecEmbeddingSourceColumns `json:"embedding_source_columns,omitempty"`
	EmbeddingVectorColumns  []ResourceVectorSearchIndexDeltaSyncIndexSpecEmbeddingVectorColumns `json:"embedding_vector_columns,omitempty"`
}

type ResourceVectorSearchIndexDirectAccessIndexSpecEmbeddingSourceColumns struct {
	EmbeddingModelEndpointName string `json:"embedding_model_endpoint_name,omitempty"`
	Name                       string `json:"name,omitempty"`
}

type ResourceVectorSearchIndexDirectAccessIndexSpecEmbeddingVectorColumns struct {
	EmbeddingDimension int    `json:"embedding_dimension,omitempty"`
	Name               string `json:"name,omitempty"`
}

type ResourceVectorSearchIndexDirectAccessIndexSpec struct {
	SchemaJson             string                                                                 `json:"schema_json,omitempty"`
	EmbeddingSourceColumns []ResourceVectorSearchIndexDirectAccessIndexSpecEmbeddingSourceColumns `json:"embedding_source_columns,omitempty"`
	EmbeddingVectorColumns []ResourceVectorSearchIndexDirectAccessIndexSpecEmbeddingVectorColumns `json:"embedding_vector_columns,omitempty"`
}

type ResourceVectorSearchIndex struct {
	Creator               string                                          `json:"creator,omitempty"`
	EndpointName          string                                          `json:"endpoint_name"`
	Id                    string                                          `json:"id,omitempty"`
	IndexType             string                                          `json:"index_type"`
	Name                  string                                          `json:"name"`
	PrimaryKey            string                                          `json:"primary_key"`
	Status                []any                                           `json:"status,omitempty"`
	DeltaSyncIndexSpec    *ResourceVectorSearchIndexDeltaSyncIndexSpec    `json:"delta_sync_index_spec,omitempty"`
	DirectAccessIndexSpec *ResourceVectorSearchIndexDirectAccessIndexSpec `json:"direct_access_index_spec,omitempty"`
}
