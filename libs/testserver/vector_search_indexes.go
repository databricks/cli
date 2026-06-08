package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

// indexNamePart matches each catalog.schema.table component the real backend
// accepts: only alphanumerics and underscores.
var indexNamePart = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

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

	if !isValidIndexName(createReq.Name) {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    "Invalid index name. Must specify the full index name <catalog>.<schema>.<table>. Only alphanumerics and underscores are allowed.",
			},
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

	// The backend assigns index_subtype when the request omits it (HYBRID by default)
	indexSubtype := createReq.IndexSubtype
	if indexSubtype == "" {
		indexSubtype = vectorsearch.IndexSubtypeHybrid
	}

	index := fakeVectorSearchIndex{
		VectorIndex: vectorsearch.VectorIndex{
			Creator:               s.CurrentUser().UserName,
			EndpointName:          createReq.EndpointName,
			IndexType:             createReq.IndexType,
			IndexSubtype:          indexSubtype,
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

// isValidIndexName checks that name is in catalog.schema.table form with
// only alphanumerics and underscores per UC, mirroring the backend's
// validation rejection at create time.
func isValidIndexName(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if !indexNamePart.MatchString(p) {
			return false
		}
	}
	return true
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
