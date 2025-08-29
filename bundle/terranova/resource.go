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

	// Type of the config snapshot struct
	ConfigType reflect.Type

	// Type of the remote state struct
	RemoteType reflect.Type

	// Function to delete a resource of this type
	DeleteFN DeleteResourceFN

	// true if ClassifyChanges() method can return a different ActionTypeRecreate
	// If RecreateAllowed is false and RecreateFields is empty, the resource id is stable.
	RecreateAllowed bool

	// If any of these fields are changed, recreation (Delete + Create) is triggered.
	// This overrides ClassifyChanges() function (so you don't need to implement that one).
	// Fields are in structdiff.Change.String() format.
	// Limitation: patterns like hello.*.world and hello[*].world are not supported
	RecreateFields map[string]struct{}
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

func TypeOfRemote(resource IResource) reflect.Type {
	return reflect.TypeOf(resource.RemoteState())
}

var SupportedResources = map[string]ResourceSettings{
	"jobs": {
		New:        reflect.ValueOf(tnresources.NewResourceJob),
		ConfigType: TypeOfConfig(&tnresources.ResourceJob{}),
		RemoteType: TypeOfRemote(&tnresources.ResourceJob{}),
		DeleteFN:   tnresources.DeleteJob,
	},
	"pipelines": {
		New:        reflect.ValueOf(tnresources.NewResourcePipeline),
		ConfigType: TypeOfConfig(&tnresources.ResourcePipeline{}),
		RemoteType: TypeOfRemote(&tnresources.ResourcePipeline{}),
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
		RemoteType: TypeOfRemote(&tnresources.ResourceSchema{}),
		DeleteFN:   tnresources.DeleteSchema,
		// TF: https://github.com/databricks/terraform-provider-databricks/blob/03a2515/catalog/resource_schema.go#L14
		RecreateFields: mkMap(
			".name",
			".catalog_name",
			".storage_root",
		),
	},
	"volumes": {
		New:        reflect.ValueOf(tnresources.NewResourceVolume),
		ConfigType: TypeOfConfig(&tnresources.ResourceVolume{}),
		RemoteType: TypeOfRemote(&tnresources.ResourceVolume{}),
		DeleteFN:   tnresources.DeleteVolume,
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
		RemoteType: TypeOfRemote(&tnresources.ResourceApp{}),
		DeleteFN:   tnresources.DeleteApp,
		RecreateFields: mkMap(
			".name",
		),
	},
	"sql_warehouses": {
		New:        reflect.ValueOf(tnresources.NewResourceSqlWarehouse),
		ConfigType: TypeOfConfig(&tnresources.ResourceSqlWarehouse{}),
		RemoteType: TypeOfRemote(&tnresources.ResourceSqlWarehouse{}),
		DeleteFN:   tnresources.DeleteSqlWarehouse,
	},
	"database_instances": {
		New:        reflect.ValueOf(tnresources.NewResourceDatabaseInstance),
		ConfigType: TypeOfConfig(&tnresources.ResourceDatabaseInstance{}),
		RemoteType: TypeOfRemote(&tnresources.ResourceDatabaseInstance{}),
		DeleteFN:   tnresources.DeleteDatabaseInstance,
	},
}

// Resource needs to implement IResourceCommon and one of the following:
// Group 1: no remote state:
//  1.1. only IResource: basic create/update
//  1.2. IResource + IResourceWait: basic create/update + extra waiting to bring resource to correct state.
// Group 2: with remote state:
//  2.1. IResourceWithRemoteState: create/update return remote state
//  2.2. IResource + IResourceWithRemoteState: basic create/update + extra waiting that also returns final remote state.
//

type IResource interface {
	// Returns stored configuration snapshot. This is the snapshot that was provided to NewResource<Type> method.
	// The return value is struct of type ConfigType (not a pointer). Never nil.
	Config() any

	// Returns stored remote state. This state is updated by DoRefresh() or any of the methods that have "Refresh" in their name.
	// This is a pointer, can be nil if refresh was not called yet and resource was never created.
	RemoteState() any

	// Reads remote state from the backend. Result is available in RemoteState().
	DoRefresh(ctx context.Context, id string) error
}

type IResourceBasic interface {
	// Create the resource. Returns id of the resource.
	// This method must not update stored remote state (this will be checked).
	DoCreate(ctx context.Context) (string, error)

	// Update the resource. ID must not change.
	DoUpdate(ctx context.Context, id string) error
}

type IResourceWithRefresh interface {
	// Create the resource. Returns id of the resource. Must update remoteState.
	DoCreateWithRefresh(ctx context.Context) (string, error)

	// Update the resource. ID must not change. Updates remoteState.
	DoUpdateWithRefresh(ctx context.Context, id string) error
}

type IResourceWaitCreateBasic interface {
	WaitAfterCreate(ctx context.Context) error
}

type IResourceWaitUpdateBasic interface {
	WaitAfterUpdate(ctx context.Context) error
}

type IResourceWaitCreateWithRefresh interface {
	WaitAfterCreateWithRefresh(ctx context.Context) error
}

type IResourceWaitUpdateWithRefresh interface {
	WaitAfterUpdateWithRefresh(ctx context.Context) error
}

// Optional method for non-default change classification.
// Default is to consider any change "an update" (RecreateFields handled separately).
type IResourceCustomClassify interface {
	ClassifyChanges(changes []structdiff.Change) deployplan.ActionType
}

// Optional method for resources that may update ID as part of update operation.
type IResourceUpdatesID interface {
	// Update the resource. Returns new id of the resource, which may be different from the old one.
	// This will only be called if actiontype is ActionTypeUpdateWithID, so ClassifyChanges must be implemented as well.
	DoUpdateWithID(ctx context.Context, oldID string) (string, error)
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
