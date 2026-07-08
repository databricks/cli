package dresources

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
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
	var nonUpdatableFields []string
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

func TestAppOverrideChangeDescActiveDeployment(t *testing.T) {
	r := &ResourceApp{}

	// The hook skips drift on the deploy-only fields (source_code_path, config,
	// git_source) when the app has no active deployment. remote may be a typed
	// nil (--plan-mode=local or resource missing remotely); the hook must treat
	// that as "no active deployment" without dereferencing.
	tests := []struct {
		name       string
		path       *structpath.PathNode
		remote     *AppRemote
		wantAction deployplan.ActionType
	}{
		{"source_code_path skips when remote is nil", structpath.MustParsePath("source_code_path"), nil, deployplan.Skip},
		{"config.command skips when remote is nil", structpath.MustParsePath("config.command"), nil, deployplan.Skip},
		{"git_source skips when remote is nil", structpath.MustParsePath("git_source"), nil, deployplan.Skip},
		{"source_code_path skips when ActiveDeployment is nil", structpath.MustParsePath("source_code_path"), &AppRemote{}, deployplan.Skip},
		{"source_code_path untouched when ActiveDeployment is set", structpath.MustParsePath("source_code_path"), &AppRemote{App: apps.App{ActiveDeployment: &apps.AppDeployment{}}}, deployplan.Update},
		{"name untouched (not a deploy-only field)", structpath.MustParsePath("name"), nil, deployplan.Update},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			change := &ChangeDesc{Action: deployplan.Update, Old: "a", New: "b"}
			require.NoError(t, r.OverrideChangeDesc(t.Context(), tc.path, change, tc.remote))
			assert.Equal(t, tc.wantAction, change.Action)
		})
	}
}
