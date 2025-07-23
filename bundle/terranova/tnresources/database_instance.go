package tnresources

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/structdiff"
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

func (d ResourceDatabaseInstance) DoUpdate(ctx context.Context, oldID string) (string, error) {
	request := database.UpdateDatabaseInstanceRequest{
		DatabaseInstance: d.config,
	}
	request.DatabaseInstance.Uid = oldID

	response, err := d.client.Database.UpdateDatabaseInstance(ctx, request)
	if err != nil {
		return "", SDKError{Method: "Database.UpdateDatabaseInstance", Err: err}
	}
	return response.Uid, nil
}

func (d ResourceDatabaseInstance) WaitAfterCreate(ctx context.Context) error {
	for {
		resp, err := d.client.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{
			Name: d.config.Name,
		})
		if err != nil {
			return SDKError{Method: "Database.GetDatabaseInstance", Err: err}
		}

		cmdio.LogString(ctx, fmt.Sprintf("Database instance status: %s", resp.State))

		if resp.State == database.DatabaseInstanceStateAvailable {
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}

func (d ResourceDatabaseInstance) WaitAfterUpdate(ctx context.Context) error {
	return nil
}

func (d ResourceDatabaseInstance) ClassifyChanges(changes []structdiff.Change) deployplan.ActionType {
	return deployplan.ActionTypeUpdate
}

func NewResourceDatabaseInstance(client *databricks.WorkspaceClient, resource *resources.DatabaseInstance) (*ResourceDatabaseInstance, error) {
	return &ResourceDatabaseInstance{
		client: client,
		config: resource.DatabaseInstance,
	}, nil
}

func DeleteDatabaseInstance(ctx context.Context, client *databricks.WorkspaceClient, oldName string) error {
	err := client.Database.DeleteDatabaseInstance(ctx, database.DeleteDatabaseInstanceRequest{
		Name: oldName,
	})
	if err != nil {
		return SDKError{Method: "Database.DeleteDatabaseInstance", Err: err}
	}
	return nil
}
