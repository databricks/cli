package dresources

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAppDoCreate_RetriesWhenAppIsDeleting verifies that DoCreate retries when
// an app already exists but is in DELETING state.
func TestAppDoCreate_RetriesWhenAppIsDeleting(t *testing.T) {
	server := testserver.New(t)

	createCallCount := 0
	getCallCount := 0

	server.Handle("POST", "/api/2.0/apps", func(req testserver.Request) any {
		createCallCount++
		if createCallCount == 1 {
			return testserver.Response{
				StatusCode: 409,
				Body: map[string]string{
					"error_code": "RESOURCE_ALREADY_EXISTS",
					"message":    "An app with the same name already exists.",
				},
			}
		}
		return apps.App{
			Name: new("test-app"),
			ComputeStatus: &apps.ComputeStatus{
				State: apps.ComputeStatus_ComputeState_Active,
			},
		}
	})

	server.Handle("GET", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		getCallCount++
		return apps.App{
			Name: new(req.Vars["name"]),
			ComputeStatus: &apps.ComputeStatus{
				State: apps.ComputeStatus_ComputeState_Deleting,
			},
		}
	})

	testserver.AddDefaultHandlers(server)

	client, err := sdkclients.New(t.Context(), &config.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	r := (&ResourceApp{}).New(client)
	ctx := t.Context()
	name, _, err := r.DoCreate(ctx, &AppState{App: apps.App{Name: new("test-app")}})

	require.NoError(t, err)
	assert.Equal(t, "test-app", name)
	assert.Equal(t, 2, createCallCount, "expected Create to be called twice (1 retry)")
	assert.Equal(t, 1, getCallCount, "expected Get to be called once to check app state")
}

// TestAppDoCreate_RetriesWhenGetReturnsNotFound verifies that DoCreate retries
// when the app was just deleted between the create call and the get call.
func TestAppDoCreate_RetriesWhenGetReturnsNotFound(t *testing.T) {
	server := testserver.New(t)

	createCallCount := 0
	getCallCount := 0

	server.Handle("POST", "/api/2.0/apps", func(req testserver.Request) any {
		createCallCount++
		if createCallCount == 1 {
			return testserver.Response{
				StatusCode: 409,
				Body: map[string]string{
					"error_code": "RESOURCE_ALREADY_EXISTS",
					"message":    "An app with the same name already exists.",
				},
			}
		}
		return apps.App{
			Name: new("test-app"),
			ComputeStatus: &apps.ComputeStatus{
				State: apps.ComputeStatus_ComputeState_Active,
			},
		}
	})

	server.Handle("GET", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		getCallCount++
		return testserver.Response{
			StatusCode: 404,
			Body: map[string]string{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    "App not found.",
			},
		}
	})

	testserver.AddDefaultHandlers(server)

	client, err := sdkclients.New(t.Context(), &config.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	r := (&ResourceApp{}).New(client)
	ctx := t.Context()
	name, _, err := r.DoCreate(ctx, &AppState{App: apps.App{Name: new("test-app")}})

	require.NoError(t, err)
	assert.Equal(t, "test-app", name)
	assert.Equal(t, 2, createCallCount, "expected Create to be called twice")
	assert.Equal(t, 1, getCallCount, "expected Get to be called once to check app state")
}

func TestAppDoUpdate_UpdateMaskHasAllFields(t *testing.T) {
	// iterate over all apps.App fields using reflection and ensure that UpdateMaskFields contains all of them.
	config := GetGeneratedResourceConfig("apps")
	require.NotNil(t, config)
	// forward_user_access_token is not supported, so it's not in UpdateMaskFields.
	nonUpdatableFields := []string{"forward_user_access_token"}
	for _, field := range config.IgnoreRemoteChanges {
		nonUpdatableFields = append(nonUpdatableFields, field.Field.String())
	}

	for _, field := range config.RecreateOnChanges {
		nonUpdatableFields = append(nonUpdatableFields, field.Field.String())
	}

	config = GetResourceConfig("apps")
	require.NotNil(t, config)
	for _, field := range config.IgnoreRemoteChanges {
		nonUpdatableFields = append(nonUpdatableFields, field.Field.String())
	}

	for _, field := range config.RecreateOnChanges {
		nonUpdatableFields = append(nonUpdatableFields, field.Field.String())
	}

	// provided_id_fields recreate on local changes, so they are not updatable either.
	for _, field := range config.ProvidedIDFields {
		nonUpdatableFields = append(nonUpdatableFields, field.Field.String())
	}

	fields := reflect.TypeFor[apps.App]()
	var allFields []string
	for field := range fields.Fields() {
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		jsonTag = strings.TrimSuffix(jsonTag, ",omitempty")
		allFields = append(allFields, jsonTag)
		if !slices.Contains(nonUpdatableFields, jsonTag) {
			assert.Contains(t, UpdateMaskFields, jsonTag, "field %s is not in UpdateMaskFields and not marked as non-updatable", jsonTag)
		}
	}

	for _, field := range UpdateMaskFields {
		assert.Contains(t, allFields, field, "field %s is in UpdateMaskFields but not in apps.App struct", field)
	}
}

// TestAppRequestBody_StripsDeployOnlyFields verifies that appRequestBody removes
// source_code_path and git_source, which DABs applies through the Deploy API, so
// that they never reach the create/update body (a "source_code_path": "" in the
// UpdateApp body caused a 400 in v1.14.0).
func TestAppRequestBody_StripsDeployOnlyFields(t *testing.T) {
	config := &AppState{
		App: apps.App{
			Name:           new("my-app"),
			SourceCodePath: new("/Workspace/Users/me/app"),
			GitSource:      &apps.GitSource{Branch: new("main")},
		},
	}

	body := appRequestBody(config)

	assert.Nil(t, body.SourceCodePath)
	assert.Nil(t, body.GitSource)
	// The config itself is not modified.
	assert.Equal(t, "/Workspace/Users/me/app", *config.SourceCodePath)

	data, err := json.Marshal(body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.NotContains(t, m, "source_code_path", "source_code_path must not appear in the UpdateApp request body")
	assert.Equal(t, "my-app", m["name"])
}
