package tnresources

import (
	"context"
	"reflect"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type ResourceApp struct {
	client *databricks.WorkspaceClient
	config apps.App
}

func NewResourceApp(client *databricks.WorkspaceClient, config resources.App) (*ResourceApp, error) {
	return &ResourceApp{
		client: client,
		config: config.App,
	}, nil
}

func (r *ResourceApp) Config() any {
	return r.config
}

func (r *ResourceApp) DoCreate(ctx context.Context) (string, error) {
	request := apps.CreateAppRequest{
		App:       r.config,
		NoCompute: true,
	}
	waiter, err := r.client.Apps.Create(ctx, request)
	if err != nil {
		return "", SDKError{Method: "Apps.Create", Err: err}
	}

	// TODO: Store waiter for Wait method

	return waiter.Response.Name, nil
}

func (r *ResourceApp) DoUpdate(ctx context.Context, id string) (string, error) {
	request := apps.UpdateAppRequest{
		App:  r.config,
		Name: id,
	}
	response, err := r.client.Apps.Update(ctx, request)
	if err != nil {
		return "", SDKError{Method: "Apps.Update", Err: err}
	}

	return response.Name, nil
}

func (r *ResourceApp) DoDelete(ctx context.Context, id string) error {
	return nil
}

func (r *ResourceApp) WaitAfterCreate(ctx context.Context) error {
	return nil
}

func (r *ResourceApp) WaitAfterUpdate(ctx context.Context) error {
	return nil
}

func (r *ResourceApp) ClassifyChanges(changes []structdiff.Change) ChangeType {
	return ChangeTypeUpdate
}

var appType = reflect.TypeOf(ResourceApp{}.config)

func (r *ResourceApp) GetType() reflect.Type {
	return appType
}
