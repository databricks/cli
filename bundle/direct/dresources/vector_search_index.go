package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

type ResourceVectorSearchIndex struct {
	client *databricks.WorkspaceClient
}

func (*ResourceVectorSearchIndex) New(client *databricks.WorkspaceClient) *ResourceVectorSearchIndex {
	return &ResourceVectorSearchIndex{client: client}
}

func (*ResourceVectorSearchIndex) PrepareState(input *resources.VectorSearchIndex) *vectorsearch.CreateVectorIndexRequest {
	return &input.CreateVectorIndexRequest
}

func (*ResourceVectorSearchIndex) RemapState(remote *vectorsearch.VectorIndex) *vectorsearch.CreateVectorIndexRequest {
	req := &vectorsearch.CreateVectorIndexRequest{
		DeltaSyncIndexSpec:    nil,
		DirectAccessIndexSpec: nil,
		Name:                  remote.Name,
		EndpointName:          remote.EndpointName,
		IndexType:             remote.IndexType,
		PrimaryKey:            remote.PrimaryKey,
	}
	if remote.DeltaSyncIndexSpec != nil {
		req.DeltaSyncIndexSpec = &vectorsearch.DeltaSyncVectorIndexSpecRequest{
			ColumnsToSync:           nil,
			EmbeddingSourceColumns:  remote.DeltaSyncIndexSpec.EmbeddingSourceColumns,
			EmbeddingVectorColumns:  remote.DeltaSyncIndexSpec.EmbeddingVectorColumns,
			EmbeddingWritebackTable: remote.DeltaSyncIndexSpec.EmbeddingWritebackTable,
			PipelineType:            remote.DeltaSyncIndexSpec.PipelineType,
			SourceTable:             remote.DeltaSyncIndexSpec.SourceTable,
			ForceSendFields:         nil,
		}
	}
	if remote.DirectAccessIndexSpec != nil {
		req.DirectAccessIndexSpec = remote.DirectAccessIndexSpec
	}
	return req
}

func (r *ResourceVectorSearchIndex) DoRead(ctx context.Context, id string) (*vectorsearch.VectorIndex, error) {
	return r.client.VectorSearchIndexes.GetIndexByIndexName(ctx, id)
}

func (r *ResourceVectorSearchIndex) DoCreate(ctx context.Context, config *vectorsearch.CreateVectorIndexRequest) (string, *vectorsearch.VectorIndex, error) {
	index, err := r.client.VectorSearchIndexes.CreateIndex(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	return config.Name, index, nil
}

func (r *ResourceVectorSearchIndex) DoUpdate(ctx context.Context, id string, config *vectorsearch.CreateVectorIndexRequest, entry *PlanEntry) (*vectorsearch.VectorIndex, error) {
	// Vector search indexes have no update API; all field changes trigger recreation via resources.yml.
	return nil, nil
}

func (r *ResourceVectorSearchIndex) DoDelete(ctx context.Context, id string) error {
	return r.client.VectorSearchIndexes.DeleteIndexByIndexName(ctx, id)
}
