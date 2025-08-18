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

func (d *ResourceDatabaseInstance) DoCreate(ctx context.Context) (string, error) {
	waiter, err := d.client.Database.CreateDatabaseInstance(ctx, database.CreateDatabaseInstanceRequest{
		DatabaseInstance: d.config,
	})
	if err != nil {
		return "", err
	}
	d.waiter = waiter
	return waiter.Response.Name, nil
}

func (d ResourceDatabaseInstance) DoUpdate(ctx context.Context, id string) error {
	request := database.UpdateDatabaseInstanceRequest{
		DatabaseInstance: d.config,
		Name:             d.config.Name,
		UpdateMask:       "*",
	}
	request.DatabaseInstance.Uid = id

	_, err := d.client.Database.UpdateDatabaseInstance(ctx, request)
	return err
}

func (d *ResourceDatabaseInstance) WaitAfterCreate(ctx context.Context) error {
	if d.waiter == nil {
		return nil
	}
	_, err := d.waiter.Get()
	return err
}

func (d ResourceDatabaseInstance) WaitAfterUpdate(ctx context.Context) error {
	return nil
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
