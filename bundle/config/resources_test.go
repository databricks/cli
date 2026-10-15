package config

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/qualitymonitors"
	"github.com/databricks/cli/libs/sdktypes/alerts/v2"
	"github.com/databricks/cli/libs/sdktypes/experiments/v1"
	"github.com/databricks/cli/libs/sdktypes/uc/registeredmodels/v1"
	"github.com/databricks/cli/libs/sdktypes/warehouses/v1"

	"github.com/databricks/cli/libs/sdktypes/database/v1"

	"github.com/databricks/cli/libs/sdktypes/modelserving/v1"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	ucsecrets "github.com/databricks/cli/libs/sdktypes/uc/secrets/v1"
	"github.com/databricks/cli/libs/sdktypes/vectorsearch/v1"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/cli/libs/workspaceurls"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

// This test ensures that all resources have a custom marshaller and unmarshaller.
// This is required because DABs resources map to Databricks APIs, and they do so
// by embedding the corresponding Go SDK structs.
//
// Go SDK structs often implement custom marshalling and unmarshalling methods (based on the API specifics).
// If the Go SDK struct implements custom marshalling and unmarshalling and we do not
// for the resources at the top level, marshalling and unmarshalling operations will panic.
// Thus we will be overly cautious and ensure that all resources need a custom marshaller and unmarshaller.
//
// Why do we not assert this using an interface to assert MarshalJSON and UnmarshalJSON
// are implemented at the top level?
// If a method is implemented for an embedded struct, the top level struct will
// also have that method and satisfy the interface. This is why we cannot assert
// that the methods are implemented at the top level using an interface.
//
// Why don't we use reflection to assert that the methods are implemented at the
// top level?
// Same problem as above, the golang reflection package does not seem to provide
// a way to directly assert that MarshalJSON and UnmarshalJSON are implemented
// at the top level.
func TestCustomMarshallerIsImplemented(t *testing.T) {
	rt := reflect.TypeFor[Resources]()

	for field := range rt.Fields() {
		// Fields in Resources are expected be of the form map[string]*resourceStruct
		assert.Equal(t, reflect.Map, field.Type.Kind(), "Resource %s is not a map", field.Name)
		kt := field.Type.Key()
		assert.Equal(t, reflect.String, kt.Kind(), "Resource %s is not a map with string keys", field.Name)
		vt := field.Type.Elem()
		assert.Equal(t, reflect.Pointer, vt.Kind(), "Resource %s is not a map with pointer values", field.Name)

		// Marshalling a resourceStruct will panic if resourceStruct does not have a custom marshaller
		// This is because resourceStruct embeds a Go SDK struct that implements
		// a custom marshaller.
		// Eg: resource.Job implements MarshalJSON
		v := reflect.Zero(vt.Elem()).Interface()
		assert.NotPanics(t, func() {
			_, err := json.Marshal(v)
			assert.NoError(t, err)
		}, "Resource %s does not have a custom marshaller", field.Name)

		// Unmarshalling a *resourceStruct will panic if the resource does not have a custom unmarshaller
		// This is because resourceStruct embeds a Go SDK struct that implements
		// a custom unmarshaller.
		// Eg: *resource.Job implements UnmarshalJSON
		v = reflect.New(vt.Elem()).Interface()
		assert.NotPanics(t, func() {
			err := json.Unmarshal([]byte("{}"), v)
			assert.NoError(t, err)
		}, "Resource %s does not have a custom unmarshaller", field.Name)
	}
}

func TestResourcesAllResourcesCompleteness(t *testing.T) {
	r := Resources{}
	rt := reflect.TypeFor[Resources]()

	// Collect set of includes resource types
	var types []string
	for _, group := range r.AllResources() {
		types = append(types, group.Description.PluralName)
	}

	for field := range rt.Fields() {
		jsonTag := field.Tag.Get("json")

		if idx := strings.Index(jsonTag, ","); idx != -1 {
			jsonTag = jsonTag[:idx]
		}

		assert.Contains(t, types, jsonTag, "Field %s is missing in AllResources", field.Name)
	}
}

