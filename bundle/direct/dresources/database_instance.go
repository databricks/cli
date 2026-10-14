package dresources

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/internal/retries"
	"github.com/databricks/cli/libs/sdkclients"
	database "github.com/databricks/cli/libs/sdktypes/database/v1"
	"github.com/databricks/cli/libs/utils"
	sdkdatabase "github.com/databricks/sdk-go/database/v1"
)

type ResourceDatabaseInstance struct {
	client *sdkclients.Clients
}

func (*ResourceDatabaseInstance) New(client *sdkclients.Clients) *ResourceDatabaseInstance {
	return &ResourceDatabaseInstance{client: client}
}

func (*ResourceDatabaseInstance) PrepareState(input *resources.DatabaseInstance) *database.DatabaseInstance {
	return &input.DatabaseInstance
}

func (d *ResourceDatabaseInstance) DoRead(ctx context.Context, id string) (*database.DatabaseInstance, error) {
	instance, err := d.client.Database.GetDatabaseInstance(ctx, sdkdatabase.GetDatabaseInstanceRequest{Name: new(id)})
	if err != nil {
		return nil, err
	}
	return database.DatabaseInstanceFromSDK(instance)
}

func (d *ResourceDatabaseInstance) DoCreate(ctx context.Context, config *database.DatabaseInstance) (string, *database.DatabaseInstance, error) {
	request, err := database.CreateDatabaseInstanceRequestToSDK(&database.CreateDatabaseInstanceRequest{
		DatabaseInstance: config,
	})
	if err != nil {
		return "", nil, err
	}
	waiter, err := d.client.Database.CreateDatabaseInstance(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	return waiter.GetName(), nil, nil
}

func (d *ResourceDatabaseInstance) DoUpdate(ctx context.Context, id string, config *database.DatabaseInstance, _ *PlanEntry) (*database.DatabaseInstance, error) {
	instance := *config
	instance.Uid = new(id)
	request, err := database.UpdateDatabaseInstanceRequestToSDK(&database.UpdateDatabaseInstanceRequest{
		DatabaseInstance: &instance,
		UpdateMask:       new("*"),
	})
	if err != nil {
		return nil, err
	}
	_, err = d.client.Database.UpdateDatabaseInstance(ctx, *request)
	return nil, err
}

func (d *ResourceDatabaseInstance) WaitAfterCreate(ctx context.Context, id string, config *database.DatabaseInstance) (*database.DatabaseInstance, error) {
	name := utils.Deref(config.Name)
	// _ is remoteState, should we return it here?
	_, err := retries.Poll(ctx, 20*time.Minute, func() (*sdkdatabase.DatabaseInstance, *retries.Err) {
		instance, err := d.client.Database.GetDatabaseInstance(ctx, sdkdatabase.GetDatabaseInstanceRequest{Name: new(name)})
		if err != nil {
			return nil, retries.Halt(err)
		}
		if instance.State == sdkdatabase.DatabaseInstance_State_Available {
			return instance, nil
		}
		return nil, retries.Continues(fmt.Sprintf("current status: %s", instance.State))
	})
	return nil, err
}

func (d *ResourceDatabaseInstance) DoDelete(ctx context.Context, name string, _ *database.DatabaseInstance) error {
	return d.client.Database.DeleteDatabaseInstance(ctx, sdkdatabase.DeleteDatabaseInstanceRequest{
		Name:  new(name),
		Purge: new(true),
	})
}
