package dresources

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

func (*ResourceDatabaseInstance) PrepareState(input *resources.DatabaseInstance) *database.DatabaseInstance {
	return &input.DatabaseInstance
}

func (d *ResourceDatabaseInstance) DoRead(ctx context.Context, id string) (*database.DatabaseInstance, error) {
	return d.client.Database.GetDatabaseInstanceByName(ctx, id)
}

func (d *ResourceDatabaseInstance) DoCreate(ctx context.Context, engine *Engine, config *database.DatabaseInstance) (string, *database.DatabaseInstance, error) {
	waiter, err := d.client.Database.CreateDatabaseInstance(ctx, database.CreateDatabaseInstanceRequest{
		DatabaseInstance: *config,
	})
	if err != nil {
		return "", nil, err
	}
	id := waiter.Response.Name

	// Save state immediately after the instance is created so it is not orphaned
	// if the subsequent wait is interrupted.
	engine.SetID(id)
	if err := engine.SaveState(config); err != nil {
		return "", nil, err
	}

	waiterObj := &database.WaitGetDatabaseInstanceDatabaseAvailable[database.DatabaseInstance]{
		Response: config,
		Name:     config.Name,
		Poll: func(timeout time.Duration, callback func(*database.DatabaseInstance)) (*database.DatabaseInstance, error) {
			return d.client.Database.WaitGetDatabaseInstanceDatabaseAvailable(ctx, config.Name, timeout, callback)
		},
	}
	// _ is remoteState, should we return it here?
	_, err = waiterObj.GetWithTimeout(20 * time.Minute)
	return id, nil, err
}

func (d *ResourceDatabaseInstance) DoUpdate(ctx context.Context, _ *Engine, id string, config *database.DatabaseInstance, _ *PlanEntry) (*database.DatabaseInstance, error) {
	request := database.UpdateDatabaseInstanceRequest{
		DatabaseInstance: *config,
		Name:             config.Name,
		UpdateMask:       "*",
	}
	request.DatabaseInstance.Uid = id
	_, err := d.client.Database.UpdateDatabaseInstance(ctx, request)
	return nil, err
}

func (d *ResourceDatabaseInstance) DoDelete(ctx context.Context, name string, _ *database.DatabaseInstance) error {
	return d.client.Database.DeleteDatabaseInstance(ctx, database.DeleteDatabaseInstanceRequest{
		Name:            name,
		Purge:           true,
		Force:           false,
		ForceSendFields: nil,
	})
}
