package tnresources

import (
	"context"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/cli/libs/structwalk"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/stretchr/testify/require"
)

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

	"database_instances": &resources.DatabaseInstance{
		DatabaseInstance: database.DatabaseInstance{
			Name: "mydbinstance",
		},
	},

	"database_catalogs": &resources.DatabaseCatalog{
		DatabaseCatalog: database.DatabaseCatalog{
			Name:                 "mydbcatalog",
			DatabaseInstanceName: "mydbinstance1",
		},
	},

	"synced_database_tables": &resources.SyncedDatabaseTable{
		SyncedDatabaseTable: database.SyncedDatabaseTable{
			Name: "main.myschema.my_synced_table",
		},
	},
}

type prepareWorkspace func(client *databricks.WorkspaceClient) error

// some resource require other resources to exist
var testDeps = map[string]prepareWorkspace{
	"database_catalogs": func(client *databricks.WorkspaceClient) error {
		_, err := client.Database.CreateDatabaseInstance(context.Background(), database.CreateDatabaseInstanceRequest{
			DatabaseInstance: database.DatabaseInstance{
				Name: "mydbinstance1",
			},
		})
		return err
	},
}

func TestAll(t *testing.T) {
	_, client := setupTestServerClient(t)

	for group, resource := range SupportedResources {
		t.Run(group, func(t *testing.T) {
			adapter, err := NewAdapter(resource, client)
			require.NoError(t, err)
			require.NotNil(t, adapter)

			testCRUD(t, group, adapter, client)
		})
	}

	m, err := InitAll(client)
	require.NoError(t, err)
	require.Len(t, m, len(SupportedResources))
}

func testCRUD(t *testing.T, group string, adapter *Adapter, client *databricks.WorkspaceClient) {
	prepDeps, hasDeps := testDeps[group]
	if hasDeps {
		require.NoError(t, prepDeps(client))
	}

	var inputConfig any
	inputConfig, ok := testConfig[group]

	if ok {
		require.Equal(t, adapter.InputConfigType(), reflect.TypeOf(inputConfig))
	} else {
		inputConfig = reflect.New(adapter.InputConfigType().Elem()).Interface()
	}

	config, err := adapter.PrepareConfig(inputConfig)
	require.NoError(t, err, "PrepareConfig failed")

	ctx := context.Background()

	createdID, _, err := adapter.DoCreate(ctx, config)
	require.NoError(t, err, "DoCreate failed config=%v", config)
	require.NotEmpty(t, createdID, "ID returned from DoCreate was empty")

	_, err = adapter.DoUpdate(ctx, createdID, config)
	require.NoError(t, err, "DoUpdate failed")
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

// TestRecreateFields validates that all fields in RecreateFields
// exist in the corresponding ConfigType for each resource.
func TestRecreateFields(t *testing.T) {
	for resourceName, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, nil)
		require.NoError(t, err)

		t.Run(resourceName, func(t *testing.T) {
			validateFields(t, adapter.InputConfigType(), adapter.recreateFields)
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
