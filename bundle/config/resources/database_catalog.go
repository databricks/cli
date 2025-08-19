package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type DatabaseCatalog struct {
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	URL            string         `json:"url,omitempty" bundle:"internal"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`

	database.DatabaseCatalog
}

func (d *DatabaseCatalog) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Database.GetDatabaseCatalog(ctx, database.GetDatabaseCatalogRequest{Name: name})
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
	return d.Name
}

func (d *DatabaseCatalog) GetURL() string {
	return d.URL
}

func (d *DatabaseCatalog) InitializeURL(baseURL url.URL) {
	if d.Name == "" {
		return
	}
	baseURL.Path = "explore/data/" + d.Name
	d.URL = baseURL.String()
}
