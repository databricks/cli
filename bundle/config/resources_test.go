package config

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/database"

	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/databricks/databricks-sdk-go/service/workspace"

	"github.com/databricks/databricks-sdk-go/service/serving"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/mock"

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
	r := Resources{}
	rt := reflect.TypeOf(r)

	for i := range rt.NumField() {
		field := rt.Field(i)

		// Fields in Resources are expected be of the form map[string]*resourceStruct
		assert.Equal(t, reflect.Map, field.Type.Kind(), "Resource %s is not a map", field.Name)
		kt := field.Type.Key()
		assert.Equal(t, reflect.String, kt.Kind(), "Resource %s is not a map with string keys", field.Name)
		vt := field.Type.Elem()
		assert.Equal(t, reflect.Ptr, vt.Kind(), "Resource %s is not a map with pointer values", field.Name)

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
	rt := reflect.TypeOf(r)

	// Collect set of includes resource types
	var types []string
	for _, group := range r.AllResources() {
		types = append(types, group.Description.PluralName)
	}

	for i := range rt.NumField() {
		field := rt.Field(i)
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

	typ := reflect.TypeOf(Resources{})
	for i := range typ.NumField() {
		field := typ.Field(i)
		jsonTags := strings.Split(field.Tag.Get("json"), ",")
		pluralName := jsonTags[0]
		assert.Equal(t, actual[pluralName].PluralName, pluralName)
	}
}

func TestResourcesBindSupport(t *testing.T) {
	supportedResources := &Resources{
		Jobs: map[string]*resources.Job{
			"my_job": {
				JobSettings: jobs.JobSettings{},
			},
		},
		Pipelines: map[string]*resources.Pipeline{
			"my_pipeline": {
				CreatePipeline: pipelines.CreatePipeline{},
			},
		},
		Experiments: map[string]*resources.MlflowExperiment{
			"my_experiment": {
				CreateExperiment: ml.CreateExperiment{},
			},
		},
		RegisteredModels: map[string]*resources.RegisteredModel{
			"my_registered_model": {
				CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{},
			},
		},
		Schemas: map[string]*resources.Schema{
			"my_schema": {
				CreateSchema: catalog.CreateSchema{},
			},
		},
		Clusters: map[string]*resources.Cluster{
			"my_cluster": {},
		},
		Dashboards: map[string]*resources.Dashboard{
			"my_dashboard": {},
		},
		GenieSpaces: map[string]*resources.GenieSpace{
			"my_genie_space": {},
		},
		Volumes: map[string]*resources.Volume{
			"my_volume": {
				CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{},
			},
		},
		Apps: map[string]*resources.App{
			"my_app": {
				App: apps.App{},
			},
		},
		Alerts: map[string]*resources.Alert{
			"my_alert": {
				AlertV2: sql.AlertV2{},
			},
		},
		QualityMonitors: map[string]*resources.QualityMonitor{
			"my_quality_monitor": {
				CreateMonitor: catalog.CreateMonitor{},
			},
		},
		ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
			"my_model_serving_endpoint": {
				CreateServingEndpoint: serving.CreateServingEndpoint{},
			},
		},
		SecretScopes: map[string]*resources.SecretScope{
			"my_secret_scope": {
				Name: "0",
			},
		},
		SqlWarehouses: map[string]*resources.SqlWarehouse{
			"my_sql_warehouse": {
				CreateWarehouseRequest: sql.CreateWarehouseRequest{},
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
	}
	unbindableResources := map[string]bool{"model": true}

	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockJobsAPI().EXPECT().Get(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockPipelinesAPI().EXPECT().Get(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockExperimentsAPI().EXPECT().GetExperiment(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockRegisteredModelsAPI().EXPECT().Get(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockSchemasAPI().EXPECT().GetByFullName(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockClustersAPI().EXPECT().GetByClusterId(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockLakeviewAPI().EXPECT().Get(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockGenieAPI().EXPECT().GetSpace(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockVolumesAPI().EXPECT().Read(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockAppsAPI().EXPECT().GetByName(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockAlertsV2API().EXPECT().GetAlertById(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockQualityMonitorsAPI().EXPECT().Get(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockServingEndpointsAPI().EXPECT().Get(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockSecretsAPI().EXPECT().ListScopesAll(mock.Anything).Return([]workspace.SecretScope{
		{Name: "0"},
	}, nil)
	m.GetMockWarehousesAPI().EXPECT().GetById(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockDatabaseAPI().EXPECT().GetDatabaseInstance(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockDatabaseAPI().EXPECT().GetDatabaseCatalog(mock.Anything, mock.Anything).Return(nil, nil)
	m.GetMockDatabaseAPI().EXPECT().GetSyncedDatabaseTable(mock.Anything, mock.Anything).Return(nil, nil)

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
			exists, err := r.Exists(ctx, m.WorkspaceClient, "0")
			assert.NoError(t, err)
			assert.True(t, exists)
		}
	}
}
