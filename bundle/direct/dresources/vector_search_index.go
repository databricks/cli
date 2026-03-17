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

func (*ResourceVectorSearchIndex) RemapState(info *vectorsearch.VectorIndex) *vectorsearch.CreateVectorIndexRequest {
	// Response spec types (DeltaSyncVectorIndexSpecResponse, etc.) differ from request types,
	// so we only remap the scalar fields for drift comparison.
	return &vectorsearch.CreateVectorIndexRequest{
		DeltaSyncIndexSpec:    nil, // response spec types differ from request; only scalar fields remapped
		DirectAccessIndexSpec: nil,
		EndpointName:          info.EndpointName,
		IndexType:             info.IndexType,
		Name:                  info.Name,
		PrimaryKey:            info.PrimaryKey,
	}
}

func (r *ResourceVectorSearchIndex) DoRead(ctx context.Context, id string) (*vectorsearch.VectorIndex, error) {
	return r.client.VectorSearchIndexes.GetIndex(ctx, vectorsearch.GetIndexRequest{
		IndexName:                id,
		EnsureRerankerCompatible: false,
		ForceSendFields:          nil,
	})
}

func (r *ResourceVectorSearchIndex) DoCreate(ctx context.Context, config *vectorsearch.CreateVectorIndexRequest) (string, *vectorsearch.VectorIndex, error) {
	response, err := r.client.VectorSearchIndexes.CreateIndex(ctx, *config)
	if err != nil {
		return "", nil, err
	}
	if response == nil {
		return config.Name, nil, nil
	}
	return response.Name, response, nil
}

func (r *ResourceVectorSearchIndex) DoDelete(ctx context.Context, id string) error {
	return r.client.VectorSearchIndexes.DeleteIndex(ctx, vectorsearch.DeleteIndexRequest{
		IndexName: id,
	})
}
