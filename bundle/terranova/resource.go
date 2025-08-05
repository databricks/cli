package terranova

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
)

type DeleteResourceFN = func(ctx context.Context, client *databricks.WorkspaceClient, oldID string) error

type ResourceSettings struct {
	// Method to call to create new resource
	// First argument must be client* databricks.Workspace and second argument is *resource.<Resource> from bundle config
	// where Resource is appropriate resource e.g. resource.Job.
	New reflect.Value

	// Type of the stored config state
	ConfigType reflect.Type

	// Function to delete a resource of this type
	DeleteFN DeleteResourceFN

	// true if Update() method can return a different ID than that was passed in
	// If ID changes during Update and UpdateUpdatesID is false, deployment of that resource will fail with internal error.
	// This allows to make assumptions about references stability (${resources.jobs.foo.id}) when we see that
	// operation is going to be "update" & ID is guarantee not to change.
	UpdateUpdatesID bool

	// true if ClassifyChanges() method can return a different ActionTypeRecreate
	// If RecreateAllowed is false and RecreateFields is empty, the resource id is stable.
	RecreateAllowed bool

	// If any of these fields are changed, recreation (Delete + Create) is triggered.
	// This overrides ClassifyChanges() function (so you don't need to implement that one).
	// Fields are in structdiff.Change.String() format.
	// Limitation: patterns like hello.*.world and hello[*].world are not supported
	RecreateFields map[string]struct{}

	// If resource does not set RecreateFields, RecreateAllowed, UpdateUpdatesID then
	// it's ${resources.<group>.<name>.id} will considered stable for the purposes of concurrent deployment.
}

func (s *ResourceSettings) MustRecreate(changes []structdiff.Change) bool {
	if len(s.RecreateFields) == 0 {
		return false
	}
	for _, change := range changes {
		if _, ok := s.RecreateFields[change.Path.String()]; ok {
			return true
		}
	}
	return false
}

// TypeOfConfig returns the reflect.Type of the configuration returned by the resource's Config() method.
func TypeOfConfig(resource IResource) reflect.Type {
	return reflect.TypeOf(resource.Config())
}

var SupportedResources = map[string]ResourceSettings{
	"jobs": {
		New:        reflect.ValueOf(tnresources.NewResourceJob),
		ConfigType: TypeOfConfig(&tnresources.ResourceJob{}),
		DeleteFN:   tnresources.DeleteJob,
	},
	"pipelines": {
		New:        reflect.ValueOf(tnresources.NewResourcePipeline),
		ConfigType: TypeOfConfig(&tnresources.ResourcePipeline{}),
		DeleteFN:   tnresources.DeletePipeline,
		// See TF's ForceNew fields:
		// https://github.com/databricks/terraform-provider-databricks/blob/8ae24ac/pipelines/resource_pipeline.go#L207
		RecreateFields: mkMap(
			".storage",
			".catalog",
			".ingestion_definition.connection_name",
			".ingestion_definition.ingestion_gateway_id",
		),
	},
	"schemas": {
		New:        reflect.ValueOf(tnresources.NewResourceSchema),
		ConfigType: TypeOfConfig(&tnresources.ResourceSchema{}),
		DeleteFN:   tnresources.DeleteSchema,
		// TF: https://github.com/databricks/terraform-provider-databricks/blob/03a2515/catalog/resource_schema.go#L14
		RecreateFields: mkMap(
			".name",
			".catalog_name",
			".storage_root",
		),
	},
	"volumes": {
		New:             reflect.ValueOf(tnresources.NewResourceVolume),
		ConfigType:      TypeOfConfig(&tnresources.ResourceVolume{}),
		DeleteFN:        tnresources.DeleteVolume,
		UpdateUpdatesID: true,
		// TF: https://github.com/databricks/terraform-provider-databricks/blob/f5fce0f/catalog/resource_volume.go#L19
		RecreateFields: mkMap(
			".catalog_name",
			".schema_name",
			".storage_location",
			".volume_type",
		),
	},
	"apps": {
		New:        reflect.ValueOf(tnresources.NewResourceApp),
		ConfigType: TypeOfConfig(&tnresources.ResourceApp{}),
		DeleteFN:   tnresources.DeleteApp,
	},
	"sql_warehouses": {
		New:        reflect.ValueOf(tnresources.NewResourceSqlWarehouse),
		ConfigType: TypeOfConfig(&tnresources.ResourceSqlWarehouse{}),
		DeleteFN:   tnresources.DeleteSqlWarehouse,
	},
}

