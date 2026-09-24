package dresources

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
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
			Name: "test-app",
			ComputeStatus: &apps.ComputeStatus{
				State: apps.ComputeStateActive,
			},
		}
	})

	server.Handle("GET", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		getCallCount++
		return apps.App{
			Name: req.Vars["name"],
			ComputeStatus: &apps.ComputeStatus{
				State: apps.ComputeStateDeleting,
			},
		}
	})

	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	r := (&ResourceApp{}).New(client)
	ctx := t.Context()
	name, _, err := r.DoCreate(ctx, &AppState{App: apps.App{Name: "test-app"}})

	require.NoError(t, err)
	assert.Equal(t, "test-app", name)
	assert.Equal(t, 2, createCallCount, "expected Create to be called twice (1 retry)")
	assert.Equal(t, 1, getCallCount, "expected Get to be called once to check app state")
}

func TestAppDoCreate_RetriesWhenComputeStatusIsUnavailable(t *testing.T) {
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
			Name:          "test-app",
			ComputeStatus: &apps.ComputeStatus{State: apps.ComputeStateActive},
		}
	})
	server.Handle("GET", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		getCallCount++
		return apps.App{Name: req.Vars["name"]}
	})
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	name, _, err := (&ResourceApp{}).New(client).DoCreate(t.Context(), &AppState{App: apps.App{Name: "test-app"}})

	require.NoError(t, err)
	assert.Equal(t, "test-app", name)
	assert.Equal(t, 2, createCallCount)
	assert.Equal(t, 1, getCallCount)
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
			Name: "test-app",
			ComputeStatus: &apps.ComputeStatus{
				State: apps.ComputeStateActive,
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

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	r := (&ResourceApp{}).New(client)
	ctx := t.Context()
	name, _, err := r.DoCreate(ctx, &AppState{App: apps.App{Name: "test-app"}})

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

// TestAppRequestBody_StripsSourceCodePathFromForceSendFields verifies that
// appRequestBody removes SourceCodePath from ForceSendFields even when the plan
// JSON round-trip has forced it in — reproduces the v1.14.0 regression where
// "source_code_path": "" was sent in the UpdateApp body, causing a 400.
func TestAppRequestBody_StripsSourceCodePathFromForceSendFields(t *testing.T) {
	config := &AppState{
		App: apps.App{
			Name:           "my-app",
			SourceCodePath: "/Workspace/Users/me/app",
			// Simulate ForceSendFields as populated by marshal.Unmarshal when the
			// plan entry is deserialized from JSON (ToStructVar calls json.Unmarshal,
			// which calls apps.App.UnmarshalJSON -> marshal.Unmarshal, which adds
			// every basic-type field present in the JSON to ForceSendFields).
			ForceSendFields: []string{"Name", "SourceCodePath"},
		},
	}

	body := appRequestBody(config)

	// SourceCodePath must be cleared and absent from ForceSendFields so that
	// the SDK's marshal.Marshal omits it from the request body.
	assert.Empty(t, body.SourceCodePath)
	assert.NotContains(t, body.ForceSendFields, "SourceCodePath")

	// Verify the field is absent from the marshaled JSON (the root cause of the 400).
	data, err := json.Marshal(body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.NotContains(t, m, "source_code_path", "source_code_path must not appear in the UpdateApp request body")

	// Other ForceSendFields entries must be preserved.
	assert.Contains(t, body.ForceSendFields, "Name")
}

func TestAppUpdateRequestBodyPreservesUnmanagedComputeForDescriptionOnly(t *testing.T) {
	config := &AppState{App: apps.App{
		Name:        "my-app",
		Description: "new description",
	}}
	entry := &PlanEntry{
		RemoteState: &AppRemote{App: apps.App{
			Name:                "my-app",
			Description:         "old description",
			ComputeSize:         apps.ComputeSize("LARGE"),
			ComputeMinInstances: 2,
			ComputeMaxInstances: 7,
		}},
		Changes: deployplan.Changes{
			"description": {Action: deployplan.Update},
		},
	}

	body, err := appUpdateRequestBody(config, entry)

	require.NoError(t, err)
	assert.Equal(t, "new description", body.Description)
	assert.Equal(t, apps.ComputeSize("LARGE"), body.ComputeSize)
	assert.Equal(t, 2, body.ComputeMinInstances)
	assert.Equal(t, 7, body.ComputeMaxInstances)
}

func TestAppUpdateRequestBodySendsEveryMaskedField(t *testing.T) {
	entry := &PlanEntry{
		RemoteState: &AppRemote{App: apps.App{Name: "my-app"}},
		Changes:     deployplan.Changes{"description": {Action: deployplan.Update}},
	}
	config := &AppState{App: apps.App{Name: "my-app", Description: "new description"}}

	body, err := appUpdateRequestBody(config, entry)
	require.NoError(t, err)
	assert.ElementsMatch(t, appUpdateForceSendFields, body.ForceSendFields)
}

func TestAppUpdateRequestBodyOverlaysExplicitComputeChanges(t *testing.T) {
	config := &AppState{App: apps.App{
		Name:                "my-app",
		ComputeSize:         apps.ComputeSize("MEDIUM"),
		ComputeMinInstances: 1,
		ComputeMaxInstances: 3,
	}}
	entry := &PlanEntry{
		RemoteState: &AppRemote{App: apps.App{
			Name:                "my-app",
			ComputeSize:         apps.ComputeSize("LARGE"),
			ComputeMinInstances: 2,
			ComputeMaxInstances: 7,
		}},
		Changes: deployplan.Changes{
			"compute_size":          {Action: deployplan.Update},
			"compute_min_instances": {Action: deployplan.Update},
			"compute_max_instances": {Action: deployplan.Update},
		},
	}

	body, err := appUpdateRequestBody(config, entry)

	require.NoError(t, err)
	assert.Equal(t, apps.ComputeSize("MEDIUM"), body.ComputeSize)
	assert.Equal(t, 1, body.ComputeMinInstances)
	assert.Equal(t, 3, body.ComputeMaxInstances)
}

func TestAppWaitForCreateRetriesUntilComputeStatusIsAvailable(t *testing.T) {
	server := testserver.New(t)
	getCallCount := 0
	server.Handle("GET", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		getCallCount++
		if getCallCount == 1 {
			return apps.App{Name: req.Vars["name"]}
		}
		return apps.App{
			Name:          req.Vars["name"],
			ComputeStatus: &apps.ComputeStatus{State: apps.ComputeStateActive},
		}
	})
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	remote, err := (&ResourceApp{}).New(client).waitForApp(t.Context(), client, "test-app")

	require.NoError(t, err)
	assert.Equal(t, 2, getCallCount)
	assert.Equal(t, apps.ComputeStateActive, remote.ComputeStatus.State)
}
