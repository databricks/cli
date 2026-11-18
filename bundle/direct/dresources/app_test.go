package dresources

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/retry"
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

// TestAppRequestBody_StripsEmptySourceCodePathFromForceSendFields verifies that an empty
// SourceCodePath is removed from ForceSendFields even when the plan JSON round-trip has
// forced it in — reproduces the v1.14.0 regression (#6401) where "source_code_path": ""
// was sent in the UpdateApp body, causing a 400.
func TestAppRequestBody_StripsEmptySourceCodePathFromForceSendFields(t *testing.T) {
	app := apps.App{
		Name: "my-app",
		// Simulate ForceSendFields as populated by marshal.Unmarshal when the
		// plan entry is deserialized from JSON (ToStructVar calls json.Unmarshal,
		// which calls apps.App.UnmarshalJSON -> marshal.Unmarshal, which adds
		// every basic-type field present in the JSON to ForceSendFields).
		ForceSendFields: []string{"Name", "SourceCodePath"},
	}

	body := appRequestBody(app)

	// Absent from ForceSendFields so that the SDK's marshal.Marshal omits it from the request body.
	assert.NotContains(t, body.ForceSendFields, "SourceCodePath")

	// Verify the field is absent from the marshaled JSON (the root cause of the 400).
	assert.NotContains(t, marshalFields(t, body), "source_code_path", "source_code_path must not appear in the request body when empty")

	// Other ForceSendFields entries must be preserved.
	assert.Contains(t, body.ForceSendFields, "Name")
	// The plan entry itself must not be modified.
	assert.Contains(t, app.ForceSendFields, "SourceCodePath")
}

func TestAppRequestBody_SendsNonEmptySourceCodePath(t *testing.T) {
	app := apps.App{
		Name:            "my-app",
		SourceCodePath:  "/Workspace/Users/me/app",
		ForceSendFields: []string{"SourceCodePath"},
	}

	body := appRequestBody(app)

	assert.Equal(t, "/Workspace/Users/me/app", body.SourceCodePath)
	assert.Contains(t, body.ForceSendFields, "SourceCodePath")
}

// TestAppUpdateBody_OmitsSourceCodePath verifies that the UpdateApp body never carries
// source_code_path, even when the plan entry has a value and ForceSendFields lists it,
// while git_source is kept. Production rejects source_code_path on UpdateApp.
func TestAppUpdateBody_OmitsSourceCodePath(t *testing.T) {
	config := &AppState{}
	config.App = apps.App{
		Name:            "my-app",
		SourceCodePath:  "/Workspace/Users/me/app",
		GitSource:       &apps.GitSource{Branch: "main"},
		ForceSendFields: []string{"Name", "SourceCodePath"},
	}

	body := appUpdateBody(config)

	fields := marshalFields(t, body)
	assert.NotContains(t, fields, "source_code_path")
	assert.Contains(t, fields, "git_source")
	assert.NotContains(t, body.ForceSendFields, "SourceCodePath")
	// The plan entry itself must not be modified.
	assert.Equal(t, "/Workspace/Users/me/app", config.SourceCodePath)
	assert.Contains(t, config.ForceSendFields, "SourceCodePath")
}

func TestAppUpdateMaskOmitsSourceCodePath(t *testing.T) {
	assert.NotContains(t, UpdateMaskFields, "source_code_path")
	assert.Contains(t, UpdateMaskFields, "git_source")
}

// marshalFields returns the JSON field names the SDK marshaler emits for app.
func marshalFields(t *testing.T, app apps.App) map[string]any {
	data, err := json.Marshal(app)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
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

const (
	appComputeUpdatingDeleteMessage = "Cannot delete app test-app as it is not terminal with state UPDATING, " +
		"and was updated less than 20 minutes ago. Please wait before trying again."
	appComputeUpdatingUpdateMessage = "Cannot update app test-app as its compute is in UPDATING state. " +
		"App compute needs to be ACTIVE or STOPPED to update."
)

func appBadRequest(message string) testserver.Response {
	return testserver.Response{
		StatusCode: 400,
		Body:       map[string]string{"error_code": "BAD_REQUEST", "message": message},
	}
}

// fakeAppCompute serves GET /api/2.0/apps/{name} and the start/stop endpoints. GET returns the
// states in transitional first, then settled; start and stop change settled. A nil settled makes
// GET return 404 once transitional is used up.
type fakeAppCompute struct {
	transitional []apps.ComputeState
	settled      *apps.ComputeState
	gets         int
	starts       int
	stops        int
}

func (f *fakeAppCompute) register(t *testing.T, server *testserver.Server) {
	t.Setenv(retry.IntervalMsVariable, "1")
	server.Handle("GET", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		f.gets++
		state := f.settled
		if f.gets <= len(f.transitional) {
			state = &f.transitional[f.gets-1]
		}
		if state == nil {
			return testserver.Response{
				StatusCode: 404,
				Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "App not found."},
			}
		}
		return apps.App{Name: req.Vars["name"], ComputeStatus: &apps.ComputeStatus{State: *state}}
	})
	server.Handle("POST", "/api/2.0/apps/{name}/start", func(req testserver.Request) any {
		f.starts++
		f.settled = new(apps.ComputeStateActive)
		return apps.App{Name: req.Vars["name"], ComputeStatus: &apps.ComputeStatus{State: apps.ComputeStateActive}}
	})
	server.Handle("POST", "/api/2.0/apps/{name}/stop", func(req testserver.Request) any {
		f.stops++
		f.settled = new(apps.ComputeStateStopped)
		return apps.App{Name: req.Vars["name"], ComputeStatus: &apps.ComputeStatus{State: apps.ComputeStateStopped}}
	})
}

