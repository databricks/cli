package dresources

import (
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVectorSearchIndexAllSDKFieldsAreClassified guards against a future SDK
// bump silently adding a field that the planner classifies as Update. The
// resource has no update API and intentionally omits DoUpdate, so any
// unclassified field would surface as a deploy-time framework error
// ("resource does not support update action but plan produced update"). This
// test catches the gap at unit-test time instead.
func TestVectorSearchIndexAllSDKFieldsAreClassified(t *testing.T) {
	config := GetResourceConfig("vector_search_indexes")
	require.NotNil(t, config)

	classified := map[string]bool{}
	for _, field := range config.RecreateOnChanges {
		classified[field.Field.String()] = true
	}
	// provided_id_fields also recreate on local changes, so they are classified.
	for _, field := range config.ProvidedIDFields {
		classified[field.Field.String()] = true
	}
	for _, field := range config.IgnoreRemoteChanges {
		classified[field.Field.String()] = true
	}

	sdkType := reflect.TypeFor[vectorsearch.CreateVectorIndexRequest]()
	for field := range sdkType.Fields() {
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		jsonTag = strings.TrimSuffix(jsonTag, ",omitempty")
		assert.Truef(t, classified[jsonTag],
			"field %q is not declared in vector_search_indexes.yml; "+
				"vector_search_indexes has no update API, so every SDK field must be in "+
				"recreate_on_changes, provided_id_fields or ignore_remote_changes",
			jsonTag,
		)
	}
}

// TestVectorSearchIndexCreateIndexHonorsResourceMaxWait verifies the recreate create retry is bounded.
func TestVectorSearchIndexCreateIndexHonorsResourceMaxWait(t *testing.T) {
	server := testserver.New(t)
	var calls atomic.Int32
	server.Handle("POST", "/api/2.0/vector-search/indexes", func(testserver.Request) any {
		calls.Add(1)
		return testserver.Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    "Index is currently pending deletion",
			},
		}
	})
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	ctx := WithResourceMaxWait(t.Context(), 100*time.Millisecond)
	start := time.Now()
	_, err = (&ResourceVectorSearchIndex{}).New(client).createIndex(ctx, vectorsearch.CreateVectorIndexRequest{Name: "main.default.index"})
	require.Error(t, err)
	assert.Less(t, time.Since(start), time.Second)
	assert.Equal(t, int32(1), calls.Load())
}

// TestVectorSearchIndexWaitAfterDeleteHonorsResourceMaxWait verifies the post-delete poll is bounded.
func TestVectorSearchIndexWaitAfterDeleteHonorsResourceMaxWait(t *testing.T) {
	server := testserver.New(t)
	var calls atomic.Int32
	server.Handle("GET", "/api/2.0/vector-search/indexes/{index_name}", func(testserver.Request) any {
		calls.Add(1)
		return vectorsearch.VectorIndex{Name: "main.default.index"}
	})
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	ctx := WithResourceMaxWait(t.Context(), 100*time.Millisecond)
	start := time.Now()
	err = (&ResourceVectorSearchIndex{}).New(client).WaitAfterDelete(ctx, "main.default.index")
	require.Error(t, err)
	assert.Less(t, time.Since(start), time.Second)
	assert.Equal(t, int32(1), calls.Load())
}

// TestVectorSearchIndexMaxWaitZeroPreservesInternalMaximums verifies zero means no cap for these resource-owned polls.
func TestVectorSearchIndexMaxWaitZeroPreservesInternalMaximums(t *testing.T) {
	ctx := WithResourceMaxWait(t.Context(), 0)
	assert.Equal(t, pendingDeletionTimeout, vectorSearchIndexTimeout(ctx, pendingDeletionTimeout))
	assert.Equal(t, deleteIndexTimeout, vectorSearchIndexTimeout(ctx, deleteIndexTimeout))
}

// TestVectorSearchIndexMaxWaitLargerThanInternalMaximumPreservesInternalMaximum verifies resource maxima still win.
func TestVectorSearchIndexMaxWaitLargerThanInternalMaximumPreservesInternalMaximum(t *testing.T) {
	ctx := WithResourceMaxWait(t.Context(), 2*deleteIndexTimeout)
	assert.Equal(t, deleteIndexTimeout, vectorSearchIndexTimeout(ctx, deleteIndexTimeout))
}

// TestVectorSearchIndexNoMaxWaitPreservesInternalMaximum verifies an absent cap leaves the resource defaults intact.
func TestVectorSearchIndexNoMaxWaitPreservesInternalMaximum(t *testing.T) {
	assert.Equal(t, pendingDeletionTimeout, vectorSearchIndexTimeout(t.Context(), pendingDeletionTimeout))
	assert.Equal(t, deleteIndexTimeout, vectorSearchIndexTimeout(t.Context(), deleteIndexTimeout))
}
