package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"

	"github.com/databricks/cli/libs/log"
)

type Catalog struct {
	BaseResource
	catalog.CreateCatalog

	// List of grants to apply on this catalog.
	Grants []catalog.PrivilegeAssignment `json:"grants,omitempty"`
}

func (c *Catalog) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Catalogs.GetByName(ctx, name)
	if err != nil {
		log.Debugf(ctx, "catalog with name %s does not exist: %v", name, err)

		if apierr.IsMissing(err) {
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
	baseURL.Path = "explore/data/" + strings.ReplaceAll(c.ID, ".", "/")
	c.URL = baseURL.String()
}

func (c *Catalog) GetURL() string {
	return c.URL
}

func (c *Catalog) GetName() string {
	return c.Name
}

func (c *Catalog) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c Catalog) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}
