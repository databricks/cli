package tnresources

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
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
}

var SupportedResources = map[string]ResourceSettings{
	"jobs": {
		New:        reflect.ValueOf(NewResourceJob),
		ConfigType: reflect.TypeOf(ResourceJob{}.config),
		DeleteFN:   DeleteJob,
	},
	"pipelines": {
		New:        reflect.ValueOf(NewResourcePipeline),
		ConfigType: reflect.TypeOf(ResourcePipeline{}.config),
		DeleteFN:   DeletePipeline,
	},
	"schemas": {
		New:             reflect.ValueOf(NewResourceSchema),
		ConfigType:      reflect.TypeOf(ResourceSchema{}.config),
		DeleteFN:        DeleteSchema,
		UpdateUpdatesID: true,
	},
	"volumes": {
		New:             reflect.ValueOf(NewResourceVolume),
		ConfigType:      reflect.TypeOf(ResourceVolume{}.config),
		DeleteFN:        DeleteVolume,
		UpdateUpdatesID: true,
	},
	"apps": {
		New:        reflect.ValueOf(NewResourceApp),
		ConfigType: reflect.TypeOf(ResourceApp{}.config),
		DeleteFN:   DeleteApp,
	},
	"sql_warehouses": {
		New:        reflect.ValueOf(NewResourceSqlWarehouse),
		ConfigType: reflect.TypeOf(ResourceSqlWarehouse{}.config),
		DeleteFN:   DeleteSqlWarehouse,
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