type IResource interface {
	Config() any

	// Create the resource. Returns id of the resource.
	DoCreate(ctx context.Context) (string, error)

	// Update the resource. Returns id of the resource.
	// Usually returns the same id as oldId but can also return a different one (e.g. schemas and volumes when certain fields are changed)
	// Note, SupportedResources[group].UpdateUpdatesID must be true for this group if ID can be changed. Otherwise function must return the same ID.
	DoUpdate(ctx context.Context, oldID string) (string, error)

	WaitAfterCreate(ctx context.Context) error
	WaitAfterUpdate(ctx context.Context) error

	ClassifyChanges(changes []structdiff.Change) deployplan.ActionType
}

// invokeConstructor converts cfg to the parameter type expected by ctor and
// executes the call, returning the IResource instance or error.
func invokeConstructor(ctor reflect.Value, client *databricks.WorkspaceClient, cfg any) (IResource, error) {
	ft := ctor.Type()

	// Sanity check – every registered function must have two inputs and two outputs.
	if ft.NumIn() != 2 || ft.NumOut() != 2 {
		return nil, errors.New("invalid constructor signature: want func(*WorkspaceClient, T) (IResource, error)")
	}

	expectedCfgType := ft.In(1) // T

	// Prepare the config value matching the expected type.
	var cfgVal reflect.Value
	if cfg == nil {
		return nil, errors.New("internal error, config must not be nil")
	} else {
		suppliedVal := reflect.ValueOf(cfg)
		if suppliedVal.Type() != expectedCfgType {
			return nil, fmt.Errorf("unexpected config type: expected %s, got %T", expectedCfgType, cfg)
		}
		cfgVal = suppliedVal
	}

	results := ctor.Call([]reflect.Value{reflect.ValueOf(client), cfgVal})

	if errIface := results[1].Interface(); errIface != nil {
		return nil, errIface.(error)
	}

	res, ok := results[0].Interface().(IResource)
	if !ok {
		return nil, errors.New("constructor did not return IResource")
	}
	return res, nil
}

func New(client *databricks.WorkspaceClient, group, name string, config any) (IResource, reflect.Type, error) {
	settings, ok := SupportedResources[group]
	if !ok {
		return nil, nil, fmt.Errorf("unsupported resource type: %s", group)
	}

	// Disallow nil configs (including typed nil pointers).
	if config == nil {
		return nil, nil, fmt.Errorf("unexpected nil in config: %s.%s", group, name)
	}

	// If the supplied config is a pointer value, dereference it so that we pass
	// the underlying struct value to the constructor. Typed nil pointers were
	// handled above.
	v := reflect.ValueOf(config)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil, fmt.Errorf("unexpected nil in config: %s.%s", group, name)
		}
	}

	result, err := invokeConstructor(settings.New, client, config)
	if err != nil {
		return nil, nil, err
	}

	return result, settings.ConfigType, nil
}

func DeleteResource(ctx context.Context, client *databricks.WorkspaceClient, group, id string) error {
	settings, ok := SupportedResources[group]
	if !ok {
		return fmt.Errorf("cannot delete %s", group)
	}
	return settings.DeleteFN(ctx, client, id)
}

func mkMap(names ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(names))
	for _, name := range names {
		result[name] = struct{}{}
	}
	return result
}
