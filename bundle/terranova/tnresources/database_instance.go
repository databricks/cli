package tnresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
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
	return response.Uid, nil
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
	return nil
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

func DeleteDatabaseInstance(ctx context.Context, client *databricks.WorkspaceClient, oldID string) error {
	err := client.Database.DeleteDatabaseInstanceByName(ctx, oldID)
	if err != nil {
		return SDKError{Method: "Database.DeleteDatabaseInstanceByName", Err: err}
	}
	return nil
}
