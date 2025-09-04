package tnresources

import (
	"context"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type ResourceDatabaseInstance struct {
	client *databricks.WorkspaceClient
}

func (*ResourceDatabaseInstance) New(client *databricks.WorkspaceClient) *ResourceDatabaseInstance {
	return &ResourceDatabaseInstance{client: client}
}

func (*ResourceDatabaseInstance) PrepareConfig(input *resources.DatabaseInstance) *database.DatabaseInstance {
	return &input.DatabaseInstance
}

func (d *ResourceDatabaseInstance) DoCreate(ctx context.Context, config *database.DatabaseInstance) (string, error) {
	waiter, err := d.client.Database.CreateDatabaseInstance(ctx, database.CreateDatabaseInstanceRequest{
		DatabaseInstance: *config,
	})
	if err != nil {
		return "", err
	}
	return waiter.Response.Name, nil
}

func (d *ResourceDatabaseInstance) DoUpdate(ctx context.Context, id string, config *database.DatabaseInstance) error {
	request := database.UpdateDatabaseInstanceRequest{
		DatabaseInstance: *config,
		Name:             config.Name,
		UpdateMask:       "*",
	}
	request.DatabaseInstance.Uid = id
	_, err := d.client.Database.UpdateDatabaseInstance(ctx, request)
	return err
}

func (d *ResourceDatabaseInstance) WaitAfterCreate(ctx context.Context, config *database.DatabaseInstance) error {
	waiter := &database.WaitGetDatabaseInstanceDatabaseAvailable[database.DatabaseInstance]{
		Response: config,
		Name:     config.Name,
		Poll: func(timeout time.Duration, callback func(*database.DatabaseInstance)) (*database.DatabaseInstance, error) {
			return d.client.Database.WaitGetDatabaseInstanceDatabaseAvailable(ctx, config.Name, timeout, callback)
		},
	}
	_, err := waiter.GetWithTimeout(20 * time.Minute)
	return err
}

func (d *ResourceDatabaseInstance) DoDelete(ctx context.Context, name string) error {
	return d.client.Database.DeleteDatabaseInstance(ctx, database.DeleteDatabaseInstanceRequest{
		Name:            name,
		Purge:           true,
		Force:           false,
		ForceSendFields: nil,
	})
}
