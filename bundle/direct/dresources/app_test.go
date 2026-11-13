package dresources

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
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
	name, _, err := r.DoCreate(ctx, &AppState{Name: "test-app"})

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
	name, _, err := r.DoCreate(ctx, &AppState{Name: "test-app"})

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

func TestAppUpdateMask_ComputeSizeFollowsBody(t *testing.T) {
	tests := []struct {
		name        string
		computeSize apps.ComputeSize
		wantCompute bool
	}{
		{name: "declared", computeSize: apps.ComputeSizeLarge, wantCompute: true},
		{name: "undeclared", computeSize: "", wantCompute: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mask := strings.Split(appUpdateMask(&apps.App{ComputeSize: tt.computeSize}), ",")
			assert.Equal(t, tt.wantCompute, slices.Contains(mask, "compute_size"))
			assert.Contains(t, mask, "description")
			assert.Contains(t, mask, "compute_min_instances")
		})
	}
}

// TestAppRequestBody_StripsSourceCodePathFromForceSendFields verifies that
// appRequestBody removes SourceCodePath from ForceSendFields even when the plan
// JSON round-trip has forced it in — reproduces the v1.14.0 regression where
// "source_code_path": "" was sent in the UpdateApp body, causing a 400.
func TestAppRequestBody_StripsSourceCodePathFromForceSendFields(t *testing.T) {
	config := &AppState{
		Name:           "my-app",
		SourceCodePath: "/Workspace/Users/me/app",
		// Simulate ForceSendFields as populated by marshal.Unmarshal when the
		// plan entry is deserialized from JSON (ToStructVar calls json.Unmarshal,
		// which calls apps.App.UnmarshalJSON -> marshal.Unmarshal, which adds
		// every basic-type field present in the JSON to ForceSendFields).
		ForceSendFields: []string{"Name", "SourceCodePath"},
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

func TestWithMissingServicePrincipalHint(t *testing.T) {
	assert.NoError(t, withMissingServicePrincipalHint(nil, "myapp"))

	other := errors.New("some other error")
	assert.Same(t, other, withMissingServicePrincipalHint(other, "myapp"))

	apiErr := &apierr.APIError{
		StatusCode: 400,
		ErrorCode:  "BAD_REQUEST",
		Message:    "App: myapp's service principal does not exist or is deleted. App cannot run without a valid service principal. Please delete your app.",
	}
	computeErr := errors.New("failed to reach ACTIVE, got ERROR: App service principal does not exist or is deleted, application cannot run without a valid service principal. Please delete the app and re-create it.")
	capitalizedErr := errors.New("Service principal does not exist or is deleted")

	for _, err := range []error{apiErr, fmt.Errorf("wrapped: %w", apiErr), computeErr, capitalizedErr} {
		got := withMissingServicePrincipalHint(err, "myapp")
		assert.ErrorIs(t, got, err)
		assert.Contains(t, got.Error(), `delete it with "databricks apps delete myapp"`)
	}
}

func TestAppDoDelete_RetriesWhileComputeSettles(t *testing.T) {
	tests := []struct {
		name string
		// computeStates are the states GET returns in order; the last one repeats.
		computeStates []apps.ComputeState
		wantDeletes   int
		wantErr       bool
	}{
		{
			name:          "retries after compute leaves UPDATING",
			computeStates: []apps.ComputeState{apps.ComputeStateUpdating, apps.ComputeStateActive},
			wantDeletes:   2,
		},
		{
			name:          "fails at once when compute is not transitional",
			computeStates: []apps.ComputeState{apps.ComputeStateActive},
			wantDeletes:   1,
			wantErr:       true,
		},
		{
			name:          "fails at once when app is already deleting",
			computeStates: []apps.ComputeState{apps.ComputeStateDeleting},
			wantDeletes:   1,
			wantErr:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := testserver.New(t)
			deletes := 0
			gets := 0

			server.Handle("DELETE", "/api/2.0/apps/{name}", func(req testserver.Request) any {
				deletes++
				if deletes == 1 {
					return testserver.Response{
						StatusCode: 400,
						Body: map[string]string{
							"error_code": "BAD_REQUEST",
							"message":    "Cannot delete app test-app as it is not terminal with state UPDATING, and was updated less than 20 minutes ago. Please wait before trying again.",
						},
					}
				}
				return apps.App{Name: req.Vars["name"]}
			})
			server.Handle("GET", "/api/2.0/apps/{name}", func(req testserver.Request) any {
				state := tc.computeStates[min(gets, len(tc.computeStates)-1)]
				gets++
				return apps.App{Name: req.Vars["name"], ComputeStatus: &apps.ComputeStatus{State: state}}
			})
			testserver.AddDefaultHandlers(server)

			client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
			require.NoError(t, err)

			r := (&ResourceApp{}).New(client)
			err = r.DoDelete(t.Context(), "test-app", nil)
			if tc.wantErr {
				require.ErrorContains(t, err, "not terminal with state UPDATING")
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantDeletes, deletes)
		})
	}
}

func TestAppRetryWhileComputeSettles_IgnoresNonBadRequest(t *testing.T) {
	server := testserver.New(t)
	gets := 0
	server.Handle("GET", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		gets++
		return apps.App{Name: req.Vars["name"], ComputeStatus: &apps.ComputeStatus{State: apps.ComputeStateUpdating}}
	})
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	r := (&ResourceApp{}).New(client)
	calls := 0
	err = r.retryWhileComputeSettles(t.Context(), "test-app", func() error {
		calls++
		return &apierr.APIError{StatusCode: 500, ErrorCode: "INTERNAL_ERROR", Message: "boom"}
	})
	require.ErrorContains(t, err, "boom")
	assert.Equal(t, 1, calls)
	assert.Equal(t, 0, gets)
}
