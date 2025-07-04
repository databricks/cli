package resources

import (
	"context"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
	"net/url"
)

type DatabaseInstance struct {
	URL string `json:"url,omitempty" bundle:"internal"`

	database.DatabaseInstance
}

func (d DatabaseInstance) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	//TODO implement me
	panic("implement me: Exists")
}

func (d DatabaseInstance) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "database instance",
		PluralName:    "database instances",
		SingularTitle: "Database instance",
		PluralTitle:   "Database instances",
	}
}

func (d DatabaseInstance) GetName() string {
	return d.Name
}

func (d DatabaseInstance) GetURL() string {
	return d.URL
}

func (d DatabaseInstance) InitializeURL(baseURL url.URL) {
	if d.Name == "" {
		return
	}
	baseURL.Path = "compute/database-instances/" + d.Name
	d.URL = baseURL.String()
}
