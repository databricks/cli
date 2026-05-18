package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type PostgresSyncedTableConfig struct {
	postgres.SyncedTableSyncedTableSpec

	// SyncedTableId is the user-specified three-part UC name (catalog.schema.table).
	// Becomes the trailing component of the server-assigned Name:
	// "synced_tables/{synced_table_id}".
	SyncedTableId string `json:"synced_table_id"`
}

func (c *PostgresSyncedTableConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c *PostgresSyncedTableConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type PostgresSyncedTable struct {
	BaseResource
	PostgresSyncedTableConfig
}

func (s *PostgresSyncedTable) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Postgres.GetSyncedTable(ctx, postgres.GetSyncedTableRequest{Name: name})
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
	// Synced tables don't expose a display name distinct from their three-part id.
	return s.SyncedTableId
}

func (s *PostgresSyncedTable) GetURL() string {
	return s.URL
}

func (s *PostgresSyncedTable) InitializeURL(baseURL url.URL) {
	if s.SyncedTableId == "" {
		return
	}
	baseURL.Path = "explore/data/" + s.SyncedTableId
	s.URL = baseURL.String()
}
