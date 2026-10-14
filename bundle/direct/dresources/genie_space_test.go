package dresources

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/sdk-go/core/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func apiError(status int, errorCode, message string) error {
	body, _ := json.Marshal(map[string]string{"error_code": errorCode, "message": message})
	return apierr.FromHTTPError(status, nil, body)
}

func TestIsMissingGenieParentPathError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "standard missing error",
			err:  apiError(404, "NOT_FOUND", "not found"),
			want: true,
		},
		{
			name: "invalid parameter tree node missing error",
			err:  apiError(400, "INVALID_PARAMETER_VALUE", "NOT_FOUND: Tree node with path /Workspace/foo does not exist"),
			want: true,
		},
		{
			name: "other invalid parameter error",
			err:  apiError(400, "INVALID_PARAMETER_VALUE", "some other validation failure"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isMissingGenieParentPathError(tt.err))
		})
	}
}

// newGenieTestClient returns a client that talks to server.
func newGenieTestClient(t *testing.T, server *testserver.Server) *sdkclients.Clients {
	client, err := sdkclients.New(t.Context(), &config.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)
	return client
}

func TestGenieSpaceDoCreateRetriesWhenParentPathLooksMissing(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)

	var createBodies []string
	server.Handle("POST", "/api/2.0/genie/spaces", func(req testserver.Request) any {
		createBodies = append(createBodies, string(req.Body))
		if len(createBodies) == 1 {
			return testserver.Response{
				StatusCode: 400,
				Body: map[string]string{
					"error_code": "INVALID_PARAMETER_VALUE",
					"message":    "NOT_FOUND: Tree node with path /Workspace/test-parent does not exist",
				},
			}
		}
		return map[string]string{
			"space_id":         "space-id",
			"title":            "test genie space",
			"description":      "test description",
			"warehouse_id":     "test-warehouse-id",
			"serialized_space": "{}",
		}
	})
	var mkdirsBodies []string
	server.Handle("POST", "/api/2.0/workspace/mkdirs", func(req testserver.Request) any {
		mkdirsBodies = append(mkdirsBodies, string(req.Body))
		return map[string]string{}
	})

	r := (&ResourceGenieSpace{}).New(newGenieTestClient(t, server))
	id, state, err := r.DoCreate(ctx, &resources.GenieSpaceConfig{
		Title:           "test genie space",
		Description:     "test description",
		ParentPath:      "/Workspace/test-parent",
		WarehouseId:     "test-warehouse-id",
		SerializedSpace: "{}",
	})
	require.NoError(t, err)
	assert.Equal(t, "space-id", id)
	require.NotNil(t, state)
	assert.Equal(t, "test genie space", state.Title)

	want := `{"warehouse_id":"test-warehouse-id","parent_path":"/Workspace/test-parent","serialized_space":"{}","title":"test genie space","description":"test description"}`
	require.Len(t, createBodies, 2)
	assert.JSONEq(t, want, createBodies[0])
	assert.JSONEq(t, want, createBodies[1])
	require.Len(t, mkdirsBodies, 1)
	assert.JSONEq(t, `{"path":"/Workspace/test-parent"}`, mkdirsBodies[0])
}

func TestGenieSpaceDoUpdateRoundTripsEtag(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)

	var updateBodies []string
	server.Handle("PATCH", "/api/2.0/genie/spaces/{id}", func(req testserver.Request) any {
		updateBodies = append(updateBodies, string(req.Body))
		return map[string]string{"space_id": "space-id", "title": "new", "etag": "etag-8"}
	})

	r := (&ResourceGenieSpace{}).New(newGenieTestClient(t, server))
	entry := &deployplan.PlanEntry{
		Changes: deployplan.Changes{
			"title": {Action: deployplan.Update, Old: "old", New: "new"},
		},
	}

	state, err := r.DoUpdate(ctx, "space-id", &resources.GenieSpaceConfig{
		Title: "new",
		Etag:  "etag-7",
	}, entry)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, "etag-8", state.Etag)

	// The stored etag (etag-7) must NOT be sent as an If-Match guard — it would
	// 409 after a backend serialized_space schema migration. Only the etag from
	// the response is persisted, for drift detection on the next plan.
	require.Len(t, updateBodies, 1)
	assert.JSONEq(t, `{"space_id":"space-id","title":"new","description":""}`, updateBodies[0])
}

func TestGenieSpaceDoUpdateAlwaysSendsSerializedSpace(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)

	var updateBodies []string
	server.Handle("PATCH", "/api/2.0/genie/spaces/{id}", func(req testserver.Request) any {
		updateBodies = append(updateBodies, string(req.Body))
		return map[string]string{"space_id": "space-id", "title": "new", "serialized_space": `{"converge":"me"}`}
	})

	r := (&ResourceGenieSpace{}).New(newGenieTestClient(t, server))

	// Even though the plan entry only marks an unrelated field changed, the body
	// is sent so the deploy converges the space to the bundle config.
	entry := &deployplan.PlanEntry{
		Changes: deployplan.Changes{
			"title": {Action: deployplan.Update, Old: "old", New: "new"},
		},
	}

	state, err := r.DoUpdate(ctx, "space-id", &resources.GenieSpaceConfig{
		Title:           "new",
		SerializedSpace: "{\"converge\":\"me\"}",
	}, entry)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, "{\"converge\":\"me\"}", state.SerializedSpace)

	require.Len(t, updateBodies, 1)
	assert.JSONEq(t, `{"space_id":"space-id","title":"new","description":"","serialized_space":"{\"converge\":\"me\"}"}`, updateBodies[0])
}

func TestGenieSpaceOverrideChangeDescEtag(t *testing.T) {
	r := &ResourceGenieSpace{}
	etagPath := structpath.MustParsePath("etag")

	t.Run("Skip when stored matches remote", func(t *testing.T) {
		change := &ChangeDesc{Old: "etag-7", Remote: "etag-7"}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), etagPath, change, nil))
		assert.Equal(t, deployplan.Skip, change.Action)
	})

	t.Run("Update when stored differs from remote", func(t *testing.T) {
		change := &ChangeDesc{Old: "etag-7", Remote: "etag-8"}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), etagPath, change, nil))
		assert.Equal(t, deployplan.Update, change.Action)
	})

	t.Run("Other paths are untouched", func(t *testing.T) {
		titlePath := structpath.MustParsePath("title")
		change := &ChangeDesc{Action: deployplan.Update, Old: "a", Remote: "b"}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), titlePath, change, nil))
		assert.Equal(t, deployplan.Update, change.Action)
	})
}
