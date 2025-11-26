package dresources

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
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

	"synced_database_tables": &resources.SyncedDatabaseTable{
		SyncedDatabaseTable: database.SyncedDatabaseTable{
			Name: "main.myschema.my_synced_table",
		},
	},

	"registered_models": &resources.RegisteredModel{
		CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{
			Name:            "my_registered_model",
			Comment:         "Test registered model",
			CatalogName:     "main",
			SchemaName:      "default",
			StorageLocation: "s3://my-bucket/my-path",
		},
	},

	"experiments": &resources.MlflowExperiment{
		CreateExperiment: ml.CreateExperiment{
			Name: "my-experiment",
			Tags: []ml.ExperimentTag{
				{
					Key:   "my-tag",
					Value: "my-value",
				},
			},
			ArtifactLocation: "s3://my-bucket/my-experiment",
		},
	},

	"models": &resources.MlflowModel{
		CreateModelRequest: ml.CreateModelRequest{
			Name:        "my_mlflow_model",
			Description: "my_mlflow_model_description",
			Tags: []ml.ModelTag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
		},
	},

	"model_serving_endpoints": &resources.ModelServingEndpoint{
		CreateServingEndpoint: serving.CreateServingEndpoint{
			Name: "my-endpoint",
			Config: &serving.EndpointCoreConfigInput{
				Name: "my-endpoint",
				AutoCaptureConfig: &serving.AutoCaptureConfigInput{
					CatalogName:     "main",
					SchemaName:      "myschema",
					TableNamePrefix: "my_table",
					Enabled:         true,
					ForceSendFields: nil,
				},
				ServedModels: nil,
				ServedEntities: []serving.ServedEntityInput{
					{
						EntityName:                "entity-name",
						EntityVersion:             "1",
						WorkloadSize:              "Small",
						ScaleToZeroEnabled:        true,
						WorkloadType:              serving.ServingModelWorkloadTypeCpu,
						EnvironmentVars:           map[string]string{"key": "value"},
						InstanceProfileArn:        "arn:aws:iam::123456789012:instance-profile/my-instance-profile",
						MaxProvisionedConcurrency: 10,
						MaxProvisionedThroughput:  100,
						MinProvisionedConcurrency: 1,
						MinProvisionedThroughput:  10,
						Name:                      "entity-name",
						ProvisionedModelUnits:     100,
						ExternalModel:             nil,
						ForceSendFields:           nil,
					},
				},
				TrafficConfig: &serving.TrafficConfig{
					Routes: []serving.Route{
						{
							ServedModelName:   "model-name-1",
							TrafficPercentage: 100,
						},
					},
				},
			},
		},
	},
}

type prepareWorkspace func(client *databricks.WorkspaceClient) (any, error)

