package tnresources

import (
	"context"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/config/resources"
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

func New(client *databricks.WorkspaceClient, section, name string, config any) (IResource, error) {
	switch section {
	case "jobs":
		typedConfig, ok := config.(*resources.Job)
		if !ok {
			return nil, fmt.Errorf("unexpected config type for jobs: %T", config)
		}
		if typedConfig == nil {
			return nil, fmt.Errorf("unexpected nil in config: %s.%s", section, name)
		}
		return NewResourceJob(client, *typedConfig)

	case "pipelines":
		typedConfig, ok := config.(*resources.Pipeline)
		if !ok {
			return nil, fmt.Errorf("unexpected config type for pipelines: %T", config)
		}
		if typedConfig == nil {
			return nil, fmt.Errorf("unexpected nil in config: %s.%s", section, name)
		}
		return NewResourcePipeline(client, *typedConfig)

	case "schemas":
		typedConfig, ok := config.(*resources.Schema)
		if !ok {
			return nil, fmt.Errorf("unexpected config type for schemas: %T", config)
		}
		if typedConfig == nil {
			return nil, fmt.Errorf("unexpected nil in config: %s.%s", section, name)
		}
		return NewResourceSchema(client, *typedConfig)

	case "apps":
		typedConfig, ok := config.(*resources.App)
		if !ok {
			return nil, fmt.Errorf("unexpected config type for apps: %T", config)
		}
		if typedConfig == nil {
			return nil, fmt.Errorf("unexpected nil in config: %s.%s", section, name)
		}
		return NewResourceApp(client, *typedConfig)

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", section)
	}
}

func DestroyResource(ctx context.Context, client *databricks.WorkspaceClient, section, id string) error {
	var err error
	var r IResource

	switch section {
	case "jobs":
		r, err = NewResourceJob(client, resources.Job{})
	case "pipelines":
		r, err = NewResourcePipeline(client, resources.Pipeline{})
	case "schemas":
		r, err = NewResourceSchema(client, resources.Schema{})
	case "apps":
		r, err = NewResourceApp(client, resources.App{})
	default:
		return fmt.Errorf("unsupported resource type: %s", section)
	}

	if err != nil {
		return err
	}

	return r.DoDelete(ctx, id)
}
