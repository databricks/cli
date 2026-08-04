package testserver

import (
	"bytes"
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

// indexNamePendingDeletion scopes the two-phase deletion simulation to the
// pending-deletion recreate test. On the real backend every delete goes through
// both phases, but the second phase usually completes before the CLI gets to
// CREATE, so modelling it unconditionally would add a spurious retry to every
// recreate test. Mirrors catalogNameManagedDefaults.
const indexNamePendingDeletion = "vs_index_pending_deletion"

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
	if s.VectorSearchIndexesPendingDeletion[createReq.Name] > 0 {
		s.VectorSearchIndexesPendingDeletion[createReq.Name]--
		return Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    fmt.Sprintf("Index %s is currently pending deletion. Operations on the index are not permitted while the index is being deleted.", createReq.Name),
			},
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

	// The backend rewrites schema_json on create: user-facing type names are
	// stored as Spark type names (e.g. "integer" -> "int") and the columns are
	// returned in sorted key order rather than the user's original order.
	// Mirror that here so the create -> get round-trip matches the real API.
	if createReq.DirectAccessIndexSpec != nil {
		createReq.DirectAccessIndexSpec.SchemaJson = normalizeSchemaJSON(createReq.DirectAccessIndexSpec.SchemaJson)
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

// VectorSearchIndexDelete removes the index and, for the pending-deletion test
// name, records that the next CREATE for the same name must still be rejected.
// The real backend releases the name only after the index has already stopped
// being visible on GET, so a CLI that polls GET for absence can observe the
// index as gone and still lose the race on CREATE.
func (s *FakeWorkspace) VectorSearchIndexDelete(indexName string) Response {
	defer s.LockUnlock()()

	if _, ok := s.VectorSearchIndexes[indexName]; !ok {
		return Response{StatusCode: http.StatusNotFound}
	}
	delete(s.VectorSearchIndexes, indexName)
	if strings.Contains(indexName, indexNamePendingDeletion) {
		s.VectorSearchIndexesPendingDeletion[indexName] = 1
	}
	return Response{}
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

// normalizeSchemaJSON rewrites a schema_json document the way the backend
// stores it: user-facing column type names are folded to Spark type names and
// the columns are re-serialized in sorted key order (encoding/json sorts map
// keys, matching the backend). Returns the input unchanged when it isn't the
// expected {"column":"type"} JSON object.
func normalizeSchemaJSON(schemaJSON string) string {
	if schemaJSON == "" {
		return schemaJSON
	}
	var schema map[string]string
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return schemaJSON
	}
	for column, columnType := range schema {
		schema[column] = normalizeColumnType(columnType)
	}
	// Disable HTML escaping so array<...> keeps its angle brackets verbatim
	// rather than being rewritten to < / >.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(schema); err != nil {
		return schemaJSON
	}
	return strings.TrimRight(buf.String(), "\n")
}

// normalizeColumnType maps the user-facing column type names the Vector
// Search API accepts ("integer", "long", "short", "byte") to the Spark type
// names Unity Catalog stores and GET returns, recursing into array element
// types. Types whose user-facing and Spark spellings coincide ("float",
// "string", ...) pass through unchanged.
func normalizeColumnType(columnType string) string {
	if inner, ok := strings.CutPrefix(columnType, "array<"); ok {
		if elem, ok := strings.CutSuffix(inner, ">"); ok {
			return "array<" + normalizeColumnType(elem) + ">"
		}
	}
	switch columnType {
	case "integer":
		return "int"
	case "long":
		return "bigint"
	case "short":
		return "smallint"
	case "byte":
		return "tinyint"
	default:
		return columnType
	}
}

// remapDeltaSyncSpec converts a request spec to a response spec.
func remapDeltaSyncSpec(req *vectorsearch.DeltaSyncVectorIndexSpecRequest) *vectorsearch.DeltaSyncVectorIndexSpecResponse {
	if req == nil {
		return nil
	}
	return &vectorsearch.DeltaSyncVectorIndexSpecResponse{
		ColumnsToIndex:          req.ColumnsToIndex,
		ColumnsToSync:           req.ColumnsToSync,
		EmbeddingSourceColumns:  req.EmbeddingSourceColumns,
		EmbeddingVectorColumns:  req.EmbeddingVectorColumns,
		EmbeddingWritebackTable: req.EmbeddingWritebackTable,
		PipelineType:            req.PipelineType,
		SourceTable:             req.SourceTable,
	}
}
