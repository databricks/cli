package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type ResourceApp struct {
	client *databricks.WorkspaceClient
	config apps.App
}

func NewResourceApp(client *databricks.WorkspaceClient, config *resources.App) (*ResourceApp, error) {
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
		App:             r.config,
		NoCompute:       true,
		ForceSendFields: nil,
	}
	waiter, err := r.client.Apps.Create(ctx, request)
	if err != nil {
		return "", err
	}

	// TODO: Store waiter for Wait method

	return waiter.Response.Name, nil
}

func (r *ResourceApp) DoUpdate(ctx context.Context, id string) error {
	request := apps.UpdateAppRequest{
		App:  r.config,
		Name: id,
	}
	response, err := r.client.Apps.Update(ctx, request)
	if err != nil {
		return err
	}

	if response.Name != id {
		log.Warnf(ctx, "apps: response contains unexpected name=%#v (expected %#v)", response.Name, id)
	}

	return nil
}

func DeleteApp(ctx context.Context, client *databricks.WorkspaceClient, id string) error {
	_, err := client.Apps.DeleteByName(ctx, id)
	return err
}

func (r *ResourceApp) WaitAfterCreate(ctx context.Context) error {
	// Intentional no-op
	return nil
}

func (r *ResourceApp) WaitAfterUpdate(ctx context.Context) error {
	// Intentional no-op
	return nil
}
