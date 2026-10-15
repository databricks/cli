package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	"github.com/databricks/cli/libs/utils"
	sdkcatalogs "github.com/databricks/sdk-go/uc/catalogs/v1"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type Catalog struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	CreateCatalog

	// List of grants to apply on this catalog.
	Grants []grants.PrivilegeAssignment `json:"grants,omitempty"`
}

func (c *Catalog) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Catalogs.GetCatalog(ctx, sdkcatalogs.GetCatalogRequest{NameArg: new(name)})
	if err != nil {
		log.Debugf(ctx, "catalog with name %s does not exist: %v", name, err)

		if apierrs.IsMissing(err) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (*Catalog) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "catalog",
		PluralName:    "catalogs",
		SingularTitle: "Catalog",
		PluralTitle:   "Catalogs",
	}
}

func (c *Catalog) InitializeURL(baseURL url.URL) {
	if c.ID == "" {
		return
	}
	c.URL = workspaceurls.ResourceURL(baseURL, "catalogs", c.ID)
}

func (c *Catalog) GetName() string {
	return utils.Deref(c.Name)
}
