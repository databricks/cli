package dresources

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// PostgresCatalogRemote is the return type for DoRead. It embeds CatalogCatalogSpec so
// that all paths in StateType are valid paths in RemoteType, enabling drift detection
// for spec fields once the backend echoes spec on GET.
type PostgresCatalogRemote struct {
	postgres.CatalogCatalogSpec

	CatalogId string `json:"catalog_id,omitempty"`

	Name       string                         `json:"name,omitempty"`
	Status     *postgres.CatalogCatalogStatus `json:"status,omitempty"`
	Uid        string                         `json:"uid,omitempty"`
	CreateTime *sdktime.Time                  `json:"create_time,omitempty"`
	UpdateTime *sdktime.Time                  `json:"update_time,omitempty"`
}

// Custom marshaler needed because embedded CatalogCatalogSpec has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *PostgresCatalogRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PostgresCatalogRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

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

func (*ResourcePostgresCatalog) RemapState(remote *PostgresCatalogRemote) *PostgresCatalogState {
	return &PostgresCatalogState{
		CatalogId:          remote.CatalogId,
		CatalogCatalogSpec: remote.CatalogCatalogSpec,
	}
}

// makePostgresCatalogRemote converts the SDK Catalog into the embedded remote shape.
// GET does not echo spec today (only status is returned); the embedded spec fields
// stay at their zero values, and resources.yml suppresses phantom drift via
// ignore_remote_changes with reason spec:input_only.
//
// The user-facing catalog id only appears as the trailing component of
// remote.Name, so we strip the constant "catalogs/" prefix.
func makePostgresCatalogRemote(catalog *postgres.Catalog) *PostgresCatalogRemote {
	var spec postgres.CatalogCatalogSpec
	if catalog.Spec != nil {
		spec = *catalog.Spec
	}
	return &PostgresCatalogRemote{
		CatalogCatalogSpec: spec,
		CatalogId:          strings.TrimPrefix(catalog.Name, "catalogs/"),
		Name:               catalog.Name,
		Status:             catalog.Status,
		Uid:                catalog.Uid,
		CreateTime:         catalog.CreateTime,
		UpdateTime:         catalog.UpdateTime,
	}
}

func (r *ResourcePostgresCatalog) DoRead(ctx context.Context, id string) (*PostgresCatalogRemote, error) {
	catalog, err := r.client.Postgres.GetCatalog(ctx, postgres.GetCatalogRequest{Name: id})
	if err != nil {
		return nil, err
	}
	return makePostgresCatalogRemote(catalog), nil
}

func (r *ResourcePostgresCatalog) DoCreate(ctx context.Context, _ *Engine, config *PostgresCatalogState) (string, *PostgresCatalogRemote, error) {
	waiter, err := r.client.Postgres.CreateCatalog(ctx, postgres.CreateCatalogRequest{
		CatalogId: config.CatalogId,
		Catalog: postgres.Catalog{
			Spec: &config.CatalogCatalogSpec,

			// Output-only fields.
			CatalogId:       "",
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
	// TODO: save state before the wait to prevent orphaning on interruption.
	// waiter.Name() returns the LRO operation name (e.g. .../operations/UUID),
	// not the real resource name. We need the resource name to save a valid state
	// entry; options: (1) derive it from input (Parent + resource-type + Id),
	// (2) call waiter.Metadata() if it exposes the resource name early.

	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}
	remote := makePostgresCatalogRemote(result)
	return remote.Name, remote, nil
}

func (r *ResourcePostgresCatalog) DoDelete(ctx context.Context, id string, _ *PostgresCatalogState) error {
	waiter, err := r.client.Postgres.DeleteCatalog(ctx, postgres.DeleteCatalogRequest{
		Name: id,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