func TestSupportedResources(t *testing.T) {
	// Please add your resource to the SupportedResources() function in resources.go if you add a new resource.
	actual := SupportedResources()

	typ := reflect.TypeFor[Resources]()
	for field := range typ.Fields() {
		jsonTags := strings.Split(field.Tag.Get("json"), ",")
		pluralName := jsonTags[0]
		assert.Equal(t, actual[pluralName].PluralName, pluralName)
	}
}

// All bundle resources must have a workspace URL, except those listed in noURL.
// When a pattern key or a bundle plural name drifts, ResourceURL returns "" and
// this test fails loudly instead of silently producing empty URLs in bundle summary.
// When adding a new resource, add a URL pattern in libs/workspaceurls or add the
// plural name to noURL with a comment explaining why no URL exists.
func TestBundleResourcePluralNamesResolveInWorkspaceURLs(t *testing.T) {
	// Resources that intentionally have no workspace URL.
	noURL := map[string]bool{
		"external_locations": true,
		// A job run does have a workspace URL, but it's addressed by two IDs
		// (job + run) so it can't be expressed as a single-ID pattern here; it's
		// built in JobRun.InitializeURL via workspaceurls.JobRunURL instead.
		"job_runs":                     true,
		"postgres_branches":            true,
		"postgres_databases":           true,
		"postgres_endpoints":           true,
		"postgres_projects":            true,
		"postgres_roles":               true,
		"postgres_snapshot_schedules":  true,
		"secret_scopes":                true,
		"internal_immutable_snapshots": true,
	}

	supported := SupportedResources()

	// Catch stale noURL entries when a resource is removed from SupportedResources.
	for name := range noURL {
		_, ok := supported[name]
		require.Truef(t, ok, "%q is in the noURL list but is not a supported bundle resource; remove it from noURL", name)
	}

	base := url.URL{Scheme: "https", Host: "example.com"}
	for name := range supported {
		got := workspaceurls.ResourceURL(base, name, "test-id")
		if noURL[name] {
			assert.Emptyf(t, got, "workspaceurls.ResourceURL(%q) returned non-empty; remove %q from noURL or remove its URL pattern", name, name)
		} else {
			assert.NotEmptyf(t, got, "workspaceurls.ResourceURL(%q) returned empty; add a URL pattern in libs/workspaceurls or add %q to noURL", name, name)
		}
	}
}

