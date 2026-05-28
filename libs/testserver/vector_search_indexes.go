package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

// fakeVectorSearchIndex captures the endpoint's UUID at index creation time.
// On the real backend an index is bound to a specific endpoint instance, not
// just the name: deleting and recreating an endpoint with the same name yields
// a different UUID, and the existing index keeps pointing at the OLD UUID
// (i.e. is orphaned). Tracking this here lets tests reason about that drift.
// The field is omitted from JSON responses since the real API doesn't return
// it on the index path; the CLI looks it up via GetEndpointByEndpointName.
type fakeVectorSearchIndex struct {
	vectorsearch.VectorIndex
	EndpointUuid string `json:"-"`
}

func (s *FakeWorkspace) VectorSearchIndexCreate(req Request) Response {
	defer s.LockUnlock()()

	var createReq vectorsearch.CreateVectorIndexRequest
	if err := json.Unmarshal(req.Body, &createReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	if _, exists := s.VectorSearchIndexes[createReq.Name]; exists {
		return Response{
			StatusCode: http.StatusConflict,
			Body:       map[string]string{"error_code": "RESOURCE_ALREADY_EXISTS", "message": fmt.Sprintf("Vector search index with name %s already exists", createReq.Name)},
		}
	}
	endpoint, exists := s.VectorSearchEndpoints[createReq.EndpointName]
	if !exists {
		return Response{
			StatusCode: http.StatusNotFound,
			Body: map[string]string{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    fmt.Sprintf("Vector search endpoint %s not found", createReq.EndpointName),
			},
		}
	}

	index := fakeVectorSearchIndex{
		VectorIndex: vectorsearch.VectorIndex{
			Creator:               s.CurrentUser().UserName,
			EndpointName:          createReq.EndpointName,
			IndexType:             createReq.IndexType,
			Name:                  createReq.Name,
			PrimaryKey:            createReq.PrimaryKey,
			DeltaSyncIndexSpec:    remapDeltaSyncSpec(createReq.DeltaSyncIndexSpec),
			DirectAccessIndexSpec: createReq.DirectAccessIndexSpec,
			Status: &vectorsearch.VectorIndexStatus{
				Ready: true,
			},
		},
		EndpointUuid: endpoint.Id,
	}

	s.VectorSearchIndexes[createReq.Name] = index

	return Response{
		Body: index,
	}
}

// remapDeltaSyncSpec converts a request spec to a response spec.
func remapDeltaSyncSpec(req *vectorsearch.DeltaSyncVectorIndexSpecRequest) *vectorsearch.DeltaSyncVectorIndexSpecResponse {
	if req == nil {
		return nil
	}
	return &vectorsearch.DeltaSyncVectorIndexSpecResponse{
		EmbeddingSourceColumns:  req.EmbeddingSourceColumns,
		EmbeddingVectorColumns:  req.EmbeddingVectorColumns,
		EmbeddingWritebackTable: req.EmbeddingWritebackTable,
		PipelineType:            req.PipelineType,
		SourceTable:             req.SourceTable,
	}
}
