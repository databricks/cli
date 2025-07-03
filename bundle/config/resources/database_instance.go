package resources

import (
	"context"
	"github.com/databricks/databricks-sdk-go"
	"net/url"
)

type DatabaseInstance struct {
	// A unique name to identify the database instance.
	Name string `json:"name"`
}

func (d DatabaseInstance) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	//TODO implement me
	panic("implement me: Exists")
}

func (d DatabaseInstance) ResourceDescription() ResourceDescription {
	//TODO implement me
	panic("implement me: ResourceDescription")
}

func (d DatabaseInstance) GetName() string {
	return d.Name
}

func (d DatabaseInstance) GetURL() string {
	//TODO implement me
	panic("implement me: GetURL")
}

func (d DatabaseInstance) InitializeURL(baseURL url.URL) {
	//TODO implement me
	panic("implement me: InitializeURL")
}
