package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type PostgresCatalogConfig struct {
	postgres.Catalog_CatalogSpec

	// CatalogId is the user-specified UC catalog name. Becomes the trailing
	// component of the server-assigned Name: "catalogs/{catalog_id}".
	CatalogId string `json:"catalog_id"`
}

type PostgresCatalog struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	PostgresCatalogConfig
}

func (c *PostgresCatalog) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Postgres.GetCatalog(ctx, sdkpostgres.GetCatalogRequest{Name: new(name)})
	if err != nil {
		log.Debugf(ctx, "postgres catalog %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (c *PostgresCatalog) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "postgres_catalog",
		PluralName:    "postgres_catalogs",
		SingularTitle: "Postgres catalog",
		PluralTitle:   "Postgres catalogs",
	}
}

func (c *PostgresCatalog) GetName() string {
	return c.CatalogId
}

func (c *PostgresCatalog) InitializeURL(baseURL url.URL) {
	if c.CatalogId == "" {
		return
	}
	c.URL = workspaceurls.ResourceURL(baseURL, "postgres_catalogs", c.CatalogId)
}
