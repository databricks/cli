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

	// The backend rewrites schema_json on create: user-facing type names are
	// stored as Spark type names (e.g. "integer" -> "int") and the columns are
	// returned in sorted key order rather than the user's original order.
	// Mirror that here so the create -> get round-trip matches the real API.
	if createReq.DirectAccessIndexSpec != nil {
		createReq.DirectAccessIndexSpec.SchemaJson = normalizeSchemaJSON(createReq.DirectAccessIndexSpec.SchemaJson)
	}

	// EndpointId is frozen at creation: the index records the UUID of the
	// endpoint instance it was bound to and never re-resolves it, so after the
	// endpoint is deleted/recreated under the same name it still reports the old
	// UUID. This mirrors the orphaned index on the real backend; the CLI detects
	// the drift by looking up the live endpoint by name, not from this field.
	index := vectorsearch.VectorIndex{
		Creator:               s.CurrentUser().UserName,
		EndpointId:            endpoint.Id,
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
