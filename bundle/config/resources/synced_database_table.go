package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"

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
	return s.Name
}

func (s *SyncedDatabaseTable) GetURL() string {
	return s.URL
}

func (s *SyncedDatabaseTable) InitializeURL(baseURL url.URL) {
	if s.Name == "" {
		return
	}
	baseURL.Path = "explore/data/" + s.Name
	s.URL = baseURL.String()
}
