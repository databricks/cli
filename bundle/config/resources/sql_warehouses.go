package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/warehouses/v1"
	"github.com/databricks/cli/libs/utils"
	sdkwarehouses "github.com/databricks/sdk-go/warehouses/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type SqlWarehouse struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	warehouses.CreateWarehouseRequest

	// Lifecycle shadows BaseResource.Lifecycle to add support for lifecycle.started.
	Lifecycle *LifecycleWithStarted `json:"lifecycle,omitempty"`

	Permissions []SqlWarehousePermission `json:"permissions,omitempty"`
}

// GetLifecycle returns the lifecycle settings, using LifecycleWithStarted.
func (sw *SqlWarehouse) GetLifecycle() LifecycleConfig {
	if sw.Lifecycle == nil {
		return LifecycleWithStarted{}
	}
	return *sw.Lifecycle
}

func (sw *SqlWarehouse) Exists(ctx context.Context, w *sdkclients.Clients, id string) (bool, error) {
	_, err := w.Warehouses.GetWarehouse(ctx, sdkwarehouses.GetWarehouseRequest{Id: new(id)})
	if err != nil {
		log.Debugf(ctx, "sql warehouse %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*SqlWarehouse) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "sql_warehouse",
		PluralName:    "sql_warehouses",
		SingularTitle: "Sql Warehouse",
		PluralTitle:   "SQL Warehouses",
	}
}

func (sw *SqlWarehouse) InitializeURL(baseURL url.URL) {
	if sw.ID == "" {
		return
	}
	sw.URL = workspaceurls.ResourceURL(baseURL, "warehouses", sw.ID)
}

func (sw *SqlWarehouse) GetName() string {
	return utils.Deref(sw.Name)
}
