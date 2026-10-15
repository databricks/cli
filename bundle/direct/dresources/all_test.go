package dresources

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/apierrs"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/alerts/v2"
	"github.com/databricks/cli/libs/sdktypes/database/v1"
	"github.com/databricks/cli/libs/sdktypes/experiments/v1"
	jobs "github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/modelregistry/v1"
	"github.com/databricks/cli/libs/sdktypes/modelserving/v1"
	postgres "github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/sdktypes/secrets/v1"
	grants "github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	"github.com/databricks/cli/libs/sdktypes/vectorsearch/v1"
	"github.com/databricks/cli/libs/snapshot"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go/config"
	sdkapps "github.com/databricks/sdk-go/apps/v1"
	sdkgenie "github.com/databricks/sdk-go/genie/v1"
	sdklakeview "github.com/databricks/sdk-go/lakeview/v1"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
	sdkvectorsearch "github.com/databricks/sdk-go/vectorsearch/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testConfig map[string]any = map[string]any{
	"apps": &resources.App{
		Name: new("myapp"),
	},

	"catalogs": &resources.Catalog{
		Name:    new("mycatalog"),
		Comment: new("Test catalog"),
		// Note: EnablePredictiveOptimization and IsolationMode cannot be set during creation,
		// only during updates. They are not included in the test config.
	},

	"external_locations": &resources.ExternalLocation{
		Name:           new("myexternallocation"),
		Url:            new("s3://mybucket/mypath"),
		CredentialName: new("mycredential"),
		Comment:        new("Test external location"),
	},

	"schemas": &resources.Schema{
		CatalogName: new("main"),
		Name:        new("myschema"),
		Comment:     new("Test schema"),
	},

	"volumes": &resources.Volume{
		CatalogName: new("main"),
		SchemaName:  new("myschema"),
		Name:        new("myvolume"),
		Comment:     new("Test volume"),
	},

	"database_instances": &resources.DatabaseInstance{
		Name: new("mydbinstance"),
	},

	"instance_pools": &resources.InstancePool{
		InstancePoolName: new("my-instance-pool"),
		NodeTypeId:       new("i3.xlarge"),
	},

	"synced_database_tables": &resources.SyncedDatabaseTable{
		Name: new("main.myschema.my_synced_table"),
	},

	"model_services": &resources.ModelService{
		Parent:         "schemas/main.default",
		ModelServiceId: "my_model_service",
		Comment:        "Test model service",
	},

	"mcp_services": &resources.McpService{
		Parent:       "schemas/main.default",
		McpServiceId: "my_mcp_service",
		Comment:      "Test mcp service",
	},

	"model_provider_services": &resources.ModelProviderService{
		Parent:                 "schemas/main.default",
		ModelProviderServiceId: "my_model_provider_service",
		Comment:                "Test model provider service",
	},

	"registered_models": &resources.RegisteredModel{
		Name:            new("my_registered_model"),
		Comment:         new("Test registered model"),
		CatalogName:     new("main"),
		SchemaName:      new("default"),
		StorageLocation: new("s3://my-bucket/my-path"),
	},

	"experiments": &resources.MlflowExperiment{
		Name: new("my-experiment"),
		Tags: []experiments.ExperimentTag{
			{
				Key:   new("my-tag"),
				Value: new("my-value"),
			},
		},
		ArtifactLocation: new("s3://my-bucket/my-experiment"),
	},

	"models": &resources.MlflowModel{
		Name:        new("my_mlflow_model"),
		Description: new("my_mlflow_model_description"),
		Tags: []modelregistry.RegisteredModelTag{
			{
				Key:   new("k1"),
				Value: new("v1"),
			},
		},
	},

	"secrets": &resources.Secret{
		CatalogName: new("main"),
		SchemaName:  new("default"),
		Name:        new("my_secret"),
		Value:       new("my_secret_value"),
		Comment:     new("Test secret"),
	},

	"secret_scopes": &resources.SecretScope{
		Name:        "my_secret_scope",
		BackendType: secrets.ScopeBackendType_AzureKeyvault,
		Permissions: []resources.SecretScopePermission{
			{
				Level:    resources.SecretScopePermissionLevelManage,
				UserName: "user@example.com",
			},
		},
		KeyvaultMetadata: &secrets.AzureKeyVaultSecretScopeMetadata{
			DnsName:    new("https://my-keyvault.vault.azure.net/"),
			ResourceId: new("my-keyvault-resource-id"),
		},
	},

	"model_serving_endpoints": &resources.ModelServingEndpoint{
		Name: new("my-endpoint"),
		Config: &modelserving.EndpointCoreConfig{
			//nolint:staticcheck
			AutoCaptureConfig: &modelserving.AutoCaptureConfig{
				CatalogName:     new("main"),
				SchemaName:      new("myschema"),
				TableNamePrefix: new("my_table"),
				Enabled:         new(true),
			},
			ServedModels: nil,
			ServedEntities: []modelserving.ServedModel{
				{
					EntityName:                new("entity-name"),
					EntityVersion:             new("1"),
					WorkloadSize:              new("Small"),
					ScaleToZeroEnabled:        new(true),
					EnvironmentVars:           map[string]string{"key": "value"},
					InstanceProfileArn:        new("arn:aws:iam::123456789012:instance-profile/my-instance-profile"),
					MaxProvisionedConcurrency: new(10),
					MaxProvisionedThroughput:  new(100),
					MinProvisionedConcurrency: new(1),
					MinProvisionedThroughput:  new(10),
					Name:                      new("entity-name"),
					ProvisionedModelUnits:     new(int64(100)),
					ExternalModel:             nil,
				},
			},
			TrafficConfig: &modelserving.TrafficConfig{
				Routes: []modelserving.Route{
					{
						ServedModelName:   new("model-name-1"),
						TrafficPercentage: new(100),
					},
				},
			},
		},
	},

	"postgres_projects": &resources.PostgresProject{
		ProjectId:   "test-project",
		DisplayName: new("Test Project"),
		PgVersion:   new(16),
	},

	"postgres_catalogs": &resources.PostgresCatalog{
		CatalogId: "test_catalog",
	},

	"postgres_synced_tables": &resources.PostgresSyncedTable{
		SyncedTableId: "main.public.trips_synced",
	},

	"alerts": &resources.Alert{
		DisplayName: new("my-alert"),
		QueryText:   new("SELECT 1"),
		WarehouseId: new("test-warehouse-id"),
		Schedule: &alerts.CronSchedule{
			QuartzCronSchedule: new("0 0 12 * * ?"),
			TimezoneId:         new("UTC"),
		},
		Evaluation: &alerts.AlertEvaluation{
			ComparisonOperator: alerts.ComparisonOperator_GreaterThan,
			Source: &alerts.AlertOperandColumn{
				Name: new("column1"),
			},
			Threshold: &alerts.AlertOperand{
				Column: &alerts.AlertOperandColumn{
					Name: new("column2"),
				},
			},
		},
	},

	"quality_monitors": &resources.QualityMonitor{
		TableName:        "main.myschema.mytable",
		AssetsDir:        "/Workspace/Users/user@example.com/assets",
		OutputSchemaName: "main.myschema",
	},

	"dashboards": &resources.Dashboard{
		DisplayName:         "my-dashboard",
		ParentPath:          "/Workspace/Users/user@example.com",
		WarehouseId:         "test-warehouse-id",
		SerializedDashboard: `{"pages":[{"name":"page1","displayName":"Page 1","pageType":"PAGE_TYPE_CANVAS"}]}`,

		DatasetCatalog: "main",
		DatasetSchema:  "myschema",
	},

	"genie_spaces": &resources.GenieSpace{
		Title:           "my-genie-space",
		WarehouseId:     "test-warehouse-id",
		ParentPath:      "/Workspace/Users/user@example.com",
		SerializedSpace: "{}",
	},

	"vector_search_endpoints": &resources.VectorSearchEndpoint{
		Name:         new("my-endpoint"),
		EndpointType: vectorsearch.EndpointType_Standard,
	},

	"vector_search_indexes": &resources.VectorSearchIndex{
		Name:         new("main.default.my_index"),
		EndpointName: new("my-index-endpoint"),
		PrimaryKey:   new("id"),
		IndexType:    vectorsearch.VectorIndexType("DELTA_SYNC"),
		DeltaSyncIndexSpec: &vectorsearch.DeltaSyncVectorIndexSpecRequest{
			SourceTable:  new("main.default.source_table"),
			PipelineType: vectorsearch.PipelineType("TRIGGERED"),
		},
		Grants: []grants.PrivilegeAssignment{{
			Principal:  new("user@example.com"),
			Privileges: []string{"SELECT"},
		}},
	},

	"internal_immutable_snapshots": &resources.Snapshot{
		RemoteRoot: "/Workspace/Users/" + utils.Deref(testserver.TestUserSP.UserName) + "/.snapshots",
		BundleID:   "test-bundle-id",
		ACL:        []snapshot.ACLEntry{{UserName: "user@example.com", PermissionLevel: "CAN_READ"}},
	},
}

