package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/database/v1"
	"github.com/databricks/cli/libs/utils"
	sdkdatabase "github.com/databricks/sdk-go/database/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type DatabaseInstance struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	database.DatabaseInstance

	Permissions []Permission `json:"permissions,omitempty"`
}

func (d *DatabaseInstance) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Database.GetDatabaseInstance(ctx, sdkdatabase.GetDatabaseInstanceRequest{Name: new(name)})
	if err != nil {
		log.Debugf(ctx, "database instance %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (d *DatabaseInstance) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "database_instance",
		PluralName:    "database_instances",
		SingularTitle: "Database instance",
		PluralTitle:   "Database instances",
	}
}

func (d *DatabaseInstance) GetName() string {
	return utils.Deref(d.Name)
}

func (d *DatabaseInstance) InitializeURL(baseURL url.URL) {
	if d.ModifiedStatus == ModifiedStatusCreated {
		return
	}
	if d.Name == nil {
		return
	}
	d.URL = workspaceurls.ResourceURL(baseURL, "database_instances", *d.Name)
}
