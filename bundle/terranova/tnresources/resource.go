package tnresources

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
)

type IResource interface {
	Config() any

	// Create the resource. Returns id of the resource.
	DoCreate(ctx context.Context) (string, error)

	// Update the resource. Returns id of the resource (might be updated).
	DoUpdate(ctx context.Context, oldId string) (string, error)

	DoDelete(ctx context.Context, oldId string) error

	WaitAfterCreate(ctx context.Context) error
	WaitAfterUpdate(ctx context.Context) error

	// Get type of the struct that stores the state
	GetType() reflect.Type

	ClassifyChanges(changes []structdiff.Change) ChangeType
}

type ChangeType int

func (c ChangeType) IsRecreate() bool { return c == ChangeTypeRecreate }
func (c ChangeType) IsUpdate() bool   { return c == ChangeTypeUpdate }

const (
	ChangeTypeNone     ChangeType = 0
	ChangeTypeUpdate   ChangeType = 1
	ChangeTypeRecreate ChangeType = -1
)

var resourceConstructors = map[string]reflect.Value{
	"jobs":      reflect.ValueOf(NewResourceJob),
	"pipelines": reflect.ValueOf(NewResourcePipeline),
	"schemas":   reflect.ValueOf(NewResourceSchema),
	"apps":      reflect.ValueOf(NewResourceApp),
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
		// Treat nil as a request for the zero value of the expected config type. This
		// is useful for actions (like deletion) where the config is irrelevant.
		cfgVal = reflect.Zero(expectedCfgType)
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

func New(client *databricks.WorkspaceClient, section, name string, config any) (IResource, error) {
	ctor, ok := resourceConstructors[section]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %s", section)
	}

	// Disallow nil configs (including typed nil pointers).
	if config == nil {
		return nil, fmt.Errorf("unexpected nil in config: %s.%s", section, name)
	}

	// If the supplied config is a pointer value, dereference it so that we pass
	// the underlying struct value to the constructor. Typed nil pointers were
	// handled above.
	v := reflect.ValueOf(config)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, fmt.Errorf("unexpected nil in config: %s.%s", section, name)
		}
		config = v.Elem().Interface()
	}

	return invokeConstructor(ctor, client, config)
}

func DestroyResource(ctx context.Context, client *databricks.WorkspaceClient, section, id string) error {
	ctor, ok := resourceConstructors[section]
	if !ok {
		return fmt.Errorf("unsupported resource type: %s", section)
	}

	r, err := invokeConstructor(ctor, client, nil) // zero config is enough for deletion
	if err != nil {
		return err
	}

	return r.DoDelete(ctx, id)
}
