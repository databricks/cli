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

const (
	_jobs           = "jobs"
	_pipelines      = "pipelines"
	_schemas        = "schemas"
	_volumes        = "volumes"
	_apps           = "apps"
	_sql_warehouses = "sql_warehouses"
)

var supportedResources = map[string]reflect.Value{
	_jobs:           reflect.ValueOf(NewResourceJob),
	_pipelines:      reflect.ValueOf(NewResourcePipeline),
	_schemas:        reflect.ValueOf(NewResourceSchema),
	_volumes:        reflect.ValueOf(NewResourceVolume),
	_apps:           reflect.ValueOf(NewResourceApp),
	_sql_warehouses: reflect.ValueOf(NewResourceSqlWarehouse),
}

// This types matches what Config() returns and should match 'config' field in the resource struct
var supportedResourcesTypes = map[string]reflect.Type{
	_jobs:           reflect.TypeOf(ResourceJob{}.config),
	_pipelines:      reflect.TypeOf(ResourcePipeline{}.config),
	_schemas:        reflect.TypeOf(ResourceSchema{}.config),
	_volumes:        reflect.TypeOf(ResourceVolume{}.config),
	_apps:           reflect.TypeOf(ResourceApp{}.config),
	_sql_warehouses: reflect.TypeOf(ResourceSqlWarehouse{}.config),
}

type DeleteResourceFN = func(ctx context.Context, client *databricks.WorkspaceClient, oldID string) error

var deletableResources = map[string]DeleteResourceFN{
	_jobs:           DeleteJob,
	_pipelines:      DeletePipeline,
	_schemas:        DeleteSchema,
	_volumes:        DeleteVolume,
	_apps:           DeleteApp,
	_sql_warehouses: DeleteSqlWarehouse,
}

type IResource interface {
	Config() any

	// Create the resource. Returns id of the resource.
	DoCreate(ctx context.Context) (string, error)

	// Update the resource. Returns id of the resource.
	// Usually returns the same id as oldId but can also return a different one (e.g. schemas and volumes when certain fields are changed)
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
	ctor, ok := supportedResources[group]
	if !ok {
		return nil, nil, fmt.Errorf("unsupported resource type: %s", group)
	}

	cfgType, ok := supportedResourcesTypes[group]
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

	result, err := invokeConstructor(ctor, client, config)
	if err != nil {
		return nil, nil, err
	}

	return result, cfgType, nil
}

func DeleteResource(ctx context.Context, client *databricks.WorkspaceClient, group, id string) error {
	fn, ok := deletableResources[group]
	if !ok {
		return fmt.Errorf("cannot delete %s", group)
	}
	return fn(ctx, client, id)
}
