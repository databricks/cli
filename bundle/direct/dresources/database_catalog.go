package dresources

import (
	"context"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type ResourceDatabaseCatalog struct {
	client *databricks.WorkspaceClient
}

func (*ResourceDatabaseCatalog) New(client *databricks.WorkspaceClient) *ResourceDatabaseCatalog {
	return &ResourceDatabaseCatalog{client: client}
}

func (*ResourceDatabaseCatalog) PrepareState(input *resources.DatabaseCatalog) *database.DatabaseCatalog {
	return &input.DatabaseCatalog
}

func (r *ResourceDatabaseCatalog) DoRead(ctx context.Context, id string) (*database.DatabaseCatalog, error) {
	return r.client.Database.GetDatabaseCatalogByName(ctx, id)
}

func (r *ResourceDatabaseCatalog) DoCreate(ctx context.Context, config *database.DatabaseCatalog) (string, *database.DatabaseCatalog, error) {
	create := func(ctx context.Context) (*foundDatabaseCatalog, error) {
		result, err := r.client.Database.CreateDatabaseCatalog(ctx, database.CreateDatabaseCatalogRequest{
			Catalog: *config,
		})
		if err != nil {
			return nil, err
		}
		return &foundDatabaseCatalog{catalog: result, createdAt: 0, createdBy: ""}, nil
	}
	start := time.Now()
	found, err := create(ctx)
	if err != nil {
		found, err = r.recoverFailedCreate(ctx, config.Name, start, create, err)
		if err != nil {
			return "", nil, err
		}
	}
	return found.catalog.Name, nil, nil
}

// foundDatabaseCatalog is a database catalog with the create time and creator of the Unity Catalog
// catalog it registers, which the Database API does not return. createdAt and createdBy are only
// set on a catalog that the lookup after a failed create found.
type foundDatabaseCatalog struct {
	catalog   *database.DatabaseCatalog
	createdAt int64
	createdBy string
}

// recoverFailedCreate handles a create request that failed; see recoverCreate. Catalog creates
// are answered with 504 DEADLINE_EXCEEDED now and then (observed on Azure, 2026-11), and the
// request may have created the catalog anyway. The catalog name is unique in the metastore, so it
// names the catalog to look up.
func (r *ResourceDatabaseCatalog) recoverFailedCreate(ctx context.Context, name string, start time.Time, create func(context.Context) (*foundDatabaseCatalog, error), createErr error) (*foundDatabaseCatalog, error) {
	return recoverCreate(ctx, createRecovery[foundDatabaseCatalog, *foundDatabaseCatalog]{
		name:  "database catalog " + name,
		start: start,
		get: func(ctx context.Context) (*foundDatabaseCatalog, error) {
			dbCatalog, err := r.client.Database.GetDatabaseCatalogByName(ctx, name)
			if err != nil {
				return nil, err
			}
			ucCatalog, err := r.client.Catalogs.GetByName(ctx, name)
			if apierr.IsMissing(err) {
				// The database catalog exists, so repeating the create would only fail. Without
				// a create time it is not known to be ours, which recoverCreate reports.
				return &foundDatabaseCatalog{catalog: dbCatalog, createdAt: 0, createdBy: ""}, nil
			}
			if err != nil {
				return nil, err
			}
			return &foundDatabaseCatalog{catalog: dbCatalog, createdAt: ucCatalog.CreatedAt, createdBy: ucCatalog.CreatedBy}, nil
		},
		createTime:    func(found *foundDatabaseCatalog) (time.Time, bool) { return unixMilliTime(found.createdAt) },
		alreadyExists: isAlreadyExists,
		createdByCaller: func(ctx context.Context, found *foundDatabaseCatalog) (bool, error) {
			return createdByCurrentUser(ctx, r.client, found.createdBy)
		},
		use:    useFound[foundDatabaseCatalog],
		create: repeatCreate(create),
	}, createErr)
}

func (r *ResourceDatabaseCatalog) DoDelete(ctx context.Context, id string, _ *database.DatabaseCatalog) error {
	return r.client.Database.DeleteDatabaseCatalog(ctx, database.DeleteDatabaseCatalogRequest{
		Name: id,
	})
}