func newAppForServer(t *testing.T, server *testserver.Server) *ResourceApp {
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	return (&ResourceApp{}).New(client)
}

func TestAppDoDelete_RetriesWhileComputeSettles(t *testing.T) {
	tests := []struct {
		name    string
		compute fakeAppCompute
		// deleteMessages are the 400 messages DELETE returns in order; later calls succeed.
		deleteMessages []string
		wantDeletes    int
		wantGets       int
		wantErr        string
	}{
		{
			name:           "retries after compute leaves UPDATING",
			compute:        fakeAppCompute{transitional: []apps.ComputeState{apps.ComputeStateUpdating}, settled: new(apps.ComputeStateActive)},
			deleteMessages: []string{appComputeUpdatingDeleteMessage},
			wantDeletes:    2,
			wantGets:       2,
		},
		{
			name:           "retries when compute settled before the first read",
			compute:        fakeAppCompute{settled: new(apps.ComputeStateActive)},
			deleteMessages: []string{appComputeUpdatingDeleteMessage},
			wantDeletes:    2,
			wantGets:       1,
		},
		{
			name:           "treats an app gone while waiting as deleted",
			compute:        fakeAppCompute{},
			deleteMessages: []string{appComputeUpdatingDeleteMessage},
			wantDeletes:    1,
			wantGets:       1,
		},
		{
			name:           "returns the last 400 when retries run out",
			compute:        fakeAppCompute{settled: new(apps.ComputeStateActive)},
			deleteMessages: slices.Repeat([]string{appComputeUpdatingDeleteMessage}, retry.MaxRetries+1),
			wantDeletes:    retry.MaxRetries + 1,
			wantGets:       retry.MaxRetries,
			wantErr:        "not terminal with state UPDATING",
		},
		{
			name:           "does not retry a delete of a DELETING app",
			compute:        fakeAppCompute{settled: new(apps.ComputeStateDeleting)},
			deleteMessages: []string{"Cannot delete app test-app as it is not terminal with state DELETING, and was updated less than 20 minutes ago. Please wait before trying again."},
			wantDeletes:    1,
			wantErr:        "state DELETING",
		},
		{
			name:           "does not retry other bad requests",
			compute:        fakeAppCompute{settled: new(apps.ComputeStateUpdating)},
			deleteMessages: []string{"Invalid app name."},
			wantDeletes:    1,
			wantErr:        "Invalid app name.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := testserver.New(t)
			compute := tc.compute
			compute.register(t, server)
			deletes := 0
			server.Handle("DELETE", "/api/2.0/apps/{name}", func(req testserver.Request) any {
				deletes++
				if deletes <= len(tc.deleteMessages) {
					return appBadRequest(tc.deleteMessages[deletes-1])
				}
				return apps.App{Name: req.Vars["name"]}
			})
			r := newAppForServer(t, server)

			err := r.DoDelete(t.Context(), "test-app", nil)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantDeletes, deletes)
			assert.Equal(t, tc.wantGets, compute.gets)
		})
	}
}

// TestAppDoDelete_WaitFailureKeepsTheBadRequest checks that a failed wait does not change how
// the apply layer classifies the error: it stays the 400, not the GET's 403.
func TestAppDoDelete_WaitFailureKeepsTheBadRequest(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: 403,
			Body:       map[string]string{"error_code": "PERMISSION_DENIED", "message": "no access"},
		}
	})
	deletes := 0
	server.Handle("DELETE", "/api/2.0/apps/{name}", func(req testserver.Request) any {
		deletes++
		return appBadRequest(appComputeUpdatingDeleteMessage)
	})
	r := newAppForServer(t, server)

	err := r.DoDelete(t.Context(), "test-app", nil)
	require.ErrorContains(t, err, "not terminal with state UPDATING")
	require.ErrorContains(t, err, "waiting for the app compute to settle failed: no access")
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	require.True(t, ok)
	assert.Equal(t, 400, apiErr.StatusCode)
	assert.NotErrorIs(t, err, apierr.ErrPermissionDenied)
	assert.Equal(t, 1, deletes)
}

