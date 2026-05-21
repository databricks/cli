package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type PostgresCatalogConfig struct {
	postgres.CatalogCatalogSpec

	// CatalogId is the user-specified UC catalog name. Becomes the trailing
	// component of the server-assigned Name: "catalogs/{catalog_id}".
	CatalogId string `json:"catalog_id"`
}

func (c *PostgresCatalogConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c *PostgresCatalogConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type PostgresCatalog struct {
	BaseResource
	PostgresCatalogConfig
}

func (c *PostgresCatalog) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Postgres.GetCatalog(ctx, postgres.GetCatalogRequest{Name: name})
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

func (c *PostgresCatalog) GetURL() string {
	return c.URL
}

func (c *PostgresCatalog) InitializeURL(baseURL url.URL) {
	if c.CatalogId == "" {
		return
	}
	baseURL.Path = "explore/data/" + c.CatalogId
	c.URL = baseURL.String()
}
