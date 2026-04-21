package direct

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// Client is the narrow SDK surface the direct engine actually exercises.
// Expressed as an interface so tests can substitute an in-memory fake without
// standing up the full *databricks.WorkspaceClient.
type Client interface {
	GetCatalog(ctx context.Context, name string) (*catalog.CatalogInfo, error)
	CreateCatalog(ctx context.Context, in catalog.CreateCatalog) (*catalog.CatalogInfo, error)
	UpdateCatalog(ctx context.Context, in catalog.UpdateCatalog) (*catalog.CatalogInfo, error)
	DeleteCatalog(ctx context.Context, name string) error

	GetSchema(ctx context.Context, fullName string) (*catalog.SchemaInfo, error)
	CreateSchema(ctx context.Context, in catalog.CreateSchema) (*catalog.SchemaInfo, error)
	UpdateSchema(ctx context.Context, in catalog.UpdateSchema) (*catalog.SchemaInfo, error)
	DeleteSchema(ctx context.Context, fullName string) error

	UpdatePermissions(ctx context.Context, in catalog.UpdatePermissions) error
}

// sdkClient adapts *databricks.WorkspaceClient to the Client interface.
type sdkClient struct{ w *databricks.WorkspaceClient }

// NewClient wraps a workspace client in the narrower Client interface.
func NewClient(w *databricks.WorkspaceClient) Client {
	return &sdkClient{w: w}
}

func (c *sdkClient) GetCatalog(ctx context.Context, name string) (*catalog.CatalogInfo, error) {
	return c.w.Catalogs.GetByName(ctx, name)
}

func (c *sdkClient) CreateCatalog(ctx context.Context, in catalog.CreateCatalog) (*catalog.CatalogInfo, error) {
	return c.w.Catalogs.Create(ctx, in)
}

func (c *sdkClient) UpdateCatalog(ctx context.Context, in catalog.UpdateCatalog) (*catalog.CatalogInfo, error) {
	return c.w.Catalogs.Update(ctx, in)
}

func (c *sdkClient) DeleteCatalog(ctx context.Context, name string) error {
	return c.w.Catalogs.Delete(ctx, catalog.DeleteCatalogRequest{Name: name, Force: true})
}

func (c *sdkClient) GetSchema(ctx context.Context, fullName string) (*catalog.SchemaInfo, error) {
	return c.w.Schemas.GetByFullName(ctx, fullName)
}

func (c *sdkClient) CreateSchema(ctx context.Context, in catalog.CreateSchema) (*catalog.SchemaInfo, error) {
	return c.w.Schemas.Create(ctx, in)
}

func (c *sdkClient) UpdateSchema(ctx context.Context, in catalog.UpdateSchema) (*catalog.SchemaInfo, error) {
	return c.w.Schemas.Update(ctx, in)
}

func (c *sdkClient) DeleteSchema(ctx context.Context, fullName string) error {
	return c.w.Schemas.Delete(ctx, catalog.DeleteSchemaRequest{FullName: fullName, Force: true})
}

func (c *sdkClient) UpdatePermissions(ctx context.Context, in catalog.UpdatePermissions) error {
	_, err := c.w.Grants.Update(ctx, in)
	return err
}
