package terranova

import (
	"context"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/cli/libs/structwalk"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJobResource(t *testing.T) {
	client := &databricks.WorkspaceClient{}

	cfg := &resources.Job{
		JobSettings: jobs.JobSettings{
			Name: "test-job",
		},
	}

	res, cfgType, err := New(client, "jobs", "test-job", cfg)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Ensure we received the correct resource type.
	require.IsType(t, &tnresources.ResourceJob{}, res)
	require.IsType(t, reflect.TypeOf(jobs.JobSettings{}), cfgType)

	// The underlying config should match what we passed in.
	r := res.(*tnresources.ResourceJob)
	require.Equal(t, cfg.JobSettings, r.Config())
}

// validateFields uses structwalk to generate all valid field paths and checks membership.
func validateFields(t *testing.T, configType reflect.Type, fields map[string]struct{}) {
	validPaths := make(map[string]struct{})

	err := structwalk.WalkType(configType, func(path *structpath.PathNode, typ reflect.Type) bool {
		validPaths[path.String()] = struct{}{}
		return true // continue walking
	})
	require.NoError(t, err)

	for fieldPath := range fields {
		if _, exists := validPaths[fieldPath]; !exists {
			t.Errorf("invalid field '%s' for %s", fieldPath, configType)
		}
	}
}

// TestRecreateFieldsValidation validates that all fields in RecreateFields
// exist in the corresponding ConfigType for each resource.
func TestRecreateFieldsValidation(t *testing.T) {
	for resourceName, settings := range SupportedResources {
		if len(settings.RecreateFields) == 0 {
			continue
		}
		t.Run(resourceName, func(t *testing.T) {
			validateFields(t, settings.ConfigType, settings.RecreateFields)
		})
	}
}

func setupTestServerClient(t *testing.T) (*testserver.Server, *databricks.WorkspaceClient) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "testtoken")
	client, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)
	return server, client
}

// Tests CRUD calls against testserver
func TestCRUD(t *testing.T) {
	for group, settings := range SupportedResources {
		t.Run(group, func(t *testing.T) {
			testCRUD(t, group, settings)
		})
	}
}

var testConfig map[string]any = map[string]any{
	"apps": &resources.App{
		App: apps.App{
			Name: "myapp",
		},
	},

	"schemas": &resources.Schema{
		CreateSchema: catalog.CreateSchema{
			CatalogName: "main",
			Name:        "myschema",
		},
	},

	"volumes": &resources.Volume{
		CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
			CatalogName: "main",
			SchemaName:  "myschema",
			Name:        "myvolume",
		},
	},
}

func testCRUD(t *testing.T, group string, settings ResourceSettings) {
	initConfig, ok := testConfig[group]
	if !ok {
		ft := settings.New.Type()
		expectedCfgType := ft.In(1).Elem()
		initConfig = reflect.New(expectedCfgType).Interface()
	}

	_, client := setupTestServerClient(t)

	resource, err := invokeConstructor(settings.New, client, initConfig)
	require.NoError(t, err)

	typeCheckNoRemoteState(t, resource, settings)

	ctx := context.Background()

	// using integer because jobs will parse it
	// might need to add sample id to settings if there are more constraints like that in the future
	myid := "1234"
	err = resource.DoRefresh(ctx, myid)
	require.Error(t, err, "initial DoRefresh should fail because resource does not exist")

	modifierBasic, hasBasic := resource.(IResourceBasic)
	modifierRefresh, hasWithRefresh := resource.(IResourceWithRefresh)

	var createdID string

	if hasBasic {
		t.Log("Calling DoCreate")
		assert.False(t, hasWithRefresh, "resource must implement either IResourceBasic or IResourceWithRefresh but not both")
		createdID, err = modifierBasic.DoCreate(ctx)
	} else {
		t.Log("Calling DoCreateWithRefresh")
		assert.True(t, hasWithRefresh, "resource must implement exactly one of: IResourceBasic or IResourceWithRefresh")
		createdID, err = modifierRefresh.DoCreateWithRefresh(ctx)
	}
	require.NoError(t, err)
	require.NotEmpty(t, createdID)

	if hasBasic {
		typeCheckNoRemoteState(t, resource, settings)
	} else {
		typeCheckWithRemoteState(t, resource, settings)
	}

	// Now state can be refreshed:
	require.NoError(t, resource.DoRefresh(ctx, createdID))
	typeCheckWithRemoteState(t, resource, settings)

	// TODO: test update and wait
}

func typeCheckNoRemoteState(t *testing.T, resource IResource, settings ResourceSettings) {
	require.NotNil(t, resource.Config())
	assert.Equal(t, settings.ConfigType, reflect.TypeOf(resource.Config()))
	require.Nil(t, resource.RemoteState())
}

func typeCheckWithRemoteState(t *testing.T, resource IResource, settings ResourceSettings) {
	require.NotNil(t, resource.Config())
	assert.Equal(t, settings.ConfigType, reflect.TypeOf(resource.Config()))

	require.NotNil(t, resource.RemoteState())
	assert.Equal(t, settings.RemoteType, reflect.TypeOf(resource.RemoteState()))
}
