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

type DatabaseCatalog struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	database.DatabaseCatalog
}

func (d *DatabaseCatalog) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Database.GetDatabaseCatalog(ctx, sdkdatabase.GetDatabaseCatalogRequest{Name: new(name)})
	if err != nil {
		log.Debugf(ctx, "database Catalog %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (d *DatabaseCatalog) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "database_catalog",
		PluralName:    "database_catalogs",
		SingularTitle: "Database catalog",
		PluralTitle:   "Database catalogs",
	}
}

func (d *DatabaseCatalog) GetName() string {
	return utils.Deref(d.Name)
}

func (d *DatabaseCatalog) InitializeURL(baseURL url.URL) {
	if d.Name == nil {
		return
	}
	d.URL = workspaceurls.ResourceURL(baseURL, "database_catalogs", *d.Name)
}
