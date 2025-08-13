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
	config database.DatabaseInstance
}

func (d ResourceDatabaseInstance) Config() any {
	return d.config
}

func (d ResourceDatabaseInstance) DoCreate(ctx context.Context) (string, error) {
	response, err := d.client.Database.CreateDatabaseInstance(ctx, database.CreateDatabaseInstanceRequest{
		DatabaseInstance: d.config,
	})
	if err != nil {
		return "", SDKError{Method: "Database.CreateDatabaseInstance", Err: err}
	}
	return response.Name, nil
}

func (d ResourceDatabaseInstance) DoUpdate(ctx context.Context, id string) error {
	request := database.UpdateDatabaseInstanceRequest{
		DatabaseInstance: d.config,
		Name:             d.config.Name,
		UpdateMask:       "*",
	}
	request.DatabaseInstance.Uid = id

	_, err := d.client.Database.UpdateDatabaseInstance(ctx, request)
	if err != nil {
		return SDKError{Method: "Database.UpdateDatabaseInstance", Err: err}
	}
	return nil
}

func (d ResourceDatabaseInstance) WaitAfterCreate(ctx context.Context) error {
	timeout := 10 * time.Minute
	interval := 10 * time.Second

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		instance, err := d.client.Database.GetDatabaseInstanceByName(ctx, d.config.Name)
		if err != nil {
			return SDKError{Method: "Database.GetDatabaseInstanceByName", Err: err}
		}

		if instance.State == database.DatabaseInstanceStateAvailable {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			continue
		}
	}
}

func (d ResourceDatabaseInstance) WaitAfterUpdate(ctx context.Context) error {
	return nil
}

func NewResourceDatabaseInstance(client *databricks.WorkspaceClient, resource *resources.DatabaseInstance) (*ResourceDatabaseInstance, error) {
	return &ResourceDatabaseInstance{
		client: client,
		config: resource.DatabaseInstance,
	}, nil
}

func DeleteDatabaseInstance(ctx context.Context, client *databricks.WorkspaceClient, name string) error {
	err := client.Database.DeleteDatabaseInstance(ctx, database.DeleteDatabaseInstanceRequest{
		Name:            name,
		Purge:           true,
		Force:           false,
		ForceSendFields: nil,
	})
	if err != nil {
		return SDKError{Method: "Database.DeleteDatabaseInstanceByName", Err: err}
	}
	return nil
}
