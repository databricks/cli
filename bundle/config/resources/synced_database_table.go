package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type SyncedDatabaseTable struct {
	BaseResource
	database.SyncedDatabaseTable
}

func (s *SyncedDatabaseTable) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Database.GetSyncedDatabaseTable(ctx, database.GetSyncedDatabaseTableRequest{Name: name})
	if err != nil {
		log.Debugf(ctx, "synced database table %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (s *SyncedDatabaseTable) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "synced_database_table",
		PluralName:    "synced_database_tables",
		SingularTitle: "Synced database table",
		PluralTitle:   "Synced database tables",
	}
}

func (s *SyncedDatabaseTable) GetName() string {
	// A synced table's id is its three-part name (catalog.schema.table), so the
	// id IS the name. Prefer the post-deploy id so bundle summary renders the
	// resolved name even when the configured name still has unresolved
	// cross-resource references like ${resources.X.Y.Z}. Mirrors
	// PostgresSyncedTable.GetName.
	if s.ID != "" {
		return s.ID
	}
	return s.Name
}

func (s *SyncedDatabaseTable) GetURL() string {
	return s.URL
}

func (s *SyncedDatabaseTable) InitializeURL(baseURL url.URL) {
	// Bail if the name isn't a fully resolved three-part identifier; an
	// unresolved ${...} reference would otherwise produce a misleading URL.
	name := s.GetName()
	if strings.Count(name, ".") != 2 {
		return
	}
	s.URL = workspaceurls.ResourceURL(baseURL, "synced_database_tables", name)
}