func TestResourcesBindSupport(t *testing.T) {
	supportedResources := &Resources{
		Jobs: map[string]*resources.Job{
			"my_job": {
				JobSettings: jobs.JobSettings{},
			},
		},
		JobRuns: map[string]*resources.JobRun{
			"my_job_run": {
				RunNowRequest: jobs.RunNowRequest{},
			},
		},
		Pipelines: map[string]*resources.Pipeline{
			"my_pipeline": {
				CreatePipelineRequest: pipelines.CreatePipelineRequest{},
			},
		},
		Experiments: map[string]*resources.MlflowExperiment{
			"my_experiment": {
				CreateExperimentRequest: experiments.CreateExperimentRequest{},
			},
		},
		RegisteredModels: map[string]*resources.RegisteredModel{
			"my_registered_model": {
				CreateRegisteredModelRequest: registeredmodels.CreateRegisteredModelRequest{},
			},
		},
		Catalogs: map[string]*resources.Catalog{
			"my_catalog": {
				CreateCatalog: resources.CreateCatalog{},
			},
		},
		ExternalLocations: map[string]*resources.ExternalLocation{
			"my_external_location": {
				CreateExternalLocation: resources.CreateExternalLocation{},
			},
		},
		Schemas: map[string]*resources.Schema{
			"my_schema": {
				CreateSchema: resources.CreateSchema{},
			},
		},
		Clusters: map[string]*resources.Cluster{
			"my_cluster": {},
		},
		InstancePools: map[string]*resources.InstancePool{
			"my_instance_pool": {},
		},
		ClusterPolicies: map[string]*resources.ClusterPolicy{
			"my_cluster_policy": {},
		},
		Dashboards: map[string]*resources.Dashboard{
			"my_dashboard": {},
		},
		GenieSpaces: map[string]*resources.GenieSpace{
			"my_genie_space": {},
		},
		Volumes: map[string]*resources.Volume{
			"my_volume": {
				CreateVolumeRequestContent: resources.CreateVolumeRequestContent{},
			},
		},
		Apps: map[string]*resources.App{
			"my_app": {
				App: apps.App{},
			},
		},
		Alerts: map[string]*resources.Alert{
			"my_alert": {
				Alert: alerts.Alert{},
			},
		},
		QualityMonitors: map[string]*resources.QualityMonitor{
			"my_quality_monitor": {
				CreateMonitor: qualitymonitors.CreateMonitor{},
			},
		},
		ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
			"my_model_serving_endpoint": {
				CreateInferenceEndpointRequest: modelserving.CreateInferenceEndpointRequest{},
			},
		},
		ModelServices: map[string]*resources.ModelService{
			"my_model_service": {
				ModelServiceConfig: resources.ModelServiceConfig{},
			},
		},
		McpServices: map[string]*resources.McpService{
			"my_mcp_service": {
				McpServiceConfig: resources.McpServiceConfig{},
			},
		},
		ModelProviderServices: map[string]*resources.ModelProviderService{
			"my_model_provider_service": {
				ModelProviderServiceConfig: resources.ModelProviderServiceConfig{},
			},
		},
		SecretScopes: map[string]*resources.SecretScope{
			"my_secret_scope": {
				Name: "0",
			},
		},
		Secrets: map[string]*resources.Secret{
			"my_secret": {
				Secret: ucsecrets.Secret{
					CatalogName: new("main"),
					SchemaName:  new("default"),
					Name:        new("my_secret"),
				},
			},
		},
		SqlWarehouses: map[string]*resources.SqlWarehouse{
			"my_sql_warehouse": {
				CreateWarehouseRequest: warehouses.CreateWarehouseRequest{},
			},
		},
		DatabaseInstances: map[string]*resources.DatabaseInstance{
			"my_database_instance": {
				DatabaseInstance: database.DatabaseInstance{},
			},
		},
		DatabaseCatalogs: map[string]*resources.DatabaseCatalog{
			"my_database_catalog": {
				DatabaseCatalog: database.DatabaseCatalog{},
			},
		},
		SyncedDatabaseTables: map[string]*resources.SyncedDatabaseTable{
			"my_synced_database_table": {
				SyncedDatabaseTable: database.SyncedDatabaseTable{},
			},
		},
		PostgresProjects: map[string]*resources.PostgresProject{
			"my_postgres_project": {
				PostgresProjectConfig: resources.PostgresProjectConfig{
					ProjectId: "my-postgres-project",
					ProjectSpec: postgres.ProjectSpec{
						DisplayName: new("my_postgres_project"),
					},
				},
			},
		},
		PostgresBranches: map[string]*resources.PostgresBranch{
			"my_postgres_branch": {
				PostgresBranchConfig: resources.PostgresBranchConfig{
					BranchId: "my-postgres-branch",
					Parent:   "projects/my-postgres-project",
				},
			},
		},
		PostgresEndpoints: map[string]*resources.PostgresEndpoint{
			"my_postgres_endpoint": {
				PostgresEndpointConfig: resources.PostgresEndpointConfig{
					EndpointId: "my-postgres-endpoint",
					Parent:     "projects/my-postgres-project/branches/my-postgres-branch",
					EndpointSpec: postgres.EndpointSpec{
						EndpointType: postgres.EndpointType_EndpointTypeReadWrite,
					},
				},
			},
		},
		PostgresCatalogs: map[string]*resources.PostgresCatalog{
			"my_postgres_catalog": {
				PostgresCatalogConfig: resources.PostgresCatalogConfig{
					CatalogId: "my_postgres_catalog",
				},
			},
		},
		PostgresDatabases: map[string]*resources.PostgresDatabase{
			"my_postgres_database": {
				PostgresDatabaseConfig: resources.PostgresDatabaseConfig{
					DatabaseId: "my-postgres-database",
					Parent:     "projects/my-postgres-project/branches/my-postgres-branch",
				},
			},
		},
		PostgresRoles: map[string]*resources.PostgresRole{
			"my_postgres_role": {
				PostgresRoleConfig: resources.PostgresRoleConfig{
					RoleId: "my-postgres-role",
					Parent: "projects/my-postgres-project/branches/my-postgres-branch",
					Role_RoleSpec: postgres.Role_RoleSpec{
						PostgresRole: new("my_postgres_role"),
					},
				},
			},
		},
		PostgresSyncedTables: map[string]*resources.PostgresSyncedTable{
			"my_postgres_synced_table": {
				PostgresSyncedTableConfig: resources.PostgresSyncedTableConfig{
					SyncedTableId: "catalog.schema.my_postgres_synced_table",
				},
			},
		},
		PostgresSnapshotSchedules: map[string]*resources.PostgresSnapshotSchedule{
			"my_postgres_snapshot_schedule": {
				PostgresSnapshotScheduleConfig: resources.PostgresSnapshotScheduleConfig{
					Branch: "projects/my-postgres-project/branches/my-postgres-branch",
				},
			},
		},
		VectorSearchEndpoints: map[string]*resources.VectorSearchEndpoint{
			"my_vector_search_endpoint": {
				CreateEndpointRequest: vectorsearch.CreateEndpointRequest{
					Name:         new("my_vector_search_endpoint"),
					EndpointType: vectorsearch.EndpointType_Standard,
				},
			},
		},
		VectorSearchIndexes: map[string]*resources.VectorSearchIndex{
			"my_vector_search_index": {
				CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
					Name:         new("my_vector_search_index"),
					EndpointName: new("my_vector_search_endpoint"),
					PrimaryKey:   new("id"),
					IndexType:    vectorsearch.VectorIndexType("DELTA_SYNC"),
				},
			},
		},
	}
	unbindableResources := map[string]bool{
		"model":                       true,
		"internal_immutable_snapshot": true,
	}

	ctx := t.Context()
	// Every Exists implementation issues a GET; answer each with an empty
	// object so all resources are reported as existing. Secret scopes are
	// looked up by listing, so that listing must contain the scope.
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/secrets/scopes/list", func(testserver.Request) any {
		return map[string]any{"scopes": []map[string]any{{"name": "0"}}}
	})
	server.Handle("GET", "/{path...}", func(testserver.Request) any {
		return map[string]any{}
	})
	w, err := sdkclients.New(ctx, &sdkconfig.Config{
		Host:    server.URL,
		Token:   "testtoken",
		Loaders: []sdkconfig.Loader{sdkconfig.ConfigAttributes},
	})
	require.NoError(t, err)

	allResources := supportedResources.AllResources()
	for _, group := range allResources {
		if len(group.Resources) == 0 && !unbindableResources[group.Description.SingularName] {
			t.Fatalf("Expected at least one resource in group %s", group.Description)
		}
		for _, resource := range group.Resources {
			// bind operation requires resource to be returned from FindResourceByConfigKey
			r, err := supportedResources.FindResourceByConfigKey("my_" + resource.ResourceDescription().SingularName)
			assert.NoError(t, err)

			// bind operation requires Exists to return true
			exists, err := r.Exists(ctx, w, "0")
			assert.NoError(t, err)
			assert.True(t, exists)
		}
	}
}

func TestAllInteralResourcesAreMarkedAsInternal(t *testing.T) {
	internalResourceKeys := map[string]reflect.Type{}
	err := structwalk.WalkType(reflect.TypeFor[Resources](), func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) bool {
		if path.Len() > 2 {
			return false
		}
		if field == nil {
			return true
		}
		tag := field.Tag.Get("bundle")
		if structtag.BundleTag(tag).Internal() {
			internalResourceKeys[field.Name] = typ
		}
		return true
	})
	assert.NoError(t, err)

	for key, typ := range internalResourceKeys {
		r := reflect.MakeMap(typ)
		r.SetMapIndex(reflect.ValueOf("my_resources"), reflect.New(typ.Elem()).Elem())

		resources := &Resources{}
		res := reflect.ValueOf(resources).Elem()
		field := res.FieldByName(key)
		if !field.IsValid() && !field.CanSet() {
			t.Fatalf("Field %s is not valid", key)
		}
		field.Set(r)
		assert.True(t, resources.HasInternalResources())
	}
}
