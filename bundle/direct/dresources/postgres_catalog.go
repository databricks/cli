package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type ResourcePostgresCatalog struct {
	client *databricks.WorkspaceClient
}

type PostgresCatalogState = resources.PostgresCatalogConfig

func (*ResourcePostgresCatalog) New(client *databricks.WorkspaceClient) *ResourcePostgresCatalog {
	return &ResourcePostgresCatalog{client: client}
}

func (*ResourcePostgresCatalog) PrepareState(input *resources.PostgresCatalog) *PostgresCatalogState {
	return &PostgresCatalogState{
		CatalogId:          input.CatalogId,
		CatalogCatalogSpec: input.CatalogCatalogSpec,
	}
}

func (*ResourcePostgresCatalog) RemapState(remote *postgres.Catalog) *PostgresCatalogState {
	// The server-side ID is the full hierarchical name `catalogs/<catalog_id>`.
	// Keep it as-is — the `catalogs/` prefix is an inherent part of the ID, not
	// noise to strip.
	//
	// GET does not return the spec today (only status). Return an empty spec
	// and rely on the ignore_remote_changes / input_only classifications in
	// resources.yml to suppress phantom drift until the backend starts
	// echoing spec values on GET.
	return &PostgresCatalogState{
		CatalogId: remote.Name,
		CatalogCatalogSpec: postgres.CatalogCatalogSpec{
			Branch:                  "",
			CreateDatabaseIfMissing: false,
			PostgresDatabase:        "",
			ForceSendFields:         nil,
		},
	}
}

func (r *ResourcePostgresCatalog) DoRead(ctx context.Context, id string) (*postgres.Catalog, error) {
	return r.client.Postgres.GetCatalog(ctx, postgres.GetCatalogRequest{Name: id})
}

func (r *ResourcePostgresCatalog) DoCreate(ctx context.Context, config *PostgresCatalogState) (string, *postgres.Catalog, error) {
	waiter, err := r.client.Postgres.CreateCatalog(ctx, postgres.CreateCatalogRequest{
		CatalogId: config.CatalogId,
		Catalog: postgres.Catalog{
			Spec: &config.CatalogCatalogSpec,

			// Output-only fields.
			CreateTime:      nil,
			Name:            "",
			Status:          nil,
			Uid:             "",
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
	})
	if err != nil {
		return "", nil, err
	}

	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}
	return result.Name, result, nil
}

func (r *ResourcePostgresCatalog) DoDelete(ctx context.Context, id string) error {
	waiter, err := r.client.Postgres.DeleteCatalog(ctx, postgres.DeleteCatalogRequest{
		Name: id,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
