package dresources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/common"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaterializationSetMemberKey(t *testing.T) {
	offline := ml.MaterializedFeature{
		FeatureName:        "main.default.feat",
		OfflineStoreConfig: &ml.OfflineStoreConfig{CatalogName: "main", SchemaName: "default"},
	}
	online := ml.MaterializedFeature{
		FeatureName:       "main.default.feat",
		OnlineStoreConfig: &ml.OnlineStoreConfig{OnlineStoreName: "store_one"},
	}
	streaming := ml.MaterializedFeature{
		FeatureName:        "main.default.feat",
		OfflineStoreConfig: &ml.OfflineStoreConfig{CatalogName: "main", SchemaName: "default"},
		StreamingMode:      &ml.StreamingMode{},
	}

	field, value := materializationSetMemberKey(offline)
	assert.Equal(t, "member", field)
	assert.Equal(t, "main.default.feat||false", value)

	_, onlineValue := materializationSetMemberKey(online)
	assert.Equal(t, "main.default.feat|store_one|false", onlineValue)

	_, streamingValue := materializationSetMemberKey(streaming)
	assert.Equal(t, "main.default.feat||true", streamingValue)

	// The three members of one feature must not collide with each other.
	assert.NotEqual(t, value, onlineValue)
	assert.NotEqual(t, value, streamingValue)
}

// A feature name may contain the separator, so without escaping these two distinct members would
// share a key: the plan would diff them as one and the update request would omit the member the
// server then deletes.
func TestMaterializationSetMemberKeyEscapesSeparator(t *testing.T) {
	nameHoldsSeparator := ml.MaterializedFeature{
		FeatureName:       "main.default.a|b",
		OnlineStoreConfig: &ml.OnlineStoreConfig{OnlineStoreName: "c"},
	}
	storeHoldsSeparator := ml.MaterializedFeature{
		FeatureName:       "main.default.a",
		OnlineStoreConfig: &ml.OnlineStoreConfig{OnlineStoreName: "b|c"},
	}

	_, first := materializationSetMemberKey(nameHoldsSeparator)
	_, second := materializationSetMemberKey(storeHoldsSeparator)
	assert.NotEqual(t, first, second)

	// The escape character itself is escaped first, so a name cannot forge an escaped separator.
	_, forged := materializationSetMemberKey(ml.MaterializedFeature{
		FeatureName:       "main.default.a%7Cb",
		OnlineStoreConfig: &ml.OnlineStoreConfig{OnlineStoreName: "c"},
	})
	assert.NotEqual(t, first, forged)
}

// operationStub stands in for a generated operation interface. The three verbs' interfaces differ
// only in what Wait returns, which materializationSetOperationTarget never calls.
type operationStub struct {
	name     string
	metadata *ml.MaterializationSetOperationMetadata
	err      error
}

func (s operationStub) Name() string { return s.name }

func (s operationStub) Metadata() (*ml.MaterializationSetOperationMetadata, error) {
	return s.metadata, s.err
}

// The set id names what a failed create, update or delete was changing. The server assigns it
// before reserving the operation record, so the fallback to the operation name covers the cases
// where the metadata is unreadable rather than a set that has no id.
func TestMaterializationSetOperationTarget(t *testing.T) {
	assert.Equal(t, "materialization set set-1", materializationSetOperationTarget(operationStub{
		name:     "operations/abc",
		metadata: &ml.MaterializationSetOperationMetadata{MaterializationSetId: "set-1"},
	}))

	assert.Equal(t, "operation operations/abc", materializationSetOperationTarget(operationStub{
		name:     "operations/abc",
		metadata: &ml.MaterializationSetOperationMetadata{},
	}))

	assert.Equal(t, "operation operations/abc", materializationSetOperationTarget(operationStub{
		name: "operations/abc",
	}))

	assert.Equal(t, "operation operations/abc", materializationSetOperationTarget(operationStub{
		name: "operations/abc",
		err:  errors.New("metadata is not a MaterializationSetOperationMetadata"),
	}))
}

// apiErrorWithDetails builds the error the SDK produces from a response body carrying error
// details, which is the only way the per-member outcome of a partly applied change reaches a
// client.
func apiErrorWithDetails(t *testing.T, message string, details []map[string]any) error {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"error_code": "RESOURCE_CONFLICT",
		"message":    message,
		"details":    details,
	})
	require.NoError(t, err)

	resp := &http.Response{
		StatusCode: http.StatusConflict,
		Status:     "409 Conflict",
		Request:    &http.Request{Method: http.MethodPatch, URL: &url.URL{Path: "/api/2.0/x"}},
	}
	return apierr.GetAPIError(context.Background(), common.ResponseWrapper{
		Response:   resp,
		DebugBytes: body,
		ReadCloser: io.NopCloser(bytes.NewReader(body)),
	})
}

func updateProgressDetail(failures []map[string]any) map[string]any {
	return map[string]any{
		"@type":                            "type.googleapis.com/databricks.featurestore.MaterializationSetUpdateProgress",
		"deleted_materialized_feature_ids": []any{"mf-gone"},
		"failed_materialized_features":     failures,
	}
}

// The server's message defers to an error detail the CLI never prints, so the adapter has to
// restate the failed members in the error itself.
func TestMaterializationSetChangeFailuresAreRestatedInTheError(t *testing.T) {
	err := apiErrorWithDetails(t,
		"Changed 1 of 2 materialized features in materialization set 'set-1'; see the error "+
			"details for which ones failed.",
		[]map[string]any{updateProgressDetail([]map[string]any{{
			"materialized_feature_id": "mf-stuck",
			"error_code":              "RESOURCE_CONFLICT",
			"message":                 "pipeline is updating",
		}})})

	wrapped := withMaterializationSetChangeFailures(err)
	assert.Contains(t, wrapped.Error(), "members left unchanged:")
	assert.Contains(t, wrapped.Error(), "mf-stuck (RESOURCE_CONFLICT): pipeline is updating")
	// The server's own message survives, so nothing a caller already relied on is lost.
	assert.Contains(t, wrapped.Error(), "Changed 1 of 2 materialized features")
	assert.ErrorIs(t, wrapped, err)
}

// Every validation failure is raised before the server touches a member, so it carries no progress
// and must come back exactly as it arrived.
func TestMaterializationSetChangeFailuresLeavesOtherErrorsAlone(t *testing.T) {
	withoutDetails := apiErrorWithDetails(t, "Materialization set 'set-1' does not exist.", nil)
	assert.Equal(t, withoutDetails, withMaterializationSetChangeFailures(withoutDetails))

	// A progress detail naming no failed member is a succeeded change, not a partial one.
	empty := apiErrorWithDetails(t, "transient", []map[string]any{updateProgressDetail(nil)})
	assert.Equal(t, empty, withMaterializationSetChangeFailures(empty))

	plain := errors.New("connection reset")
	assert.Equal(t, plain, withMaterializationSetChangeFailures(plain))
}