func TestAppDoUpdate_RetriesWhileComputeSettles(t *testing.T) {
	tests := []struct {
		name string
		// planStarted is the compute started state recorded at plan time.
		planStarted bool
		// lifecycleStarted is lifecycle.started in the config; nil means not set.
		lifecycleStarted *bool
		compute          fakeAppCompute
		wantUpdates      int
		wantStarts       int
		wantStops        int
		wantDeploys      int
		wantMissing      bool
	}{
		{
			name:        "retries after compute leaves UPDATING",
			planStarted: true,
			compute:     fakeAppCompute{transitional: []apps.ComputeState{apps.ComputeStateUpdating}, settled: new(apps.ComputeStateActive)},
			wantUpdates: 2,
		},
		{
			name:             "starts an app that settled to STOPPED",
			planStarted:      true,
			lifecycleStarted: new(true),
			compute:          fakeAppCompute{transitional: []apps.ComputeState{apps.ComputeStateStopping}, settled: new(apps.ComputeStateStopped)},
			wantUpdates:      2,
			wantStarts:       1,
			wantDeploys:      1,
		},
		{
			name:             "does not stop an app that settled to STOPPED",
			planStarted:      true,
			lifecycleStarted: new(false),
			compute:          fakeAppCompute{transitional: []apps.ComputeState{apps.ComputeStateStopping}, settled: new(apps.ComputeStateStopped)},
			wantUpdates:      2,
		},
		{
			name:             "does not start an app that settled to ACTIVE",
			planStarted:      false,
			lifecycleStarted: new(true),
			compute:          fakeAppCompute{transitional: []apps.ComputeState{apps.ComputeStateStarting}, settled: new(apps.ComputeStateActive)},
			wantUpdates:      2,
			wantDeploys:      1,
		},
		{
			name:        "fails with not found when the app is gone while waiting",
			planStarted: true,
			compute:     fakeAppCompute{},
			wantUpdates: 1,
			wantMissing: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := testserver.New(t)
			compute := tc.compute
			compute.register(t, server)
			updates := 0
			server.Handle("POST", "/api/2.0/apps/{name}/update", func(req testserver.Request) any {
				updates++
				if updates == 1 {
					return appBadRequest(appComputeUpdatingUpdateMessage)
				}
				return apps.AppUpdate{Status: &apps.AppUpdateUpdateStatus{State: apps.AppUpdateUpdateStatusUpdateStateSucceeded}}
			})
			server.Handle("GET", "/api/2.0/apps/{name}/update", func(req testserver.Request) any {
				return apps.AppUpdate{Status: &apps.AppUpdateUpdateStatus{State: apps.AppUpdateUpdateStatusUpdateStateSucceeded}}
			})
			deploys := 0
			succeeded := apps.AppDeployment{
				DeploymentId: "deploy-1",
				Status:       &apps.AppDeploymentStatus{State: apps.AppDeploymentStateSucceeded},
			}
			server.Handle("POST", "/api/2.0/apps/{name}/deployments", func(req testserver.Request) any {
				deploys++
				return succeeded
			})
			server.Handle("GET", "/api/2.0/apps/{name}/deployments/{deployment_id}", func(req testserver.Request) any {
				return succeeded
			})
			r := newAppForServer(t, server)

			config := &AppState{Name: "test-app", Description: "new", SourceCodePath: "/Workspace/app"}
			if tc.lifecycleStarted != nil {
				config.Lifecycle = &StateLifecycle{Started: tc.lifecycleStarted}
			}
			entry := &PlanEntry{
				Changes:     deployplan.Changes{"description": {Action: deployplan.Update}},
				RemoteState: &AppRemote{Lifecycle: &StateLifecycle{Started: new(tc.planStarted)}},
			}
			_, err := r.DoUpdate(t.Context(), "test-app", config, entry)
			if tc.wantMissing {
				require.ErrorContains(t, err, "app is gone")
				assert.True(t, apierr.IsMissing(err), "want a not-found error, got %v", err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantUpdates, updates)
			assert.Equal(t, tc.wantStarts, compute.starts)
			assert.Equal(t, tc.wantStops, compute.stops)
			assert.Equal(t, tc.wantDeploys, deploys)
		})
	}
}

// TestComputeUpdatingRetryMessage_SuppressedInAcceptanceTests guards the wording: the warning
// only appears when a cloud run hits the race, so it must match SuppressLines.warn_retry.
func TestComputeUpdatingRetryMessage_SuppressedInAcceptanceTests(t *testing.T) {
	raw, err := os.ReadFile("../../../acceptance/test.toml")
	require.NoError(t, err)
	// The checkout may have CRLF line endings (Windows).
	m := regexp.MustCompile(`(?m)^SuppressLines\.warn_retry = '(.*)'\r?$`).FindSubmatch(raw)
	require.NotNil(t, m, "SuppressLines.warn_retry not found in acceptance/test.toml")
	pattern := regexp.MustCompile(string(m[1]))
	assert.Regexp(t, pattern, "Warn: "+computeUpdatingRetryMessage("test-app"))
}
