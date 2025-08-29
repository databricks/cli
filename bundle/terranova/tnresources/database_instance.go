package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type ResourceDatabaseInstance struct {
	client *databricks.WorkspaceClient
	config database.DatabaseInstance
	waiter *database.WaitGetDatabaseInstanceDatabaseAvailable[database.DatabaseInstance]
}

func (d ResourceDatabaseInstance) Config() any {
	return d.config
}

func (d *ResourceDatabaseInstance) DoRefresh(ctx context.Context, id string) (any, error) {
	return d.client.Database.GetDatabaseInstanceByName(ctx, id)
}

func (d *ResourceDatabaseInstance) DoCreate(ctx context.Context) (string, any, error) {
	waiter, err := d.client.Database.CreateDatabaseInstance(ctx, database.CreateDatabaseInstanceRequest{
		DatabaseInstance: d.config,
	})
	if err != nil {
		return "", nil, err
	}
	d.waiter = waiter
	return waiter.Response.Name, waiter.Response, nil
}

func (d ResourceDatabaseInstance) DoUpdate(ctx context.Context, id string) (any, error) {
	request := database.UpdateDatabaseInstanceRequest{
		DatabaseInstance: d.config,
		Name:             d.config.Name,
		UpdateMask:       "*",
	}
	request.DatabaseInstance.Uid = id

	response, err := d.client.Database.UpdateDatabaseInstance(ctx, request)
	return response, err
}

func (d *ResourceDatabaseInstance) WaitAfterCreate(ctx context.Context) (any, error) {
	if d.waiter == nil {
		return nil, nil
	}
	response, err := d.waiter.Get()
	return response, err
}

func (d *ResourceDatabaseInstance) WaitAfterUpdate(ctx context.Context) (any, error) {
	// Intentional no-op
	return nil, nil
}

func NewResourceDatabaseInstance(client *databricks.WorkspaceClient, resource *resources.DatabaseInstance) (*ResourceDatabaseInstance, error) {
	return &ResourceDatabaseInstance{
		client: client,
		config: resource.DatabaseInstance,
		waiter: nil,
	}, nil
}

func DeleteDatabaseInstance(ctx context.Context, client *databricks.WorkspaceClient, name string) error {
	return client.Database.DeleteDatabaseInstance(ctx, database.DeleteDatabaseInstanceRequest{
		Name:            name,
		Purge:           true,
		Force:           false,
		ForceSendFields: nil,
	})
}
