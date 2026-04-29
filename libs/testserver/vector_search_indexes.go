package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

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
	if _, pending := s.vectorSearchIndexesPendingDeletion[createReq.Name]; pending {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    fmt.Sprintf("Index %s is currently pending deletion. Operations on the index are not permitted while the index is being deleted.", createReq.Name),
			},
		}
	}
	if _, exists := s.VectorSearchEndpoints[createReq.EndpointName]; !exists {
		return Response{
			StatusCode: http.StatusNotFound,
			Body: map[string]string{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    fmt.Sprintf("Vector search endpoint %s not found", createReq.EndpointName),
			},
		}
	}

	index := vectorsearch.VectorIndex{
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

// VectorSearchIndexDelete moves the index out of the live map into a
// "pending deletion" buffer. CREATE on the same name returns the
// pending-deletion error until a GET (i.e. a poll) consumes the buffer entry,
// which is how the real backend's async deletion window manifests to a client
// that doesn't wait between DELETE and CREATE.
func (s *FakeWorkspace) VectorSearchIndexDelete(indexName string) Response {
	defer s.LockUnlock()()

	index, ok := s.VectorSearchIndexes[indexName]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body: map[string]string{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    fmt.Sprintf("Vector search index %s not found", indexName),
			},
		}
	}
	if index.Status == nil {
		index.Status = &vectorsearch.VectorIndexStatus{}
	}
	index.Status.Ready = false
	index.Status.Message = "Index is currently pending deletion"
	s.vectorSearchIndexesPendingDeletion[indexName] = index
	delete(s.VectorSearchIndexes, indexName)
	return Response{}
}

// VectorSearchIndexGet returns 404 once the index has been moved to the
// pending-deletion buffer. The pending entry is consumed as a side effect, so
// CREATE-during-pending only fires when the caller skips the wait and races
// straight from DELETE to CREATE without polling. This preserves the
// "remote not found -> recreate" behavior for tests that delete out-of-band
// and immediately re-plan.
func (s *FakeWorkspace) VectorSearchIndexGet(indexName string) Response {
	defer s.LockUnlock()()

	if index, ok := s.VectorSearchIndexes[indexName]; ok {
		return Response{Body: index}
	}
	delete(s.vectorSearchIndexesPendingDeletion, indexName)
	return Response{
		StatusCode: http.StatusNotFound,
		Body: map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    fmt.Sprintf("Vector search index %s not found", indexName),
		},
	}
}
