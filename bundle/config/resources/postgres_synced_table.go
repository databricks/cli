package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type PostgresSyncedTableConfig struct {
	postgres.SyncedTable_SyncedTableSpec

	// SyncedTableId is the user-specified three-part UC name (catalog.schema.table).
	// Becomes the trailing component of the server-assigned Name:
	// "synced_tables/{synced_table_id}".
	SyncedTableId string `json:"synced_table_id"`
}

type PostgresSyncedTable struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	PostgresSyncedTableConfig
}

func (s *PostgresSyncedTable) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Postgres.GetSyncedTable(ctx, sdkpostgres.GetSyncedTableRequest{Name: new(name)})
	if err != nil {
		log.Debugf(ctx, "postgres synced table %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (s *PostgresSyncedTable) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "postgres_synced_table",
		PluralName:    "postgres_synced_tables",
		SingularTitle: "Postgres synced table",
		PluralTitle:   "Postgres synced tables",
	}
}

func (s *PostgresSyncedTable) GetName() string {
	// Synced tables don't expose a display name distinct from their three-part
	// id, so the id IS the name. Prefer the post-deploy ID
	// ("synced_tables/{catalog}.{schema}.{table}") so bundle summary renders
	// the resolved id even when SyncedTableId still has unresolved
	// cross-resource references like ${resources.X.Y.Z}.
	if id, ok := strings.CutPrefix(s.ID, "synced_tables/"); ok {
		return id
	}
	return s.SyncedTableId
}

func (s *PostgresSyncedTable) InitializeURL(baseURL url.URL) {
	// UC explore expects /{catalog}/{schema}/{table}, so bail if the name isn't
	// a fully resolved three-part identifier; an unresolved ${...} reference
	// would otherwise produce a misleading URL.
	name := s.GetName()
	if strings.Count(name, ".") != 2 {
		return
	}
	s.URL = workspaceurls.ResourceURL(baseURL, "postgres_synced_tables", name)
}