type prepareWorkspace func(ctx context.Context, client *sdkclients.Clients) (any, error)

// some resource require other resources to exist
var testDeps = map[string]prepareWorkspace{
	"database_catalogs": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		_, _, err := (&ResourceDatabaseInstance{client: client}).DoCreate(ctx, &database.DatabaseInstance{
			Name: new("mydbinstance1"),
		})

		return &resources.DatabaseCatalog{
			Name:                 new("mydbcatalog"),
			DatabaseInstanceName: new("mydbinstance1"),
		}, err
	},

	"vector_search_indexes": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		_, _, err := (&ResourceVectorSearchEndpoint{client: client}).DoCreate(ctx, &vectorsearch.CreateEndpointRequest{
			Name:         new("my-index-endpoint"),
			EndpointType: vectorsearch.EndpointType_Standard,
		})
		if err != nil {
			return nil, err
		}

		return testConfig["vector_search_indexes"], nil
	},

	"job_runs": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		// A run can only be triggered against an existing job, so create one first.
		id, _, err := (&ResourceJob{client: client}).DoCreate(ctx, &jobs.JobSettings{
			Name: new("job-for-run"),
			Tasks: []jobs.TaskSettings{
				{
					TaskKey: new("t"),
					NotebookTask: &jobs.NotebookTask{
						NotebookPath: new("/Workspace/Users/user@example.com/notebook"),
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}
		jobID, err := parseJobID(id)
		if err != nil {
			return nil, err
		}

		return &resources.JobRun{
			JobId: new(jobID),
		}, nil
	},

	"jobs.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		id, _, err := (&ResourceJob{client: client}).DoCreate(ctx, &jobs.JobSettings{
			Name: new("job-permissions"),
			Tasks: []jobs.TaskSettings{
				{
					TaskKey: new("t"),
					NotebookTask: &jobs.NotebookTask{
						NotebookPath: new("/Workspace/Users/user@example.com/notebook"),
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}
		return &PermissionsState{
			ObjectID: "/jobs/" + id,
			EmbeddedSlice: []StatePermission{{
				Level:    "IS_OWNER",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"pipelines.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		id, _, err := (&ResourcePipeline{client: client}).DoCreate(ctx, &PipelineState{
			Name: new("pipeline-permissions"),
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/pipelines/" + id,
			EmbeddedSlice: []StatePermission{{
				// Pipelines require exactly one owner, like jobs.
				Level:    "IS_OWNER",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"models.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		name, _, err := (&ResourceMlflowModel{client: client}).DoCreate(ctx, &modelregistry.CreateRegisteredModelRequest{
			Name:        new("model-permissions"),
			Description: new("model for permissions testing"),
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/registered-models/" + name,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"experiments.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		id, _, err := (&ResourceExperiment{client: client}).DoCreate(ctx, &experiments.CreateExperimentRequest{
			Name: new("experiment-permissions"),
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/experiments/" + id,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"clusters.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &PermissionsState{
			ObjectID: "/clusters/cluster-permissions",
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"cluster_policies.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &PermissionsState{
			ObjectID: "/cluster-policies/cluster-policy-permissions",
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_USE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"instance_pools.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &PermissionsState{
			ObjectID: "/instance-pools/pool-permissions",
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"apps.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		waiter, err := client.Apps.CreateApp(ctx, sdkapps.CreateAppRequest{
			App: &sdkapps.App{
				Name: new("app-permissions"),
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/apps/" + waiter.GetName(),
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"sql_warehouses.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &PermissionsState{
			ObjectID: "/sql/warehouses/warehouse-permissions",
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"database_instances.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		name, _, err := (&ResourceDatabaseInstance{client: client}).DoCreate(ctx, &database.DatabaseInstance{
			Name: new("dbinstance-permissions"),
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/database-instances/" + name,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"postgres_projects.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		const projectID = "permissions-project"
		waiter, err := client.Postgres.CreateProject(ctx, sdkpostgres.CreateProjectRequest{
			ProjectId: new(projectID),
		})
		if err != nil {
			return nil, err
		}
		if _, err := waiter.Wait(ctx); err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/database-projects/" + projectID,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"dashboards.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		parentPath := "/Workspace/Users/user@example.com"

		// Create parent directory if it doesn't exist
		err := client.Workspace.MkdirsByPath(ctx, parentPath)
		if err != nil {
			return nil, err
		}

		resp, err := client.Lakeview.CreateDashboard(ctx, sdklakeview.CreateDashboardRequest{
			Dashboard: &sdklakeview.DashboardCreate{
				DisplayName:         new("dashboard-permissions"),
				ParentPath:          new(parentPath),
				SerializedDashboard: new(`{"pages":[{"name":"page1","displayName":"Page 1"}]}`),
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/dashboards/" + utils.Deref(resp.DashboardId),
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"genie_spaces.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		resp, err := client.Genie.CreateSpace(ctx, sdkgenie.GenieCreateSpaceRequest{
			Title:           new("genie-space-permissions"),
			WarehouseId:     new("test-warehouse-id"),
			SerializedSpace: new("{}"),
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/genie/" + utils.Deref(resp.SpaceId),
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"model_serving_endpoints.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		name, _, err := (&ResourceModelServingEndpoint{client: client}).DoCreate(ctx, &modelserving.CreateInferenceEndpointRequest{
			Name: new("endpoint-permissions"),
			Config: &modelserving.EndpointCoreConfig{
				ServedModels: []modelserving.ServedModel{
					{
						ModelName:          new("model-name"),
						ModelVersion:       new("1"),
						WorkloadSize:       new("Small"),
						ScaleToZeroEnabled: new(true),
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/serving-endpoints/" + name,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"vector_search_endpoints.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		waiter, err := client.VectorSearch.CreateEndpoint(ctx, sdkvectorsearch.CreateEndpointRequest{
			Name:         new("vs-endpoint-permissions"),
			EndpointType: sdkvectorsearch.EndpointType_Standard,
		})
		if err != nil {
			return nil, err
		}
		endpoint, err := client.VectorSearch.GetEndpoint(ctx, sdkvectorsearch.GetEndpointRequest{Name: new(waiter.GetName())})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/vector-search-endpoints/" + utils.Deref(endpoint.Id),
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"alerts.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		id, _, err := (&ResourceAlert{client: client}).DoCreate(ctx, &alerts.Alert{
			DisplayName: new("alert-permissions"),
			QueryText:   new("SELECT 1"),
			WarehouseId: new("test-warehouse-id"),
			Schedule: &alerts.CronSchedule{
				QuartzCronSchedule: new("0 0 12 * * ?"),
				TimezoneId:         new("UTC"),
			},
			Evaluation: &alerts.AlertEvaluation{
				ComparisonOperator: alerts.ComparisonOperator_GreaterThan,
				Source: &alerts.AlertOperandColumn{
					Name: new("column1"),
				},
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/alertsv2/" + id,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	},

	"catalogs.grants": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &GrantsState{
			SecurableType: "catalog",
			FullName:      "mycatalog",
			EmbeddedSlice: []grants.PrivilegeAssignment{{
				Privileges: []string{"USE_CATALOG"},
				Principal:  new("user@example.com"),
			}},
		}, nil
	},

	"external_locations.grants": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &GrantsState{
			SecurableType: "external_location",
			FullName:      "myexternallocation",
			EmbeddedSlice: []grants.PrivilegeAssignment{{
				Privileges: []string{"READ_FILES"},
				Principal:  new("user@example.com"),
			}},
		}, nil
	},

	"schemas.grants": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &GrantsState{
			SecurableType: "schema",
			FullName:      "main.myschema",
			EmbeddedSlice: []grants.PrivilegeAssignment{{
				Privileges: []string{"CREATE_VIEW"},
				Principal:  new("user@example.com"),
			}},
		}, nil
	},

	"volumes.grants": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &GrantsState{
			SecurableType: "volume",
			FullName:      "main.myschema.myvolume",
			EmbeddedSlice: []grants.PrivilegeAssignment{{
				Privileges: []string{"CREATE_VIEW"},
				Principal:  new("user@example.com"),
			}},
		}, nil
	},

	"registered_models.grants": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &GrantsState{
			SecurableType: "registered-model",
			FullName:      "modelid",
			EmbeddedSlice: []grants.PrivilegeAssignment{{
				Privileges: []string{"CREATE_VIEW"},
				Principal:  new("user@example.com"),
			}},
		}, nil
	},

	"vector_search_indexes.grants": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &GrantsState{
			SecurableType: "table",
			FullName:      "main.default.my_index",
			EmbeddedSlice: []grants.PrivilegeAssignment{{
				Privileges: []string{"SELECT"},
				Principal:  new("user@example.com"),
			}},
		}, nil
	},

	"secrets.grants": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &GrantsState{
			SecurableType: "secret",
			FullName:      "main.default.my_secret",
			EmbeddedSlice: []grants.PrivilegeAssignment{{
				Privileges: []string{"SELECT"},
				Principal:  new("user@example.com"),
			}},
		}, nil
	},

	"model_services.grants": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &GrantsState{
			SecurableType: "model_service",
			FullName:      "main.myschema.mymodelservice",
			EmbeddedSlice: []grants.PrivilegeAssignment{{
				Privileges: []string{"APPLY_TAG"},
				Principal:  new("user@example.com"),
			}},
		}, nil
	},

	"mcp_services.grants": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &GrantsState{
			SecurableType: "mcp_service",
			FullName:      "main.myschema.mymcpservice",
			EmbeddedSlice: []grants.PrivilegeAssignment{{
				Privileges: []string{"APPLY_TAG"},
				Principal:  new("user@example.com"),
			}},
		}, nil
	},

	"model_provider_services.grants": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		return &GrantsState{
			SecurableType: "model_provider_service",
			FullName:      "main.myschema.myproviderservice",
			EmbeddedSlice: []grants.PrivilegeAssignment{{
				Privileges: []string{"APPLY_TAG"},
				Principal:  new("user@example.com"),
			}},
		}, nil
	},

	"secret_scopes.permissions": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		_, _, err := (&ResourceSecretScope{client: client}).DoCreate(ctx, &SecretScopeConfig{
			Scope:            new("permissions_test_scope"),
			ScopeBackendType: secrets.ScopeBackendType_AzureKeyvault,
			BackendAzureKeyvault: &secrets.AzureKeyVaultSecretScopeMetadata{
				DnsName:    new("https://my-keyvault.vault.azure.net/"),
				ResourceId: new("my-keyvault-resource-id"),
			},
		})
		if err != nil {
			return nil, err
		}

		return &SecretScopeAclsState{
			ScopeName: "permissions_test_scope",
			Acls: []secrets.AclItem{
				{
					Principal:  new("user@example.com"),
					Permission: secrets.AclPermission_Manage,
				},
			},
		}, nil
	},

	"postgres_branches": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		// Create parent project first
		_, _, err := (&ResourcePostgresProject{client: client}).DoCreate(ctx, &PostgresProjectState{
			ProjectId:   "test-project-for-branch",
			DisplayName: new("Test Project for Branch"),
			PgVersion:   new(16),
		})
		if err != nil {
			return nil, err
		}

		return &resources.PostgresBranch{
			Parent:   "projects/test-project-for-branch",
			BranchId: "test-branch",
		}, nil
	},

	"postgres_snapshot_schedules": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		// Creating the project implicitly provisions the root "production"
		// branch, the only branch a snapshot schedule may target.
		_, _, err := (&ResourcePostgresProject{client: client}).DoCreate(ctx, &PostgresProjectState{
			ProjectId:   "test-project-for-snapshot-schedule",
			DisplayName: new("Test Project for Snapshot Schedule"),
			PgVersion:   new(16),
		})
		if err != nil {
			return nil, err
		}

		return &resources.PostgresSnapshotSchedule{
			Branch: "projects/test-project-for-snapshot-schedule/branches/production",
		}, nil
	},

	"postgres_endpoints": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		// Create parent project first
		_, _, err := (&ResourcePostgresProject{client: client}).DoCreate(ctx, &PostgresProjectState{
			ProjectId:   "test-project-for-endpoint",
			DisplayName: new("Test Project for Endpoint"),
			PgVersion:   new(16),
		})
		if err != nil {
			return nil, err
		}

		// Create parent branch
		_, _, err = (&ResourcePostgresBranch{client: client}).DoCreate(ctx, &PostgresBranchState{
			Parent:   "projects/test-project-for-endpoint",
			BranchId: "test-branch-for-endpoint",
		})
		if err != nil {
			return nil, err
		}

		return &resources.PostgresEndpoint{
			Parent:       "projects/test-project-for-endpoint/branches/test-branch-for-endpoint",
			EndpointId:   "test-endpoint",
			EndpointType: postgres.EndpointType_EndpointTypeReadWrite,
		}, nil
	},

	"postgres_databases": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		// Create parent project first
		_, _, err := (&ResourcePostgresProject{client: client}).DoCreate(ctx, &PostgresProjectState{
			ProjectId:   "test-project-for-database",
			DisplayName: new("Test Project for Database"),
			PgVersion:   new(16),
		})
		if err != nil {
			return nil, err
		}

		// Create parent branch
		_, _, err = (&ResourcePostgresBranch{client: client}).DoCreate(ctx, &PostgresBranchState{
			Parent:   "projects/test-project-for-database",
			BranchId: "test-branch-for-database",
		})
		if err != nil {
			return nil, err
		}

		return &resources.PostgresDatabase{
			Parent:           "projects/test-project-for-database/branches/test-branch-for-database",
			DatabaseId:       "test-database",
			PostgresDatabase: new("app_db"),
			Role:             new("projects/test-project-for-database/branches/test-branch-for-database/roles/owner"),
		}, nil
	},

	"postgres_roles": func(ctx context.Context, client *sdkclients.Clients) (any, error) {
		// Create parent project first
		_, _, err := (&ResourcePostgresProject{client: client}).DoCreate(ctx, &PostgresProjectState{
			ProjectId:   "test-project-for-role",
			DisplayName: new("Test Project for Role"),
			PgVersion:   new(16),
		})
		if err != nil {
			return nil, err
		}

		// Create parent branch
		_, _, err = (&ResourcePostgresBranch{client: client}).DoCreate(ctx, &PostgresBranchState{
			Parent:   "projects/test-project-for-role",
			BranchId: "test-branch-for-role",
		})
		if err != nil {
			return nil, err
		}

		return &resources.PostgresRole{
			Parent:       "projects/test-project-for-role/branches/test-branch-for-role",
			RoleId:       "test-role",
			PostgresRole: new("test_role"),
		}, nil
	},
}

func TestAll(t *testing.T) {
	_, client := setupTestServerClient(t)

	for resourceType, resource := range SupportedResources {
		t.Run(resourceType, func(t *testing.T) {
			adapter, err := NewAdapter(resource, resourceType, client)
			require.NoError(t, err)
			require.NotNil(t, adapter)

			testCRUD(t, resourceType, adapter, client)
		})
	}

	m, err := InitAll(client)
	require.NoError(t, err)
	require.Len(t, m, len(SupportedResources))
}

// testIgnoreFilter encapsulates the logic for filtering fields based on ignore_remote_changes config.
type testIgnoreFilter struct {
	ignoreFields map[string]bool
	adapter      *Adapter
}

// newTestIgnoreFilter creates a filter from the adapter's resource configs.
// It also ignores fields that exist in StateType but not in RemoteType, because
// those are automatically suppressed by the planner (reason: missing_in_remote)
// and RemapState cannot populate them from remote state.
func newTestIgnoreFilter(adapter *Adapter) *testIgnoreFilter {
	ignoreFields := make(map[string]bool)
	for _, cfg := range []*ResourceLifecycleConfig{adapter.ResourceConfig(), adapter.GeneratedResourceConfig()} {
		if cfg == nil {
			continue
		}
		for _, p := range cfg.IgnoreRemoteChanges {
			ignoreFields[p.Field.String()] = true
		}
	}
	// Auto-include fields present in StateType but absent from RemoteType.
	_ = structwalk.WalkType(adapter.StateType(), func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) bool {
		if path.IsRoot() {
			return true
		}
		if structaccess.ValidatePattern(adapter.RemoteType(), path) != nil {
			ignoreFields[path.String()] = true
			return false
		}
		return true
	})
	return &testIgnoreFilter{ignoreFields: ignoreFields, adapter: adapter}
}

// shouldIgnore returns true if the field at the given path should be ignored.
func (f *testIgnoreFilter) shouldIgnore(path string) bool {
	if f.ignoreFields[path] {
		return true
	}
	// Check if this is a nested field under an ignored top-level field
	topLevelField := path
	if prefix, _, ok := strings.Cut(path, "."); ok {
		topLevelField = prefix
	}

	parts := strings.Split(topLevelField, "[")
	if len(parts) > 1 {
		topLevelField = parts[0]
	}

	return f.ignoreFields[topLevelField]
}

// filterChanges returns only the changes that should not be ignored.
// It also filters out the "updated_at" timestamp field.
func (f *testIgnoreFilter) filterChanges(changes []structdiff.Change) []structdiff.Change {
	var relevantChanges []structdiff.Change
	for _, change := range changes {
		fieldName := change.Path.String()
		if !f.shouldIgnore(fieldName) {
			relevantChanges = append(relevantChanges, change)
		}
	}
	return relevantChanges
}

// requireEqual compares two structs and fails the test if there are differences
// that are not in the ignore_remote_changes list.
func (f *testIgnoreFilter) requireEqual(t *testing.T, expected, actual any, msgAndArgs ...any) {
	changes, err := structdiff.GetStructDiff(expected, actual, nil)
	require.NoError(t, err)
	relevantChanges := f.filterChanges(changes)
	require.Empty(t, relevantChanges, msgAndArgs...)
}

func testCRUD(t *testing.T, group string, adapter *Adapter, client *sdkclients.Clients) {
	var inputConfig any
	var err error

	prepDeps, hasDeps := testDeps[group]
	if hasDeps {
		inputConfig, err = prepDeps(t.Context(), client)
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

	// The snapshot is content-addressed: DoCreate reads a zip from disk and the server
	// returns a path derived from that zip's hash. The generic fixture has no ZipPath, so
	// stage a real zip and align RelativePath/FullPath with its hash; otherwise the
	// create→read round-trip compares a fixture path against a hash-derived remote path
	// and fails. This is inherent to content-addressed resources, not a missing hook.
	if ss, ok := newState.(*SnapshotState); ok {
		snap := inputConfig.(*resources.Snapshot)
		content := []byte("test snapshot content")
		hash := snapshot.HashFromContent(content)
		zipPath := filepath.Join(t.TempDir(), hash+".zip")
		require.NoError(t, os.WriteFile(zipPath, content, 0o600))
		ss.ZipPath = filepath.ToSlash(zipPath)
		ss.RelativePath = ss.BundleID + "/" + hash
		ss.FullPath = snap.RemoteRoot + "/" + ss.RelativePath
	}

	ctx := t.Context()

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

	// RemoteType is included verbatim in the JSON plan's "remote_state" field,
	// so it must survive a JSON round-trip without losing fields.
	assertJSONRoundTrip(t, reflect.ValueOf(remote).Elem().Interface(), "RemoteType "+group)

	remappedState, err := adapter.RemapState(remote)
	require.NoError(t, err)
	require.NotNil(t, remappedState)

	// Create filter for fields that should be ignored based on ignore_remote_changes config.
	ignoreFilter := newTestIgnoreFilter(adapter)

	if remoteStateFromCreate != nil {
		remappedRemoteStateFromCreate, err := adapter.RemapState(remoteStateFromCreate)
		require.NoError(t, err)
		ignoreFilter.requireEqual(t, remappedState, remappedRemoteStateFromCreate,
			"unexpected differences between remappedState and remappedRemoteStateFromCreate")
	}

	remoteStateFromWaitCreate, err := adapter.WaitAfterCreate(ctx, createdID, newState)
	require.NoError(t, err)
	if remoteStateFromWaitCreate != nil {
		// Re-read after the wait: the earlier remote can be stale for resources
		// whose state settles while WaitAfterCreate blocks (e.g. job_runs).
		remote, err = adapter.DoRead(ctx, createdID)
		require.NoError(t, err)
		require.Equal(t, remote, remoteStateFromWaitCreate)

		remappedState, err = adapter.RemapState(remote)
		require.NoError(t, err)
	}

	if adapter.HasDoUpdate() {
		remoteStateFromUpdate, err := adapter.DoUpdate(ctx, createdID, newState, &deployplan.PlanEntry{})
		require.NoError(t, err, "DoUpdate failed")
		if remoteStateFromUpdate != nil {
			remappedStateFromUpdate, err := adapter.RemapState(remoteStateFromUpdate)
			require.NoError(t, err)
			// Compare DoUpdate's result against a fresh DoRead: server-generated
			// fields (e.g. etag) may change on any write, so DoUpdate's return
			// value must match what DoRead returns right after.
			remotePostUpdate, err := adapter.DoRead(ctx, createdID)
			require.NoError(t, err)
			remappedPostUpdate, err := adapter.RemapState(remotePostUpdate)
			require.NoError(t, err)
			ignoreFilter.requireEqual(t, remappedPostUpdate, remappedStateFromUpdate,
				"unexpected differences between remappedState and remappedStateFromUpdate")
			// DoUpdate may mutate newState in place (e.g. etag), so update remappedState
			// to match the post-update server state for the field checks below.
			remappedState = remappedStateFromUpdate
		}

		remoteStateFromWaitUpdate, err := adapter.WaitAfterUpdate(ctx, createdID, newState)
		require.NoError(t, err)
		if remoteStateFromWaitUpdate != nil {
			remappedStateFromWaitUpdate, err := adapter.RemapState(remoteStateFromWaitUpdate)
			require.NoError(t, err)
			ignoreFilter.requireEqual(t, remappedState, remappedStateFromWaitUpdate,
				"unexpected differences between remappedState and remappedStateFromWaitUpdate")
		}
	}

	require.NoError(t, structwalk.Walk(newState, func(path *structpath.PathNode, val any, field *reflect.StructField) {
		// Skip fields configured in ignore_remote_changes.
		if ignoreFilter.shouldIgnore(path.String()) {
			return
		}
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

	err = adapter.DoDelete(ctx, createdID, newState)
	require.NoError(t, err)

	// WaitAfterDelete polls until the resource reads back gone; a NotFound is that
	// success and the caller (DeploymentUnit.waitDeleted) maps it to nil, so accept it here too.
	err = adapter.WaitAfterDelete(ctx, createdID)
	if !apierrs.IsMissing(err) {
		require.NoError(t, err)
	}

	p, err := structpath.ParsePath("name")
	require.NoError(t, err)

	if adapter.HasOverrideChangeDesc() {
		err = adapter.OverrideChangeDesc(ctx, p, &deployplan.ChangeDesc{}, nil)
		require.NoError(t, err)
	}

	// A resource that implements no DoDelete (permissions, grants, job_runs)
	// leaves the resource in place, so DoRead still succeeds afterwards.
	// postgres_snapshot_schedules does implement DoDelete but has no delete
	// endpoint: it disables the schedule by setting an empty cadence set, and the
	// schedule remains readable (it is intrinsic to the branch).
	deleteIsNoop := !adapter.HasDoDelete() || group == "postgres_snapshot_schedules"
	isImmutable := strings.HasSuffix(group, "internal_immutable_snapshots")
	// Apps DoDelete is fire-and-forget: the API returns success while the app
	// sits in DELETING state for up to ~20 minutes before the record is removed.
	// A GET on the DELETING app returns the app, not 404 -- the testserver
	// mirrors that in libs/testserver/apps.go. DoRead therefore still succeeds
	// here; the planner treats this transient state as gone via ResourceApp.IsGone
	// so delete/destroy stay idempotent (acceptance/bundle/invariant/{delete,destroy}_idempotent).
	deleteLeavesDeleting := group == "apps"

	remoteAfterDelete, err := adapter.DoRead(ctx, createdID)
	switch {
	case isImmutable:
		require.NoError(t, err)
		assert.True(t, adapter.IsGone(remoteAfterDelete))
	case deleteIsNoop:
		require.NoError(t, err)
		// The resource genuinely still exists, so it must not report as gone.
		assert.False(t, adapter.IsGone(remoteAfterDelete))
	case deleteLeavesDeleting:
		require.NoError(t, err)
		require.NotNil(t, remoteAfterDelete)
		// IsGone lets the planner short-circuit the second delete on this
		// transient DELETING state; this is what keeps the delete/destroy
		// invariant tests idempotent, so assert the contract directly.
		assert.True(t, adapter.IsGone(remoteAfterDelete))
	default:
		require.Error(t, err)
		require.Nil(t, remoteAfterDelete)
	}
}

// TestResourceConfig validates that all field patterns in resource config
// exist in the corresponding StateType for each resource.
func TestResourceConfig(t *testing.T) {
	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)

		cfg := adapter.ResourceConfig()
		if cfg == nil {
			continue
		}

		t.Run(resourceType, func(t *testing.T) {
			validateResourceConfig(t, adapter.StateType(), adapter.RemoteType(), cfg)
		})
	}
}

// TestGeneratedResourceConfig validates that all field patterns in generated resource config
// exist in the corresponding StateType for each resource.
func TestGeneratedResourceConfig(t *testing.T) {
	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)

		cfg := adapter.GeneratedResourceConfig()
		if cfg == nil {
			continue
		}

		t.Run(resourceType, func(t *testing.T) {
			validateResourceConfig(t, adapter.StateType(), adapter.RemoteType(), cfg)
		})
	}
}

func validateResourceConfig(t *testing.T, stateType, remoteType reflect.Type, cfg *ResourceLifecycleConfig) {
	for _, p := range cfg.RecreateOnChanges {
		assert.NoError(t, structaccess.ValidatePattern(stateType, p.Field), "RecreateOnChanges: %s", p.Field)
	}
	for _, p := range cfg.ProvidedIDFields {
		assert.NoError(t, structaccess.ValidatePattern(stateType, p.Field), "ProvidedIDFields: %s", p.Field)
	}
	for _, p := range cfg.UpdatableIDFields {
		assert.NoError(t, structaccess.ValidatePattern(stateType, p.Field), "UpdatableIDFields: %s", p.Field)
	}
	for _, p := range cfg.IgnoreRemoteChanges {
		assert.NoError(t, structaccess.ValidatePattern(stateType, p.Field), "IgnoreRemoteChanges: %s", p.Field)
	}
	for _, p := range cfg.BackendDefaults {
		assert.NoError(t, structaccess.ValidatePattern(stateType, p.Field), "BackendDefaults: %s", p.Field)
	}
	// stable_output_fields must be valid paths in RemoteType: the reference
	// resolver reads their value from the remote cache. The backend owns the
	// value, but the field may or may not also appear in StateType, so we only
	// require presence in RemoteType.
	for _, p := range cfg.StableOutputFields {
		assert.NoError(t, structaccess.ValidatePattern(remoteType, p.Field), "StableOutputFields %s: must be a valid RemoteType path", p.Field)
	}
}

// TestNoUpdateResourcesCoverAllFields ensures that resources which do not
// implement DoUpdate classify every settable field. Without DoUpdate the
// planner rejects any "update" action ("resource does not support update
// action but plan produced update"), so a field a user can change must be
// declared recreate_on_changes (or ignore_local_changes) to recreate instead.
// A provided_id_field also recreates on change (classifyIDField), so it counts
// as covered too. An output-only field is fine under ignore_remote_changes
// because the user never sets it, so it can never produce a user-driven change.
//
// The check walks the state type and fails on any uncovered scalar leaf. This
// keeps the recreate_on_changes lists exhaustive as the SDK adds fields.
func TestNoUpdateResourcesCoverAllFields(t *testing.T) {
	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)
		if adapter.HasDoUpdate() {
			continue
		}
		if adapter.HasOverrideChangeDesc() {
			// OverrideChangeDesc classifies changes programmatically (e.g.
			// vector_search_indexes handles endpoint_uuid there), which this
			// static walk can't see into.
			continue
		}

		// A user change is only neutralized by recreate_on_changes,
		// provided_id_fields, or ignore_local_changes; output-only fields are
		// covered by ignore_remote_changes since the user never sets them. A
		// root rule (omitted field) records the empty path "", which stops the
		// walk below at the root node and thus covers every field at once.
		covered := map[string]bool{}
		for _, cfg := range []*ResourceLifecycleConfig{adapter.ResourceConfig(), adapter.GeneratedResourceConfig()} {
			if cfg == nil {
				continue
			}
			for _, r := range cfg.RecreateOnChanges {
				covered[r.Field.String()] = true
			}
			for _, r := range cfg.ProvidedIDFields {
				covered[r.Field.String()] = true
			}
			for _, r := range cfg.IgnoreLocalChanges {
				covered[r.Field.String()] = true
			}
			for _, r := range cfg.IgnoreRemoteChanges {
				if strings.HasSuffix(r.Reason, "output_only") {
					covered[r.Field.String()] = true
				}
			}
		}

		t.Run(resourceType, func(t *testing.T) {
			err := structwalk.WalkType(adapter.StateType(), func(path *structpath.PatternNode, typ reflect.Type, _ *reflect.StructField) bool {
				if covered[path.String()] {
					// This field (or its enclosing object) is classified; the
					// whole subtree is covered, so stop descending.
					return false
				}
				for typ.Kind() == reflect.Pointer {
					typ = typ.Elem()
				}
				switch typ.Kind() {
				case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
					// Intermediate node: descend and check its children.
					return true
				default:
					t.Errorf("field %q is not classified; a change to it would plan an unsupported update. Add it to recreate_on_changes.", path)
					return false
				}
			})
			require.NoError(t, err)
		})
	}
}

// stableNameOptOut is the explicit escape hatch for TestNameStableOrOptedOut: a
// resource whose `name` is backend-owned (so not auto-exempt) yet must still NOT
// be declared stable_output_fields — e.g. the backend recomputes the name on
// update, so its remote value is not stable. Empty today.
var stableNameOptOut = map[string]string{}

// nameIsUserProvided reports whether `name` is a plain user-set field: present in
// StateType and touched by no ignore rule (so not output_only or otherwise
// backend-owned). Such a name resolves from local config, so a ${...name}
// reference is never delayed — and it must not be pinned stable, since a rename
// must change the reference.
func nameIsUserProvided(adapter *Adapter, path *structpath.PathNode) bool {
	if structaccess.ValidatePath(adapter.StateType(), path) != nil {
		return false
	}
	for _, cfg := range []*ResourceLifecycleConfig{adapter.ResourceConfig(), adapter.GeneratedResourceConfig()} {
		if cfg == nil {
			continue
		}
		for _, rules := range [][]FieldRule{cfg.IgnoreRemoteChanges, cfg.IgnoreLocalChanges} {
			for _, r := range rules {
				if path.HasPatternPrefix(r.Field) {
					return false
				}
			}
		}
	}
	return true
}

// TestNameStableOrOptedOut guards against silently reintroducing ES-2202624: a
// ${resources.X.name} reference delayed during X's in-place update recreates
// every dependent that uses it as an immutable parent. To prevent that, a
// backend-assigned `name` that can be referenced must be declared under
// stable_output_fields so the reference resolves from the remote cache.
//
// For every resource that supports in-place update and has a `name` in RemoteType,
// the name must be either auto-exempt (a plain user-provided field, which resolves
// from local config), declared stable, or listed in stableNameOptOut. Resources
// with no DoUpdate are skipped: they never take an in-place-update action, so the
// reference is never delayed.
func TestNameStableOrOptedOut(t *testing.T) {
	namePath := structpath.MustParsePath("name")
	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)

		if !adapter.HasDoUpdate() {
			continue
		}
		if structaccess.ValidatePath(adapter.RemoteType(), namePath) != nil {
			continue
		}

		if nameIsUserProvided(adapter, namePath) {
			t.Logf("%s: auto-exempt — name is user-provided (in StateType, not output_only or ignored); resolves from local config", resourceType)
			continue
		}

		if reason, ok := stableNameOptOut[resourceType]; ok {
			t.Logf("%s: exempt from the stable-name check: %s", resourceType, reason)
			continue
		}

		t.Run(resourceType, func(t *testing.T) {
			assert.True(t, adapter.FieldIsStableOutput(namePath),
				"backend-assigned `name` must be declared under stable_output_fields in configs/%[1]s.yml, or listed in stableNameOptOut; otherwise ${resources.%[1]s.<key>.name} references recreate dependents on an in-place update (ES-2202624)", resourceType)
		})
	}
}

func setupTestServerClient(t *testing.T) (*testserver.Server, *sdkclients.Clients) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := sdkclients.New(t.Context(), &config.Config{
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
