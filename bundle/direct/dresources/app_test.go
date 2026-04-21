package dresources

import (
	"reflect"
	"slices"
	"strings"
	"testing"

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

var nonUpdatableFields = []string{
	"id",
	"url",
	"updater",
	"create_time",
	"update_time",
	"space",
	"service_principal_name",
	"service_principal_id",
	"service_principal_client_id",
	"oauth2_app_client_id",
	"oauth2_app_integration_id",
	"pending_deployment",
	"active_deployment",
	"app_status",
	"compute_status",
	"creator",
	"default_source_code_path",
	"effective_budget_policy_id",
	"effective_usage_policy_id",
	"effective_user_api_scopes",
	"name",
}

func TestAppDoUpdate_UpdateMaskHasAllFields(t *testing.T) {
	// iterate over all apps.App fields using reflection and ensure that UpdateMaskFields contains all of them.
	app := apps.App{}
	fields := reflect.TypeOf(app)
	var allFields []string
	for i := range fields.NumField() {
		field := fields.Field(i)
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
