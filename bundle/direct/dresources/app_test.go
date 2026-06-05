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
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	ctx := t.Context()

	// Create then delete an app to put it in DELETING state.
	// The testserver's DELETE is asynchronous: it sets DELETING rather than
	// removing immediately, so the retry create will find the app in that state.
	_, err = client.Apps.Create(ctx, apps.CreateAppRequest{App: apps.App{Name: "test-app"}})
	require.NoError(t, err)
	_, err = client.Apps.DeleteByName(ctx, "test-app")
	require.NoError(t, err)

	r := (&ResourceApp{}).New(client)
	name, _, err := r.DoCreate(ctx, NewNopStateSaver(reflect.TypeFor[*AppState]()), &AppState{App: apps.App{Name: "test-app"}})

	require.NoError(t, err)
	assert.Equal(t, "test-app", name)
}

// TestAppDoCreate_RetriesWhenGetReturnsNotFound verifies that DoCreate retries
// when the app was just deleted between the create call and the get call.
func TestAppDoCreate_RetriesWhenGetReturnsNotFound(t *testing.T) {
	server := testserver.New(t)

	// Simulate a race: the app existed when Create was called (returns 409) but
	// was deleted before the existence check (GET returns 404). The first POST
	// returns 409 without storing anything so the standard GET handler returns
	// 404 naturally, and the retry POST creates the app normally.
	rejectedOnce := false
	server.Handle("POST", "/api/2.0/apps", func(req testserver.Request) any {
		if !rejectedOnce {
			rejectedOnce = true
			return testserver.Response{
				StatusCode: 409,
				Body:       map[string]string{"error_code": "RESOURCE_ALREADY_EXISTS", "message": "An app with the same name already exists."},
			}
		}
		return req.Workspace.AppsUpsert(req, "")
	})

	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	r := (&ResourceApp{}).New(client)
	ctx := t.Context()
	name, _, err := r.DoCreate(ctx, NewNopStateSaver(reflect.TypeFor[*AppState]()), &AppState{App: apps.App{Name: "test-app"}})

	require.NoError(t, err)
	assert.Equal(t, "test-app", name)
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