// some resource require other resources to exist
var testDeps = map[string]prepareWorkspace{
	"database_catalogs": func(client *databricks.WorkspaceClient) (any, error) {
		_, err := client.Database.CreateDatabaseInstance(context.Background(), database.CreateDatabaseInstanceRequest{
			DatabaseInstance: database.DatabaseInstance{
				Name: "mydbinstance1",
			},
		})

		return &resources.DatabaseCatalog{
			DatabaseCatalog: database.DatabaseCatalog{
				Name:                 "mydbcatalog",
				DatabaseInstanceName: "mydbinstance1",
			},
		}, err
	},

	"jobs.permissions": func(client *databricks.WorkspaceClient) (any, error) {
		resp, err := client.Jobs.Create(context.Background(), jobs.CreateJob{
			Name: "job-permissions",
			Tasks: []jobs.Task{
				{
					TaskKey: "t",
					NotebookTask: &jobs.NotebookTask{
						NotebookPath: "/Workspace/Users/user@example.com/notebook",
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/jobs/" + strconv.FormatInt(resp.JobId, 10),
			Permissions: []iam.AccessControlRequest{{
				PermissionLevel: "IS_OWNER",
				UserName:        "user@example.com",
			}},
		}, nil
	},

	"pipelines.permissions": func(client *databricks.WorkspaceClient) (any, error) {
		resp, err := client.Pipelines.Create(context.Background(), pipelines.CreatePipeline{
			Name: "pipeline-permissions",
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/pipelines/" + resp.PipelineId,
			Permissions: []iam.AccessControlRequest{{
				PermissionLevel: "CAN_MANAGE",
				UserName:        "user@example.com",
			}},
		}, nil
	},

	"models.permissions": func(client *databricks.WorkspaceClient) (any, error) {
		resp, err := client.ModelRegistry.CreateModel(context.Background(), ml.CreateModelRequest{
			Name:        "model-permissions",
			Description: "model for permissions testing",
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/registered-models/" + resp.RegisteredModel.Name,
			Permissions: []iam.AccessControlRequest{{
				PermissionLevel: "CAN_MANAGE",
				UserName:        "user@example.com",
			}},
		}, nil
	},

	"experiments.permissions": func(client *databricks.WorkspaceClient) (any, error) {
		resp, err := client.Experiments.CreateExperiment(context.Background(), ml.CreateExperiment{
			Name: "experiment-permissions",
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/experiments/" + resp.ExperimentId,
			Permissions: []iam.AccessControlRequest{{
				PermissionLevel: "CAN_MANAGE",
				UserName:        "user@example.com",
			}},
		}, nil
	},

	"clusters.permissions": func(client *databricks.WorkspaceClient) (any, error) {
		return &PermissionsState{
			ObjectID: "/clusters/cluster-permissions",
			Permissions: []iam.AccessControlRequest{{
				PermissionLevel: "CAN_MANAGE",
				UserName:        "user@example.com",
			}},
		}, nil
	},

	"apps.permissions": func(client *databricks.WorkspaceClient) (any, error) {
		waiter, err := client.Apps.Create(context.Background(), apps.CreateAppRequest{
			App: apps.App{
				Name: "app-permissions",
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/apps/" + waiter.Response.Name,
			Permissions: []iam.AccessControlRequest{{
				PermissionLevel: "CAN_MANAGE",
				UserName:        "user@example.com",
			}},
		}, nil
	},

	"sql_warehouses.permissions": func(client *databricks.WorkspaceClient) (any, error) {
		return &PermissionsState{
			ObjectID: "/sql/warehouses/warehouse-permissions",
			Permissions: []iam.AccessControlRequest{{
				PermissionLevel: "CAN_MANAGE",
				UserName:        "user@example.com",
			}},
		}, nil
	},

	"database_instances.permissions": func(client *databricks.WorkspaceClient) (any, error) {
		waiter, err := client.Database.CreateDatabaseInstance(context.Background(), database.CreateDatabaseInstanceRequest{
			DatabaseInstance: database.DatabaseInstance{
				Name: "dbinstance-permissions",
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/database-instances/" + waiter.Response.Name,
			Permissions: []iam.AccessControlRequest{{
				PermissionLevel: "CAN_MANAGE",
				UserName:        "user@example.com",
			}},
		}, nil
	},

	"dashboards.permissions": func(client *databricks.WorkspaceClient) (any, error) {
		ctx := context.Background()
		parentPath := "/Workspace/Users/user@example.com"

		// Create parent directory if it doesn't exist
		err := client.Workspace.MkdirsByPath(ctx, parentPath)
		if err != nil {
			return nil, err
		}

		resp, err := client.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{
			Dashboard: dashboards.Dashboard{
				DisplayName:         "dashboard-permissions",
				ParentPath:          parentPath,
				SerializedDashboard: `{"pages":[{"name":"page1","displayName":"Page 1"}]}`,
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/dashboards/" + resp.DashboardId,
			Permissions: []iam.AccessControlRequest{{
				PermissionLevel: "CAN_MANAGE",
				UserName:        "user@example.com",
			}},
		}, nil
	},

	"model_serving_endpoints.permissions": func(client *databricks.WorkspaceClient) (any, error) {
		waiter, err := client.ServingEndpoints.Create(context.Background(), serving.CreateServingEndpoint{
			Name: "endpoint-permissions",
			Config: &serving.EndpointCoreConfigInput{
				ServedModels: []serving.ServedModelInput{
					{
						ModelName:          "model-name",
						ModelVersion:       "1",
						WorkloadSize:       "Small",
						ScaleToZeroEnabled: true,
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/serving-endpoints/" + waiter.Response.Name,
			Permissions: []iam.AccessControlRequest{{
				PermissionLevel: "CAN_MANAGE",
				UserName:        "user@example.com",
			}},
		}, nil
	},

	"schemas.grants": func(client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "schema",
			FullName:      "main.myschema",
			Grants: []GrantAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeCreateView},
				Principal:  "user@example.com",
			}},
		}, nil
	},

	"volumes.grants": func(client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "volume",
			FullName:      "main.myschema.myvolume",
			Grants: []GrantAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeCreateView},
				Principal:  "user@example.com",
			}},
		}, nil
	},

	"registered_models.grants": func(client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "registered-model",
			FullName:      "modelid",
			Grants: []GrantAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeCreateView},
				Principal:  "user@example.com",
			}},
		}, nil
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
	var inputConfig any
	var err error

	prepDeps, hasDeps := testDeps[group]
	if hasDeps {
		inputConfig, err = prepDeps(client)
		require.NoError(t, err)
	} else {
		var ok bool
		inputConfig, ok = testConfig[group]

		if ok {
			// For permissions, PrepareState accepts any, so skip strict type check
			if adapter.InputConfigType().String() != "interface {}" {
				require.Equal(t, adapter.InputConfigType().String(), reflect.TypeOf(inputConfig).String())
			}
		} else {
			inputConfig = reflect.New(adapter.InputConfigType().Elem()).Interface()
		}
	}

	newState, err := adapter.PrepareState(inputConfig)
	require.NoError(t, err, "PrepareState failed")

	ctx := context.Background()

	// initial DoRead() cannot find the resource
	remote, err := adapter.DoRead(ctx, "1234")
	require.Nil(t, remote)
	require.Error(t, err)
	// TODO: if errors.Is(err, databricks.ErrResourceDoesNotExist) {... }

	createdID, remoteStateFromCreate, err := adapter.DoCreate(ctx, newState)
	require.NoError(t, err, "DoCreate failed state=%v", newState)
	require.NotEmpty(t, createdID, "ID returned from DoCreate was empty")

	remote, err = adapter.DoRead(ctx, createdID)
	require.NoError(t, err)
	require.NotNil(t, remote)

	remappedState, err := adapter.RemapState(remote)
	require.NoError(t, err)
	require.NotNil(t, remappedState)

	if remoteStateFromCreate != nil {
		remappedRemoteStateFromCreate, err := adapter.RemapState(remoteStateFromCreate)
		require.NoError(t, err)
		require.Equal(t, remappedState, remappedRemoteStateFromCreate)
	}

	remoteStateFromWaitCreate, err := adapter.WaitAfterCreate(ctx, newState)
	require.NoError(t, err)
	if remoteStateFromWaitCreate != nil {
		require.Equal(t, remote, remoteStateFromWaitCreate)
	}

	remoteStateFromUpdate, err := adapter.DoUpdate(ctx, createdID, newState)
	require.NoError(t, err, "DoUpdate failed")
	if remoteStateFromUpdate != nil {
		remappedStateFromUpdate, err := adapter.RemapState(remoteStateFromUpdate)
		require.NoError(t, err)
		changes, err := structdiff.GetStructDiff(remappedState, remappedStateFromUpdate, nil)
		require.NoError(t, err)
		// Filter out timestamp fields that are expected to differ in value
		var relevantChanges []structdiff.Change
		for _, change := range changes {
			fieldName := change.Path.String()
			if fieldName != "updated_at" {
				relevantChanges = append(relevantChanges, change)
			}
		}
		require.Empty(t, relevantChanges, "unexpected differences found: %v", relevantChanges)
	}

	remoteStateFromWaitUpdate, err := adapter.WaitAfterUpdate(ctx, newState)
	require.NoError(t, err)
	if remoteStateFromWaitUpdate != nil {
		remappedStateFromWaitUpdate, err := adapter.RemapState(remoteStateFromWaitUpdate)
		require.NoError(t, err)
		require.Equal(t, remappedState, remappedStateFromWaitUpdate)
	}

	require.NoError(t, structwalk.Walk(newState, func(path *structpath.PathNode, val any, field *reflect.StructField) {
		remoteValue, err := structaccess.Get(remappedState, path)
		if err != nil {
			t.Errorf("Failed to read %s from remapped remote state %#v", path.String(), remappedState)
		}
		if val == nil {
			// t.Logf("Ignoring %s nil, remoteValue=%#v", path.String(), remoteValue)
			return
		}
		v := reflect.ValueOf(val)
		if v.IsZero() {
			// t.Logf("Ignoring %s zero (%#v), remoteValue=%#v", path.String(), val, remoteValue)
			// testserver can set field to backend-generated value
			return
		}
		// t.Logf("Testing %s v=%#v, remoteValue=%#v", path.String(), val, remoteValue)
		// We expect fields set explicitly to be preserved by testserver, which is true for all resources as of today.
		// If not true for your resource, add exception here:
		assert.Equal(t, val, remoteValue, "path=%q\nnewState=%s\nremappedState=%s", path.String(), jsonDump(newState), jsonDump(remappedState))
	}))

	err = adapter.DoDelete(ctx, createdID)
	require.NoError(t, err)

	path, err := structpath.Parse("name")
	require.NoError(t, err)

	_, err = adapter.ClassifyChange(structdiff.Change{
		Path: path,
		Old:  nil,
		New:  "mynewname",
	}, remote, true)
	require.NoError(t, err)

	_, err = adapter.ClassifyChange(structdiff.Change{
		Path: path,
		Old:  nil,
		New:  "mynewname",
	}, remote, false)
	require.NoError(t, err)

	deleteIsNoop := strings.HasSuffix(group, "permissions") || strings.HasSuffix(group, "grants")

	remoteAfterDelete, err := adapter.DoRead(ctx, createdID)
	if deleteIsNoop {
		require.NoError(t, err)
	} else {
		require.Error(t, err)
		require.Nil(t, remoteAfterDelete)
	}
}

// validateFields uses structwalk to generate all valid field paths and checks membership.
func validateFields(t *testing.T, configType reflect.Type, fields map[string]deployplan.ActionType) {
	validPaths := make(map[string]struct{})

	err := structwalk.WalkType(configType, func(path *structpath.PathNode, typ reflect.Type, field *reflect.StructField) bool {
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

// TestFieldTriggers validates that all trigger keys
// exist in the corresponding ConfigType for each resource.
func TestFieldTriggers(t *testing.T) {
	for resourceName, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, nil)
		require.NoError(t, err)

		t.Run(resourceName+"_local", func(t *testing.T) {
			validateFields(t, adapter.StateType(), adapter.fieldTriggersLocal)
		})
		t.Run(resourceName+"_remote", func(t *testing.T) {
			validateFields(t, adapter.StateType(), adapter.fieldTriggersRemote)
		})
	}
}

func setupTestServerClient(t *testing.T) (*testserver.Server, *databricks.WorkspaceClient) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:               server.URL,
		Token:              "testtoken",
		RateLimitPerSecond: math.MaxInt,
	})
	require.NoError(t, err)
	return server, client
}

func jsonDump(obj any) string {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
