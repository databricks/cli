package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type DatabaseInstance struct {
	BaseResource
	database.DatabaseInstance

	Permissions []DatabaseInstancePermission `json:"permissions,omitempty"`
}

func (d *DatabaseInstance) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{Name: name})
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
	return d.Name
}

func (d *DatabaseInstance) GetURL() string {
	return d.URL
}

func (d *DatabaseInstance) InitializeURL(baseURL url.URL) {
	if d.ModifiedStatus == ModifiedStatusCreated {
		return
	}
	if d.Name == "" {
		return
	}
	baseURL.Path = "compute/database-instances/" + d.Name
	d.URL = baseURL.String()
}
