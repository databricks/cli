package dresources

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	postgres "github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/core/types"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
)

// PostgresCatalogRemote is the return type for DoRead. It embeds Catalog_CatalogSpec so
// that all paths in StateType are valid paths in RemoteType, enabling drift detection
// for spec fields once the backend echoes spec on GET.
type PostgresCatalogRemote struct {
	postgres.Catalog_CatalogSpec

	CatalogId string `json:"catalog_id,omitempty"`

	Name       string                          `json:"name,omitempty"`
	Status     *postgres.Catalog_CatalogStatus `json:"status,omitempty"`
	Uid        string                          `json:"uid,omitempty"`
	CreateTime *types.Time                     `json:"create_time,omitempty"`
	UpdateTime *types.Time                     `json:"update_time,omitempty"`
}

type ResourcePostgresCatalog struct {
	client *sdkclients.Clients
}

type PostgresCatalogState = resources.PostgresCatalogConfig

func (*ResourcePostgresCatalog) New(client *sdkclients.Clients) *ResourcePostgresCatalog {
	return &ResourcePostgresCatalog{client: client}
}

func (*ResourcePostgresCatalog) PrepareState(input *resources.PostgresCatalog) *PostgresCatalogState {
	return &PostgresCatalogState{
		CatalogId:           input.CatalogId,
		Catalog_CatalogSpec: input.Catalog_CatalogSpec,
	}
}

func (*ResourcePostgresCatalog) RemapState(remote *PostgresCatalogRemote) *PostgresCatalogState {
	return &PostgresCatalogState{
		CatalogId:           remote.CatalogId,
		Catalog_CatalogSpec: remote.Catalog_CatalogSpec,
	}
}

// makePostgresCatalogRemote converts the SDK Catalog into the embedded remote shape.
// GET does not echo spec today (only status is returned); the embedded spec fields
// stay at their zero values, and postgres_catalogs.yml suppresses phantom drift via
// ignore_remote_changes with reason spec:input_only.
//
// The user-facing catalog id only appears as the trailing component of
// remote.Name, so we strip the constant "catalogs/" prefix.
func makePostgresCatalogRemote(resp *sdkpostgres.Catalog) (*PostgresCatalogRemote, error) {
	catalog, err := postgres.CatalogFromSDK(resp)
	if err != nil {
		return nil, err
	}
	var spec postgres.Catalog_CatalogSpec
	if catalog.Spec != nil {
		spec = *catalog.Spec
	}
	return &PostgresCatalogRemote{
		Catalog_CatalogSpec: spec,
		CatalogId:           strings.TrimPrefix(utils.Deref(catalog.Name), "catalogs/"),
		Name:                utils.Deref(catalog.Name),
		Status:              catalog.Status,
		Uid:                 utils.Deref(catalog.Uid),
		CreateTime:          catalog.CreateTime,
		UpdateTime:          catalog.UpdateTime,
	}, nil
}

func (r *ResourcePostgresCatalog) DoRead(ctx context.Context, id string) (*PostgresCatalogRemote, error) {
	catalog, err := r.client.Postgres.GetCatalog(ctx, sdkpostgres.GetCatalogRequest{Name: new(id)})
	if err != nil {
		return nil, err
	}
	return makePostgresCatalogRemote(catalog)
}

func (r *ResourcePostgresCatalog) DoCreate(ctx context.Context, config *PostgresCatalogState) (string, *PostgresCatalogRemote, error) {
	request, err := postgres.CreateCatalogRequestToSDK(&postgres.CreateCatalogRequest{
		CatalogId: nonEmpty(config.CatalogId),
		Catalog:   &postgres.Catalog{Spec: &config.Catalog_CatalogSpec},
	})
	if err != nil {
		return "", nil, err
	}
	waiter, err := r.client.Postgres.CreateCatalog(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}
	remote, err := makePostgresCatalogRemote(result)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

func (r *ResourcePostgresCatalog) DoDelete(ctx context.Context, id string, _ *PostgresCatalogState) error {
	waiter, err := r.client.Postgres.DeleteCatalog(ctx, sdkpostgres.DeleteCatalogRequest{Name: new(id)})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
